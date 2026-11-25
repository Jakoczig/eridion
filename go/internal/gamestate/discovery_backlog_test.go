package gamestate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/journal"
)

// Task 9 / F11 through the real service: a large journal directory (bounded
// batches, newest first) still ends with chronological state: survey
// evidence known only from the oldest journal is part of the current system
// view once the backlog drains (the chronological rebuild ran); a restart
// imports nothing twice and does not rebuild again.
func TestAuditDiscoveryBacklogChronologicalState(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	const files = 600 // above the reader's full-scan limit
	for i := 1; i <= files; i++ {
		lines := fmt.Sprintf(`{"timestamp":"2026-01-01T%02d:%02d:%02dZ","event":"Fileheader","part":1}`+"\n", i/3600, i/60%60, i%60)
		lines += fmt.Sprintf(`{"timestamp":"2026-01-01T%02d:%02d:%02dZ","event":"LoadGame","FID":"SYNTHETIC-A","Commander":"Fixture","Ship":"anaconda","ShipID":7}`+"\n", i/3600, i/60%60, i%60)
		// One intermediate system: the exploration model keeps a bounded
		// number of systems per commander, so distinct ones would evict Home.
		system, address := "Fixture Elsewhere", 1001
		if i == 1 || i == files {
			system, address = "Fixture Home", 4242 // first and latest session in one system
		}
		lines += fmt.Sprintf(`{"timestamp":"2026-01-01T%02d:%02d:%02dZ","event":"Location","StarSystem":%q,"SystemAddress":%d}`+"\n", i/3600, i/60%60, i%60, system, address)
		if i == 1 {
			// Survey evidence known only from the oldest journal.
			lines += `{"timestamp":"2026-01-01T00:00:01Z","event":"Scan","ScanType":"Detailed","StarSystem":"Fixture Home","SystemAddress":4242,"BodyID":3,"BodyName":"Fixture Home 3","PlanetClass":"Icy body","DistanceFromArrivalLS":12}` + "\n"
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("Journal.2026%06d.01.log", i)), []byte(lines), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
	s, _, stop := recordingService(t, profile, dir)
	// Review #1: observe every published summary (not a sampled one). While
	// the discovery backlog drains, the import is not finished. The G3K-era
	// reader reported ImportInProgress=false / "live" from the first poll
	// (events 774..1782 of 1800 in a probe). Older journals never replaced
	// the active projection (the active-source rules held), so that part
	// of the finding was disproved.
	violation := make(chan Summary, 1)
	subCtx, cancelSub := context.WithCancel(context.Background())
	changes, unsubscribe := s.Subscribe(subCtx)
	go func() {
		for v := range changes {
			if !v.ImportInProgress && v.EventsRead > 0 && v.EventsRead < files*3 {
				select {
				case violation <- v:
				default:
				}
			}
		}
	}()
	expected := uint64(files*3 + 1) // three lines per journal plus the scan
	deadline := time.Now().Add(60 * time.Second)
	for {
		v := s.Snapshot()
		bodies := 0
		for _, b := range s.Exploration().Bodies {
			if b.Name == "Fixture Home 3" {
				bodies++
			}
		}
		select {
		case bad := <-violation:
			t.Fatalf("incomplete history presented as a finished import: %+v", bad)
		default:
		}
		if v.EventsRead >= expected && !v.ImportInProgress && v.CurrentSystem == "Fixture Home" && bodies == 1 && s.Details().ReaderStarts >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("chronological state not reached: %+v oldestBody=%d readerStarts=%d", v, bodies, s.Details().ReaderStarts)
		}
		time.Sleep(20 * time.Millisecond)
	}
	count := func() int {
		store, _ := s.History()
		n := 0
		if err := store.Walk(context.Background(), func(journal.Event) error { n++; return nil }); err != nil {
			t.Fatal(err)
		}
		return n
	}
	unsubscribe()
	cancelSub()
	select {
	case bad := <-violation:
		t.Fatalf("incomplete history presented as a finished import: %+v", bad)
	default:
	}
	n := count()
	if n < files*3 {
		t.Fatalf("stored events %d, want at least %d", n, files*3)
	}
	stop()
	// Restart: no duplicates and no new rebuild.
	s, _, stop = recordingService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress })
	time.Sleep(500 * time.Millisecond)
	if after := count(); after != n {
		t.Fatalf("restart duplicated history: %d -> %d", n, after)
	}
}
