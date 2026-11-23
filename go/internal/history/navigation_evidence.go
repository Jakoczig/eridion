package history

import (
	"database/sql"
	"encoding/json"
	"time"
)

type NavigationEvidence struct {
	Boost     *float64 `json:"boost"`
	BoostAt   string   `json:"boostAt"`
	DamageAt  string   `json:"damageAt"`
	Scooped   *float64 `json:"scooped"`
	FuelTotal *float64 `json:"fuelTotal"`
	ScoopAt   string   `json:"scoopAt"`
}

func (s *Store) FieldNavigationEvidence(fid string) (NavigationEvidence, error) {
	return s.fieldNavigationEvidenceAt(fid, time.Now())
}

// latestEventOfType returns the chronologically best retained event of one
// type: plausible observations first, then deterministic immutable ties.
// Ingestion order (event_id) never decides which evidence is current.
func (s *Store) latestEventOfType(fid, kind string, now time.Time) (raw, at string, err error) {
	const order = " ORDER BY event_utc DESC,source_id DESC,byte_offset DESC,line_hash DESC LIMIT 1"
	err = s.reader.QueryRow("SELECT raw_relevant_json,coalesce(event_utc,'') FROM journal_events WHERE commander_fid=? AND type=? AND event_utc<>'' AND event_utc<=?"+order, fid, kind, observationCutoff(now)).Scan(&raw, &at)
	if err == sql.ErrNoRows {
		err = s.reader.QueryRow("SELECT raw_relevant_json,coalesce(event_utc,'') FROM journal_events WHERE commander_fid=? AND type=?"+order, fid, kind).Scan(&raw, &at)
	}
	return raw, at, err
}

func (s *Store) fieldNavigationEvidenceAt(fid string, now time.Time) (NavigationEvidence, error) {
	out := NavigationEvidence{}
	for _, kind := range []string{"JetConeBoost", "JetConeDamage", "FuelScoop"} {
		raw, at, e := s.latestEventOfType(fid, kind, now)
		if e == sql.ErrNoRows {
			continue
		}
		if e != nil {
			return out, e
		}
		v := object([]byte(raw))
		get := func(k string) *float64 {
			if n, ok := v[k].(json.Number); ok {
				if x, e := n.Float64(); e == nil && x >= 0 {
					return &x
				}
			}
			return nil
		}
		switch kind {
		case "JetConeBoost":
			out.Boost = get("BoostValue")
			out.BoostAt = at
		case "JetConeDamage":
			out.DamageAt = at
		case "FuelScoop":
			out.Scooped = get("Scooped")
			out.FuelTotal = get("Total")
			out.ScoopAt = at
		}
	}
	return out, nil
}
