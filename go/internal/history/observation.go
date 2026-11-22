package history

import (
	"database/sql"
	"strings"
	"time"
)

// observationLayout is the fixed-width UTC form used for every comparable
// observation key. Fixed width makes lexical order equal instant order,
// including fractional seconds.
const observationLayout = "2006-01-02T15:04:05.000000000Z"

// futureTolerance is the existing allowance for clock skew between the game
// and this host. Evidence beyond it remains stored and queryable but cannot
// become the current value until the clock reaches it.
const futureTolerance = 5 * time.Second

func parseObservation(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil || t.IsZero() || t.Year() < 1 || t.Year() > 9999 {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// validObservedAt reports whether raw is a plausible observation instant at
// now: parseable, non-zero and not beyond the future tolerance.
func validObservedAt(raw string, now time.Time) (time.Time, bool) {
	t, ok := parseObservation(raw)
	if !ok || t.After(now.Add(futureTolerance)) {
		return time.Time{}, false
	}
	return t, true
}

// observationKey normalizes any RFC 3339 observation to its fixed-width UTC
// key. Unknown or invalid observation times yield "" and stay distinct.
func observationKey(raw string) string {
	t, ok := parseObservation(raw)
	if !ok {
		return ""
	}
	return t.Format(observationLayout)
}

func observationCutoff(now time.Time) string {
	return now.Add(futureTolerance).UTC().Format(observationLayout)
}

// observation identifies one piece of evidence within a single source for
// chronological selection. Ties never use ingestion IDs: they use immutable
// source identity, byte offset and content hash, so import order cannot
// change the result.
type observation struct {
	Key    string
	Source string
	Offset int64
	Hash   string
}

// betterObservation reports whether a should be preferred over b at the
// given cutoff: plausible evidence wins over unknown or future evidence,
// then later instants, then the deterministic immutable tie-breakers.
func betterObservation(a, b observation, cutoff string) bool {
	pa, pb := a.Key != "" && a.Key <= cutoff, b.Key != "" && b.Key <= cutoff
	if pa != pb {
		return pa
	}
	if a.Key != b.Key {
		return a.Key > b.Key
	}
	if a.Source != b.Source {
		return a.Source > b.Source
	}
	if a.Offset != b.Offset {
		return a.Offset > b.Offset
	}
	return a.Hash > b.Hash
}

func eventObservation(q interface {
	QueryRow(string, ...any) *sql.Row
}, eventID int64) (observation, error) {
	var o observation
	var stamp sql.NullString
	err := q.QueryRow("SELECT event_utc,source_id,byte_offset,line_hash FROM journal_events WHERE event_id=?", eventID).Scan(&stamp, &o.Source, &o.Offset, &o.Hash)
	o.Key = observationKey(stamp.String)
	return o, err
}

// chronologySchema adds comparable observation keys to derived provenance and
// an index for observation-ordered journal lookups. Its Go backfill repairs
// already-derived current values from retained evidence.
const chronologySchema = `ALTER TABLE field_provenance ADD COLUMN observed_key TEXT NOT NULL DEFAULT '';
CREATE INDEX provenance_observed ON field_provenance(commander_fid,entity,field,source,observed_key);
CREATE INDEX events_commander_type_time ON journal_events(commander_fid,type,event_utc);`

// backfillChronology runs inside the migration transaction; a failure rolls
// the migration back and leaves the prior database usable.
func backfillChronology(tx *sql.Tx) error {
	rows, err := tx.Query("SELECT id,coalesce(observed_at,'') FROM field_provenance")
	if err != nil {
		return err
	}
	keys := map[int64]string{}
	for rows.Next() {
		var id int64
		var raw string
		if err = rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return err
		}
		if k := observationKey(raw); k != "" {
			keys[id] = k
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for id, k := range keys {
		if _, err = tx.Exec("UPDATE field_provenance SET observed_key=? WHERE id=?", k, id); err != nil {
			return err
		}
	}
	// Progress percentages were previously stored as rank levels. Reclassify
	// them by their retained source event; the evidence row itself is kept.
	if _, err = tx.Exec(`UPDATE field_provenance SET field='rankProgress' WHERE source='journal' AND field='ranks' AND cast(source_ref AS INTEGER) IN (SELECT event_id FROM journal_events WHERE type='Progress')`); err != nil {
		return err
	}
	return repairColonisationProjects(tx)
}

// repairColonisationProjects recomputes each project's current row from the
// chronologically best retained depot snapshot.
func repairColonisationProjects(tx *sql.Tx) error {
	rows, err := tx.Query("SELECT event_id,commander_fid,coalesce(event_utc,''),source_id,byte_offset,line_hash,coalesce(system_address,''),raw_relevant_json FROM journal_events WHERE type='ColonisationConstructionDepot' AND commander_fid IS NOT NULL")
	if err != nil {
		return err
	}
	type best struct {
		id      int64
		fid     string
		obs     observation
		stamp   string
		address string
		v       map[string]any
	}
	cutoff := observationCutoff(time.Now())
	chosen := map[[2]string]best{}
	for rows.Next() {
		var b best
		var raw string
		if err = rows.Scan(&b.id, &b.fid, &b.stamp, &b.obs.Source, &b.obs.Offset, &b.obs.Hash, &b.address, &raw); err != nil {
			rows.Close()
			return err
		}
		b.obs.Key = observationKey(b.stamp)
		b.v = object([]byte(raw))
		market := identifier(b.v["MarketID"])
		if market == "" {
			continue
		}
		k := [2]string{b.fid, market}
		if old, ok := chosen[k]; !ok || betterObservation(b.obs, old.obs, cutoff) {
			chosen[k] = b
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for k, b := range chosen {
		if _, err = tx.Exec("UPDATE colonisation_projects SET system_address=?,construction_progress=?,complete=?,failed=?,observed_at=?,source_event_id=? WHERE commander_fid=? AND market_id=?", null(b.address), number(b.v["ConstructionProgress"]), boolean(b.v["ConstructionComplete"]), boolean(b.v["ConstructionFailed"]), b.stamp, b.id, k[0], k[1]); err != nil {
			return err
		}
	}
	return nil
}

// upsertColonisationProject replaces the current project only when the new
// depot snapshot is chronologically better than the stored one.
func upsertColonisationProject(tx *sql.Tx, fid string, eventID int64, address, stamp string, v map[string]any) error {
	market := identifier(v["MarketID"])
	var currentID sql.NullInt64
	err := tx.QueryRow("SELECT source_event_id FROM colonisation_projects WHERE commander_fid=? AND market_id=?", fid, market).Scan(&currentID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil && currentID.Valid {
		candidate, e1 := eventObservation(tx, eventID)
		current, e2 := eventObservation(tx, currentID.Int64)
		if e1 != nil {
			return e1
		}
		if e2 == nil && !betterObservation(candidate, current, observationCutoff(time.Now())) {
			return nil
		}
	}
	_, err = tx.Exec("INSERT INTO colonisation_projects VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(commander_fid,market_id) DO UPDATE SET system_address=excluded.system_address,construction_progress=excluded.construction_progress,complete=excluded.complete,failed=excluded.failed,observed_at=excluded.observed_at,source_event_id=excluded.source_event_id", fid, market, null(address), number(v["ConstructionProgress"]), boolean(v["ConstructionComplete"]), boolean(v["ConstructionFailed"]), stamp, eventID)
	return err
}
