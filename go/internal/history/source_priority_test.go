package history

import (
	"context"
	"encoding/json"
	"github.com/Jakoczig/eridion/go/internal/journal"
	"strconv"
	"testing"
	"time"
)

// sourcePriorityStore opens a store holding one journal LoadGame (credits 200)
// and Rank/Progress events for SYNTHETIC-A at journalAt.
func sourcePriorityStore(t *testing.T, journalAt time.Time) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	src := journal.Source{ID: "priority-fixture", Identity: "synthetic"}
	cp, err := s.Resume(src)
	if err != nil {
		t.Fatal(err)
	}
	stamp := journalAt.UTC().Format(time.RFC3339)
	lines := []string{
		`{"event":"LoadGame","timestamp":"` + stamp + `","FID":"SYNTHETIC-A","Commander":"Fixture","Credits":200}`,
		`{"event":"Rank","timestamp":"` + stamp + `","Combat":5,"Trade":3}`,
		`{"event":"Progress","timestamp":"` + stamp + `","Combat":55,"Trade":40}`,
	}
	records := []journal.Record{}
	for i, line := range lines {
		e, err := journal.ParseLine([]byte(line), journal.SourcePosition{FileID: src.ID, Sequence: uint64(i + 1), Offset: int64(i * 200), Length: 200})
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, journal.Record{Event: e, Phase: journal.Live})
	}
	cp.Offset, cp.Sequence = int64(len(lines)*200), uint64(len(lines))
	if err = s.Commit(context.Background(), cp, records, nil); err != nil {
		t.Fatal(err)
	}
	return s
}

func effective(t *testing.T, s *Store, now time.Time, field string) EffectiveFact {
	t.Helper()
	facts, err := s.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range facts {
		if f.Field == field {
			return f
		}
	}
	t.Fatalf("%s missing from %+v", field, facts)
	return EffectiveFact{}
}

func saveFacts(t *testing.T, s *Store, provider string, observed, fetched time.Time, fields map[string]any) {
	t.Helper()
	if err := s.SaveExternal(context.Background(), ExternalFacts{Provider: provider, Entity: "commander", EntityID: "SYNTHETIC-A", FID: "SYNTHETIC-A", IdentityVerified: true, Observed: observed, Fetched: fetched, Expires: fetched.Add(5 * time.Minute), Fields: fields}); err != nil {
		t.Fatal(err)
	}
}

// Confirmed D0: fresh validated Frontier > valid journal > Inara and every
// other provider at one equal tier.
func TestAuditSourcePriority(t *testing.T) {
	journalAt := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	now := journalAt.Add(time.Hour)
	for _, tc := range []struct {
		name      string
		observed  time.Time
		fetched   time.Time
		frontier  string
		inara     string
		want      string
		wantValue string
	}{
		{"fresh frontier with unknown observedAt beats journal", time.Time{}, now, "100", "", "frontier", `"100"`},
		{"fresh frontier older than journal still wins", journalAt.Add(-time.Hour), now, "100", "", "frontier", `"100"`},
		{"valid zero from fresh frontier is not missing", time.Time{}, now, "0", "", "frontier", `"0"`},
		{"expired frontier falls back to journal", time.Time{}, now.Add(-10 * time.Minute), "100", "", "journal", `"200"`},
		{"future frontier observation is invalid", now.Add(time.Hour), now, "100", "", "journal", `"200"`},
		{"journal beats fresh inara", time.Time{}, now.Add(-10 * time.Minute), "100", "300", "journal", `"200"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := sourcePriorityStore(t, journalAt)
			saveFacts(t, s, "frontier", tc.observed, tc.fetched, map[string]any{"credits": tc.frontier})
			if tc.inara != "" {
				saveFacts(t, s, "inara", time.Time{}, now, map[string]any{"credits": tc.inara})
			}
			// Another commander's evidence never takes part.
			if err := s.SaveExternal(context.Background(), ExternalFacts{Provider: "frontier", Entity: "commander", EntityID: "SYNTHETIC-B", FID: "SYNTHETIC-B", IdentityVerified: true, Fetched: now, Expires: now.Add(time.Minute), Fields: map[string]any{"credits": "999"}}); err != nil {
				t.Fatal(err)
			}
			f := effective(t, s, now, "credits")
			if f.Source != tc.want || string(f.Value) != tc.wantValue {
				t.Fatalf("effective credits = %s %s, want %s %s", f.Source, f.Value, tc.want, tc.wantValue)
			}
			if len(f.Alternatives) == 0 {
				t.Fatal("competing evidence discarded", f)
			}
			if f.Source == "frontier" && f.Observed != "" && tc.observed.IsZero() {
				t.Fatal("fetch time fabricated as observation", f)
			}
		})
	}

	t.Run("no journal: equal tier containing inara is used when frontier expired", func(t *testing.T) {
		s, err := Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		saveFacts(t, s, "frontier", time.Time{}, now.Add(-time.Hour), map[string]any{"debt": "5"})
		saveFacts(t, s, "inara", time.Time{}, now, map[string]any{"debt": "7"})
		f := effective(t, s, now, "debt")
		if f.Source != "inara" || string(f.Value) != `"7"` || f.Stale {
			t.Fatalf("effective debt = %+v, want fresh inara", f)
		}
		if f.Authority != "third-party" || !f.Eligible {
			t.Fatalf("authority metadata = %q eligible=%v", f.Authority, f.Eligible)
		}
	})
}

// Inara has no hidden preference over another provider in the third tier, and
// the result does not depend on input order.
func TestAuditSourcePriorityEqualTier(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	older := FactEvidence{Source: "inara", Reference: "1", Observed: now.Add(-2 * time.Hour).Format(time.RFC3339), Fetched: now.Format(time.RFC3339Nano), Value: json.RawMessage(`"inara"`)}
	newer := FactEvidence{Source: "edsm", Reference: "2", Observed: now.Add(-time.Hour).Format(time.RFC3339), Fetched: now.Format(time.RFC3339Nano), Value: json.RawMessage(`"edsm"`)}
	for _, order := range [][]FactEvidence{{older, newer}, {newer, older}} {
		got := ResolveEvidence(append([]FactEvidence{}, order...), now)
		if got[0].Source != "edsm" {
			t.Fatalf("newer equal-tier observation lost: %+v", got)
		}
	}
	// Swap names: the newer observation still wins, so there is no name rule.
	older.Source, newer.Source = "edsm", "inara"
	if got := ResolveEvidence([]FactEvidence{older, newer}, now); got[0].Source != "inara" || string(got[0].Value) != `"edsm"` {
		t.Fatalf("name preference detected: %+v", got)
	}
	// Equal observation: later fetch, then later stored reference, decides.
	a := FactEvidence{Source: "inara", Reference: "5", Fetched: now.Add(-time.Minute).Format(time.RFC3339Nano)}
	b := FactEvidence{Source: "spansh", Reference: "4", Fetched: now.Format(time.RFC3339Nano)}
	for _, order := range [][]FactEvidence{{a, b}, {b, a}} {
		if got := ResolveEvidence(append([]FactEvidence{}, order...), now); got[0].Source != "spansh" {
			t.Fatalf("later fetch lost: %+v", got)
		}
	}
	b.Fetched = a.Fetched
	for _, order := range [][]FactEvidence{{a, b}, {b, a}} {
		if got := ResolveEvidence(append([]FactEvidence{}, order...), now); got[0].Reference != "5" {
			t.Fatalf("tie not deterministic: %+v", got)
		}
	}
	// Stale third-party evidence ranks below fresh third-party evidence.
	stale := FactEvidence{Source: "edsm", Reference: "9", Stale: true, Observed: now.Format(time.RFC3339)}
	if got := ResolveEvidence([]FactEvidence{stale, a}, now); got[0].Reference != "5" || got[1].Eligible {
		t.Fatalf("stale evidence outranked fresh: %+v", got)
	}
}

func rankByCategory(ranks []ResolvedRank, category string) *ResolvedRank {
	for i := range ranks {
		if ranks[i].Category == category {
			return &ranks[i]
		}
	}
	return nil
}

// Frontier rank 8 cannot inherit progress that belongs to journal rank 5;
// categories missing from Frontier fall back independently.
func TestAuditRankCoherence(t *testing.T) {
	journalAt := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	now := journalAt.Add(time.Hour)

	s := sourcePriorityStore(t, journalAt)
	saveFacts(t, s, "frontier", time.Time{}, now, map[string]any{"ranks": map[string]string{"combat": "8"}})
	facts, err := s.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", now)
	if err != nil {
		t.Fatal(err)
	}
	ranks := ResolveRanks(facts)
	combat, trade := rankByCategory(ranks, "combat"), rankByCategory(ranks, "trade")
	if combat == nil || combat.Rank != "8" || combat.Source != "frontier" || combat.Progress != nil {
		t.Fatalf("combat = %+v, want Frontier 8 with unknown progress", combat)
	}
	if trade == nil || trade.Rank != "3" || trade.Source != "journal" || trade.Progress == nil || *trade.Progress != "40" {
		t.Fatalf("trade = %+v, want journal 3 with progress 40", trade)
	}

	// A Frontier level equal to the journal level keeps compatible progress.
	s = sourcePriorityStore(t, journalAt)
	saveFacts(t, s, "frontier", time.Time{}, now, map[string]any{"ranks": map[string]string{"combat": "5"}})
	facts, _ = s.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", now)
	combat = rankByCategory(ResolveRanks(facts), "combat")
	if combat == nil || combat.Source != "frontier" || combat.Progress == nil || *combat.Progress != "55" || combat.ProgressSource != "journal" {
		t.Fatalf("compatible progress lost: %+v", combat)
	}

	// Expired Frontier falls back to the journal level and its progress.
	s = sourcePriorityStore(t, journalAt)
	saveFacts(t, s, "frontier", time.Time{}, now.Add(-time.Hour), map[string]any{"ranks": map[string]string{"combat": "8"}})
	facts, _ = s.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", now)
	combat = rankByCategory(ResolveRanks(facts), "combat")
	if combat == nil || combat.Rank != "5" || combat.Source != "journal" || combat.Progress == nil || *combat.Progress != "55" {
		t.Fatalf("expired fallback = %+v", combat)
	}

	// Inara's list shape is normalized to the same categories.
	s, err = Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	saveFacts(t, s, "inara", time.Time{}, now, map[string]any{"ranks": []map[string]string{{"name": "exploration", "value": "4"}}})
	facts, _ = s.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", now)
	explore := rankByCategory(ResolveRanks(facts), "explore")
	if explore == nil || explore.Rank != "4" || explore.Source != "inara" {
		t.Fatalf("inara rank = %+v", explore)
	}
}

func TestAuditCurrentShipAuthority(t *testing.T) {
	ship := json.RawMessage(`{"id":"9007199254740993","model":"anaconda","name":"FIXTURE ONE","systemName":"Sol","currentReference":true}`)
	fresh := []EffectiveFact{{Field: "ship", FactEvidence: FactEvidence{Source: "frontier", Eligible: true, Value: ship}}}
	got, ok := ResolveCurrentShip(fresh)
	if !ok || got.ID != "9007199254740993" || got.Name != "FIXTURE ONE" || got.Model != "anaconda" || got.SystemName != "Sol" {
		t.Fatalf("fresh Frontier current ship = %+v %v", got, ok)
	}
	for name, facts := range map[string][]EffectiveFact{
		"expired":     {{Field: "ship", FactEvidence: FactEvidence{Source: "frontier", Eligible: false, Stale: true, Value: ship}}},
		"not current": {{Field: "ship", FactEvidence: FactEvidence{Source: "frontier", Eligible: true, Value: json.RawMessage(`{"id":"1","currentReference":false}`)}}},
		"third party": {{Field: "ship", FactEvidence: FactEvidence{Source: "inara", Eligible: true, Value: ship}}},
	} {
		if _, ok := ResolveCurrentShip(facts); ok {
			t.Fatalf("%s evidence replaced the journal ship", name)
		}
	}
}

// Journal ranks known only to the live reducer (startup prelude events that
// precede LoadGame never reach commander provenance) still resolve at the
// journal tier; Frontier keeps precedence and progress stays level-coherent.
func TestAuditMergeJournalRanks(t *testing.T) {
	three, four, twentySeven := "3", "4", "27"
	journal := []JournalRank{{Category: "Combat", Rank: &three, Progress: &twentySeven}, {Category: "Trade", Rank: &four}}
	got := MergeJournalRanks(nil, journal)
	combat, trade := rankByCategory(got, "combat"), rankByCategory(got, "trade")
	if combat == nil || combat.Rank != "3" || combat.Source != "journal" || combat.Progress == nil || *combat.Progress != "27" {
		t.Fatalf("journal-only combat = %+v", combat)
	}
	if trade == nil || trade.Progress != nil {
		t.Fatalf("journal-only trade = %+v", trade)
	}
	frontier := []ResolvedRank{{Category: "combat", Rank: "8", Source: "frontier", Authority: "frontier"}}
	if c := rankByCategory(MergeJournalRanks(frontier, journal), "combat"); c.Rank != "8" || c.Progress != nil {
		t.Fatalf("Frontier 8 inherited journal 3's progress: %+v", c)
	}
	same := []ResolvedRank{{Category: "combat", Rank: "3", Source: "frontier", Authority: "frontier"}}
	if c := rankByCategory(MergeJournalRanks(same, journal), "combat"); c.Progress == nil || *c.Progress != "27" || c.ProgressSource != "journal" {
		t.Fatalf("same-level progress lost: %+v", c)
	}
}

// Audit F1: journal ranks known only to the live reducer are valid journal
// evidence. They beat the equal third-party tier (Inara) and expired or
// otherwise ineligible Frontier values; only eligible Frontier keeps its level.
// The replaced value stays visible as a labelled alternative.
func TestAuditMergeJournalRanksFollowsAuthority(t *testing.T) {
	three, twelve := "3", "12"
	journal := []JournalRank{{Category: "Combat", Rank: &three, Progress: &twelve}}
	for name, resolved := range map[string]ResolvedRank{
		"inara":            {Category: "combat", Rank: "5", Source: "inara", Authority: "third-party"},
		"expired frontier": {Category: "combat", Rank: "8", Source: "frontier", Authority: "frontier", Stale: true},
	} {
		c := rankByCategory(MergeJournalRanks([]ResolvedRank{resolved}, journal), "combat")
		if c == nil || c.Rank != "3" || c.Source != "journal" || c.Authority != "journal" || c.Stale || c.Progress == nil || *c.Progress != "12" {
			t.Fatalf("%s kept over valid journal evidence: %+v", name, c)
		}
		if len(c.Alternatives) != 1 || c.Alternatives[0].Source != resolved.Source || c.Alternatives[0].Rank != resolved.Rank {
			t.Fatalf("%s provenance not retained: %+v", name, c.Alternatives)
		}
	}
	// An older witnessed journal level in provenance yields to the live
	// reducer, which already applies within-source chronology.
	older := []ResolvedRank{{Category: "combat", Rank: "2", Source: "journal", Authority: "journal", ObservedAt: "2026-01-01T00:00:00Z"}}
	if c := rankByCategory(MergeJournalRanks(older, journal), "combat"); c.Rank != "3" || c.Source != "journal" {
		t.Fatalf("live journal level lost to older provenance: %+v", c)
	}
	// Eligible Frontier keeps precedence.
	fresh := []ResolvedRank{{Category: "combat", Rank: "8", Source: "frontier", Authority: "frontier"}}
	if c := rankByCategory(MergeJournalRanks(fresh, journal), "combat"); c.Rank != "8" || c.Source != "frontier" || c.Progress != nil {
		t.Fatalf("fresh Frontier replaced: %+v", c)
	}
}

// G3H review finding 1: Frontier/Inara snapshots carry no observation time.
// The latest fetch must stay effective however many snapshots a commander
// has accumulated, and fetch times compare as instants, not as raw text.
func TestAuditLatestSnapshotWinsBeyondWindow(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var last time.Time
	for i := 0; i < 300; i++ {
		last = start.Add(time.Duration(i) * time.Hour)
		saveFacts(t, s, "frontier", time.Time{}, last, map[string]any{"credits": strconv.Itoa(i)})
	}
	facts, err := s.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", last.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	var credits *EffectiveFact
	for i := range facts {
		if facts[i].Field == "credits" {
			credits = &facts[i]
		}
	}
	if credits == nil || string(credits.Value) != `"299"` || !credits.Eligible || credits.Stale {
		t.Fatalf("latest of 300 snapshots not effective: %+v", credits)
	}

	// Same second, fractional later fetch: "…:00Z" must not sort after "…:00.5Z".
	s2, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	at := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	saveFacts(t, s2, "inara", time.Time{}, at, map[string]any{"credits": "1"})
	saveFacts(t, s2, "inara", time.Time{}, at.Add(500*time.Millisecond), map[string]any{"credits": "2"})
	facts, _ = s2.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", at.Add(time.Minute))
	for _, f := range facts {
		if f.Field == "credits" && string(f.Value) != `"2"` {
			t.Fatalf("earlier fetch won the raw-text tie-break: %s", f.Value)
		}
	}
}
