package gamestate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/perfstats"
)

func revisionsOf(s *Service) Revisions {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.views.rev
}

func waitRev(t *testing.T, s *Service, ok func(Revisions) bool) Revisions {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if r := revisionsOf(s); ok(r) {
			return r
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("revisions did not advance: %+v", revisionsOf(s))
	return Revisions{}
}

// TestPerfDomainRevisionDemand (G3K PERF-05/06): a change rebuilds only the
// domains it can affect and that are demanded; undemanded domains are
// deferred, read fresh, and published as soon as they are demanded.
func TestPerfDomainRevisionDemand(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-24T10:00:00Z","FID":"SYNTHETIC-A","Commander":"Fixture","Ship":"anaconda","ShipID":7}`)
	appendEvent(t, dir, `{"event":"Location","timestamp":"2026-11-24T10:00:01Z","StarSystem":"Fixture Alpha","SystemAddress":9007199254740993,"StarPos":[0,0,0]}`)
	writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
	s, _, stop := recordingService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 2 && v.SourceState == "live" })
	base := revisionsOf(s)

	// A Status-only change rebuilds the workspace alone.
	perfstats.Reset()
	status := func(flags int) {
		raw := []byte(`{"timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `","event":"Status","Flags":` + itoa(flags) + `,"Flags2":0,"Fuel":{"FuelMain":20,"FuelReservoir":0.5},"Cargo":0}`)
		if err := os.WriteFile(filepath.Join(dir, "Status.json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	status(0x01000008)
	r := waitRev(t, s, func(r Revisions) bool { return r.Flight > base.Flight })
	if r.Navigation != base.Navigation || r.Exploration != base.Exploration || r.Ship != base.Ship {
		t.Fatalf("a Status change advanced unrelated domains: %+v -> %+v", base, r)
	}
	c := perfstats.Take().Counters
	if c["domain.build.navigation"] != 0 || c["domain.build.exploration"] != 0 || c["domain.build.ship"] != 0 || c["domain.build.workspace"] == 0 {
		t.Fatalf("Status change built unrelated domains: %v", c)
	}

	// Unchanged reads reuse the published value.
	perfstats.Reset()
	a, b := s.Navigation(), s.Navigation()
	if a.CurrentSystem != b.CurrentSystem || perfstats.Take().Counters["getter.reuse.navigation"] != 2 {
		t.Fatalf("unchanged navigation was rebuilt: %v", perfstats.Take().Counters)
	}

	// Exploration not demanded: a scan is deferred, yet a read is fresh.
	s.SetDemand(DomainAll &^ DomainExploration)
	before := revisionsOf(s)
	appendEvent(t, dir, `{"event":"Scan","timestamp":"2026-11-24T10:00:05Z","ScanType":"Detailed","StarSystem":"Fixture Alpha","SystemAddress":9007199254740993,"BodyID":3,"BodyName":"Fixture Alpha 3","PlanetClass":"Icy body","DistanceFromArrivalLS":10}`)
	waitState(t, s, func(v Summary) bool { return v.EventsRead == 3 })
	if r := revisionsOf(s); r.Exploration != before.Exploration {
		t.Fatalf("undemanded exploration was built and published: %+v", r)
	}
	found := false
	for _, body := range s.Exploration().Bodies {
		found = found || body.Name == "Fixture Alpha 3"
	}
	if !found {
		t.Fatal("a read of a deferred domain returned stale data")
	}
	s.SetDemand(DomainAll)
	waitRev(t, s, func(r Revisions) bool { return r.Exploration > before.Exploration })

	// A source/account change republishes every domain.
	before = revisionsOf(s)
	appendEvent(t, dir, `{"event":"Shutdown","timestamp":"2026-11-24T10:00:06Z"}`)
	r = waitRev(t, s, func(r Revisions) bool { return r.Navigation > before.Navigation })
	if r.Exploration == before.Exploration || r.Ship == before.Ship || r.Workspace == before.Workspace {
		t.Fatalf("a source state change did not republish every domain: %+v -> %+v", before, r)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}
