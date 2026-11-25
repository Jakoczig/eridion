package gamestate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func g3lStamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func g3lStatus(t *testing.T, dir string, flags2, fuel int) {
	t.Helper()
	raw := `{"event":"Status","timestamp":"` + g3lStamp(time.Now()) + `","Flags":16777216,"Flags2":` + itoa(flags2) + `,"Fuel":{"FuelMain":` + itoa(fuel) + `,"FuelReservoir":0.5},"Cargo":0}`
	if err := os.WriteFile(filepath.Join(dir, "Status.json"), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
}

// waitSignal waits until the context signal advances past seq.
func waitSignal(t *testing.T, s *Service, seq uint64, ch <-chan struct{}) uint64 {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("navigation context signal did not advance")
	}
	next, _ := s.NavigationContextSignal()
	if next <= seq {
		t.Fatalf("signal closed without advancing: %d -> %d", seq, next)
	}
	return next
}

// TestG3LNavigationContextSignal: the signal advances for every navigation
// input change (Status charge, StartJump, arrival, NavRoute) and stays quiet
// for unrelated Status writes and for a continuing charge; the snapshot is
// consistent with the sequence it reports.
func TestG3LNavigationContextSignal(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	base := time.Now().Add(-5 * time.Second)
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"`+g3lStamp(base)+`","FID":"SYNTHETIC-A","Commander":"Fixture","Ship":"anaconda","ShipID":7}`)
	appendEvent(t, dir, `{"event":"Location","timestamp":"`+g3lStamp(base.Add(time.Second))+`","StarSystem":"Fixture Alpha","SystemAddress":9007199254740993,"StarPos":[0,0,0]}`)
	writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
	s, _, stop := recordingService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.SourceState == "live" })
	appendEvent(t, dir, `{"event":"Music","timestamp":"`+g3lStamp(time.Now())+`","MusicTrack":"NoTrack"}`)
	g3lStatus(t, dir, 0, 20)
	deadline := time.Now().Add(3 * time.Second)
	for !s.NavigationContext().Fresh || s.Workspace().Parity.Status.State != "known" {
		if time.Now().After(deadline) {
			t.Fatal("not live")
		}
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)

	quiet := func(what string, act func()) {
		t.Helper()
		seq, _ := s.NavigationContextSignal()
		flight := revisionsOf(s).Flight
		act()
		waitRev(t, s, func(r Revisions) bool { return r.Flight > flight })
		time.Sleep(30 * time.Millisecond)
		if after, _ := s.NavigationContextSignal(); after != seq {
			t.Fatalf("%s advanced the navigation context signal", what)
		}
	}
	step := func(act func(), mode, current string) {
		t.Helper()
		seq, ch := s.NavigationContextSignal()
		act()
		waitSignal(t, s, seq, ch)
		deadline := time.Now().Add(time.Second)
		for {
			v, _, until := s.NavigationContextSnapshot()
			if v.Mode == mode && v.Current != nil && v.Current.Name == current {
				if (mode == "idle") != until.IsZero() {
					t.Fatalf("mode %s deadline %v", mode, until)
				}
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("want %s at %s, got %+v", mode, current, v)
			}
			time.Sleep(2 * time.Millisecond)
		}
	}

	quiet("a fuel-only Status", func() { g3lStatus(t, dir, 0, 19) })
	step(func() { g3lStatus(t, dir, 1<<19, 19) }, "charge", "Fixture Alpha")
	quiet("a continuing charge", func() { g3lStatus(t, dir, 1<<19, 18) })
	step(func() {
		appendEvent(t, dir, `{"event":"StartJump","timestamp":"`+g3lStamp(time.Now())+`","JumpType":"Hyperspace","StarSystem":"Fixture Beta","SystemAddress":9007199254740994}`)
	}, "jump", "Fixture Alpha")
	step(func() {
		appendEvent(t, dir, `{"event":"FSDJump","timestamp":"`+g3lStamp(time.Now())+`","StarSystem":"Fixture Beta","SystemAddress":9007199254740994,"StarPos":[5,0,0]}`)
	}, "idle", "Fixture Beta")
	seq, ch := s.NavigationContextSignal()
	route := `{"timestamp":"` + g3lStamp(time.Now()) + `","event":"NavRoute","Route":[{"StarSystem":"Fixture Beta","SystemAddress":9007199254740994,"StarPos":[5,0,0],"StarClass":"G"},{"StarSystem":"Fixture Gamma","SystemAddress":9007199254740995,"StarPos":[9,0,0],"StarClass":"K"}]}`
	if err := os.WriteFile(filepath.Join(dir, "NavRoute.json"), []byte(route), 0o600); err != nil {
		t.Fatal(err)
	}
	waitSignal(t, s, seq, ch)
}
