package gamestate

import (
	"context"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/history"
)

func overviewRank(t *testing.T, s *Service, category string) history.ResolvedRank {
	t.Helper()
	overview, err := s.CommanderOverview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range overview["ranks"].([]history.ResolvedRank) {
		if r.Category == category {
			return r
		}
	}
	t.Fatalf("no %s rank in %v", category, overview["ranks"])
	return history.ResolvedRank{}
}

// Audit F1 end to end: as in real journals, Rank and Progress precede
// LoadGame, so the journal levels never reach commander provenance. They must
// still win over Inara and over an expired Frontier snapshot; only a fresh
// Frontier snapshot replaces them.
func TestAuditJournalRanksBeforeLoadGameFollowAuthority(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	now := time.Now().UTC()
	stamp := func(d time.Duration) string { return now.Add(d).Format(time.RFC3339) }
	appendEvent(t, dir, `{"event":"Fileheader","timestamp":"`+stamp(-time.Minute)+`","part":1,"gameversion":"4.1.0.0","build":"fixture"}`)
	appendEvent(t, dir, `{"event":"Commander","timestamp":"`+stamp(-time.Minute)+`","FID":"SYNTHETIC-A","Name":"Fixture"}`)
	appendEvent(t, dir, `{"event":"Rank","timestamp":"`+stamp(-time.Minute)+`","Combat":3,"Trade":4}`)
	appendEvent(t, dir, `{"event":"Progress","timestamp":"`+stamp(-time.Minute)+`","Combat":27,"Trade":10}`)
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"`+stamp(-time.Minute)+`","FID":"SYNTHETIC-A","Commander":"Fixture","Ship":"anaconda","ShipID":7}`)
	writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
	s, _, stop := recordingService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 5 })
	store, _ := s.History()
	save := func(provider string, fetched, expires time.Time, ranks any) {
		t.Helper()
		if err := store.SaveExternal(context.Background(), history.ExternalFacts{Provider: provider, Entity: "commander", EntityID: "SYNTHETIC-A", FID: "SYNTHETIC-A", IdentityVerified: true, Fetched: fetched, Expires: expires, Fields: map[string]any{"ranks": ranks}}); err != nil {
			t.Fatal(err)
		}
	}
	check := func(label, want, source string, progress *string) {
		t.Helper()
		r := overviewRank(t, s, "combat")
		if r.Rank != want || r.Source != source || (progress == nil) != (r.Progress == nil) || (progress != nil && *r.Progress != *progress) {
			t.Fatalf("%s: combat = %+v", label, r)
		}
	}
	p27 := "27"
	check("journal only", "3", "journal", &p27)

	save("inara", now, now.Add(time.Hour), []map[string]string{{"name": "combat", "value": "5"}})
	check("journal vs Inara", "3", "journal", &p27)

	save("frontier", now.Add(-10*time.Minute), now.Add(-5*time.Minute), map[string]string{"combat": "8"})
	check("journal vs expired Frontier", "3", "journal", &p27)

	save("frontier", now, now.Add(5*time.Minute), map[string]string{"combat": "8"})
	check("fresh Frontier", "8", "frontier", nil)
}
