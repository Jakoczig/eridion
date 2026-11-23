package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"
)

// ExternalFacts contains normalized allowlisted facts, never provider responses or secrets.
type ExternalFacts struct {
	Provider, Entity, EntityID, FID string
	Observed, Fetched, Expires      time.Time
	IdentityVerified                bool
	Fields                          map[string]any
}
type FactEvidence struct {
	Source    string          `json:"source"`
	Reference string          `json:"reference"`
	Observed  string          `json:"observedAt"`
	Fetched   string          `json:"fetchedAt"`
	Stale     bool            `json:"stale"`
	Value     json.RawMessage `json:"value"`
	// Authority is the D0 tier ("frontier", "journal" or "third-party");
	// Eligible is false for expired or invalid evidence kept as a fallback.
	Authority string `json:"authority"`
	Eligible  bool   `json:"eligible"`
}
type EffectiveFact struct {
	Field string `json:"field"`
	FactEvidence
	Alternatives []FactEvidence `json:"alternatives"`
}

// currentFactEvidence selects, for every field and source, the single
// chronologically best observation. Selection is per source; cross-source
// authority is applied afterwards by EffectiveFacts. Each lookup uses the
// provenance_observed index and stays bounded regardless of history size.
func (s *Store) currentFactEvidence(ctx context.Context, fid, entity string, now time.Time) (map[string][]FactEvidence, error) {
	var owner any
	if fid != "" {
		owner = fid
	}
	cutoff := observationCutoff(now)
	values := map[string][]FactEvidence{}
	// Enumerate only the field/source groups that have evidence (a covering
	// index scan), then select each group's best observation with indexed
	// lookups, so the number of queries tracks the evidence actually held.
	groupRows, err := s.reader.QueryContext(ctx, "SELECT DISTINCT field,source FROM field_provenance WHERE commander_fid IS ? AND entity=? LIMIT 256", owner, entity)
	if err != nil {
		return nil, err
	}
	type group struct{ field, source string }
	var groups []group
	for groupRows.Next() {
		var g group
		if err = groupRows.Scan(&g.field, &g.source); err != nil {
			groupRows.Close()
			return nil, err
		}
		groups = append(groups, g)
	}
	err = groupRows.Err()
	groupRows.Close()
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		field, source := g.field, g.source
		{
			var key sql.NullString
			if err := s.reader.QueryRowContext(ctx, "SELECT max(observed_key) FROM field_provenance WHERE commander_fid IS ? AND entity=? AND field=? AND source=? AND observed_key<>'' AND observed_key<=?", owner, entity, field, source, cutoff).Scan(&key); err != nil {
				return nil, err
			}
			if !key.Valid {
				// No plausible observation: fall back to the latest future or
				// unknown-time evidence, which stays labelled by its observedAt.
				if err := s.reader.QueryRowContext(ctx, "SELECT max(observed_key) FROM field_provenance WHERE commander_fid IS ? AND entity=? AND field=? AND source=?", owner, entity, field, source).Scan(&key); err != nil {
					return nil, err
				}
				if !key.Valid {
					continue
				}
			}
			rows, err := s.reader.QueryContext(ctx, `SELECT p.source_ref,coalesce(p.observed_at,''),coalesce(p.fetched_at,''),p.value_json,coalesce(e.expires_at,''),coalesce(j.source_id,e.fetched_at,''),coalesce(j.byte_offset,0),coalesce(j.line_hash,e.content_hash,'') FROM field_provenance p LEFT JOIN journal_events j ON p.source='journal' AND j.event_id=cast(p.source_ref AS INTEGER) LEFT JOIN external_snapshots e ON p.source<>'journal' AND e.id=cast(p.source_ref AS INTEGER) AND p.source=e.provider WHERE p.commander_fid IS ? AND p.entity=? AND p.field=? AND p.source=? AND p.observed_key=? ORDER BY p.id DESC LIMIT 256`, owner, entity, field, source, key.String)
			if err != nil {
				return nil, err
			}
			var best FactEvidence
			var bestObs observation
			var bestExpires string
			found := false
			for rows.Next() {
				var v FactEvidence
				var raw, expires string
				o := observation{Key: key.String}
				if err = rows.Scan(&v.Reference, &v.Observed, &v.Fetched, &raw, &expires, &o.Source, &o.Offset, &o.Hash); err != nil {
					rows.Close()
					return nil, err
				}
				v.Source = source
				v.Value = json.RawMessage(raw)
				if source != "journal" {
					// External tie-break: compare the fetch instant, not raw
					// RFC3339 text ("…:00Z" would sort after "…:00.5Z").
					if k := observationKey(o.Source); k != "" {
						o.Source = k
					}
				}
				if !found || betterObservation(o, bestObs, cutoff) {
					best, bestObs, bestExpires, found = v, o, expires, true
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return nil, err
			}
			if !found {
				continue
			}
			expiry, e := time.Parse(time.RFC3339Nano, bestExpires)
			best.Stale = e != nil || !now.Before(expiry)
			if source == "journal" {
				at, ok := validObservedAt(best.Observed, now)
				best.Stale = !ok || now.Sub(at) > 5*time.Minute
			}
			values[field] = append(values[field], best)
		}
	}
	return values, nil
}

var allowedFacts = map[string]bool{"name": true, "credits": true, "debt": true, "ranks": true, "squadron": true, "ships": true, "ship": true, "carrier": true, "inventory": true, "communityGoals": true, "market": true, "shipyard": true, "system": true, "station": true, "coordinates": true}

// Fact inspection is not a calculation API. Preserve every journal number as
// its original decimal text, including identifiers nested in goals/squadrons.
// The typed reducers separately retain numeric physical measurements.
func exactFactValue(value any) any {
	switch v := value.(type) {
	case json.Number:
		return string(v)
	case map[string]any:
		out := map[string]any{}
		for k, x := range v {
			out[k] = exactFactValue(x)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = exactFactValue(x)
		}
		return out
	default:
		return value
	}
}

func (s *Store) SaveExternal(ctx context.Context, x ExternalFacts) error {
	if (x.Provider != "frontier" && x.Provider != "inara") || x.Entity == "" || len(x.Entity) > 64 || len(x.EntityID) > 128 || len(x.FID) > 128 || !x.IdentityVerified || x.Fetched.IsZero() || !x.Expires.After(x.Fetched) || len(x.Fields) > 32 {
		return errors.New("Verified bounded external identity and freshness required")
	}
	for k := range x.Fields {
		if !allowedFacts[k] {
			return errors.New("Unsupported external field")
		}
	}
	raw, err := json.Marshal(x.Fields)
	if err != nil || len(raw) > 256<<10 {
		return errors.New("External facts exceed limit")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stamp := func(v time.Time) string {
		if v.IsZero() {
			return ""
		}
		return v.UTC().Format(time.RFC3339Nano)
	}
	observed := stamp(x.Observed)
	result, err := tx.ExecContext(ctx, "INSERT INTO external_snapshots(provider,entity_type,entity_id,commander_fid,source_observed_at,fetched_at,expires_at,identity_verified,content_hash,normalized_fields) VALUES(?,?,?,?,?,?,?,?,?,?)", x.Provider, x.Entity, x.EntityID, null(x.FID), observed, stamp(x.Fetched), stamp(x.Expires), true, digest(raw), string(raw))
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	for k, v := range x.Fields {
		b, _ := json.Marshal(v)
		if _, err = tx.ExecContext(ctx, "INSERT INTO field_provenance(entity,field,commander_fid,source,source_ref,observed_at,fetched_at,confidence,stale,value_json,observed_key) VALUES(?,?,?,?,?,?,?,?,?,?,?)", x.Entity, k, null(x.FID), x.Provider, strconv.FormatInt(id, 10), observed, stamp(x.Fetched), "verified identity; remote snapshot", false, string(b), observationKey(observed)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) EffectiveFacts(ctx context.Context, fid, entity string, now time.Time) ([]EffectiveFact, error) {
	if len(fid) > 128 || len(entity) > 64 {
		return nil, errors.New("Invalid fact query")
	}
	values, err := s.currentFactEvidence(ctx, fid, entity, now)
	if err != nil {
		return nil, err
	}
	out := []EffectiveFact{}
	for field, a := range values {
		// D0 authority: fresh Frontier, then journal, then one equal
		// third-party tier. Competing evidence is moved, never discarded.
		a = ResolveEvidence(a, now)
		out = append(out, EffectiveFact{Field: field, FactEvidence: a[0], Alternatives: a[1:]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Field < out[j].Field })
	return out, nil
}
func (s *Store) NeedsInara(ctx context.Context, fid, entity string, fields []string, now time.Time) bool {
	facts, err := s.EffectiveFacts(ctx, fid, entity, now)
	if err != nil {
		return false
	}
	for _, name := range fields {
		found := false
		for _, f := range facts {
			if f.Field == name && !f.Stale {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}
	return false
}
