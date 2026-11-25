package webapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/perfstats"
)

const contextFrame = "goNavigationContextChanged"

type contextPush struct {
	Navigation struct {
		Current *struct {
			Name string `json:"name"`
		} `json:"current"`
		Points []json.RawMessage `json:"points"`
		Mode   string            `json:"mode"`
		Fresh  bool              `json:"fresh"`
	} `json:"navigation"`
	Policy map[string]any `json:"policy"`
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// liveFixture: a verified, arrived live session (no NavRoute: route data is
// missing on purpose) with a paired display.
func liveFixture(t *testing.T) (*Server, string, *frames, *frames) {
	t.Helper()
	dir := t.TempDir()
	base := time.Now().Add(-5 * time.Second)
	writeGameEvent(t, dir, `{"event":"Fileheader","timestamp":"`+stamp(base)+`","part":1}`)
	writeGameEvent(t, dir, `{"event":"LoadGame","timestamp":"`+stamp(base)+`","FID":"SYNTHETIC-A","Commander":"Fixture","Ship":"anaconda","ShipID":7}`)
	writeGameEvent(t, dir, `{"event":"Location","timestamp":"`+stamp(base.Add(time.Second))+`","StarSystem":"Fixture Alpha","SystemAddress":9007199254740993,"StarPos":[0,0,0]}`)
	s, _, viewer, url, _ := startGameFixture(t, dir, true, "Fixture Alpha", true, true, true)
	deadline := time.Now().Add(5 * time.Second)
	for v := s.game.Snapshot(); v.ImportInProgress || v.SourceState != "live"; v = s.game.Snapshot() {
		if time.Now().After(deadline) {
			t.Fatalf("not live: %+v", v)
		}
		time.Sleep(5 * time.Millisecond)
	}
	// The running game writes live events; replayed history alone stays
	// "cached" and is never presented as fresh flight state.
	writeGameEvent(t, dir, `{"event":"Music","timestamp":"`+stamp(time.Now())+`","MusicTrack":"NoTrack"}`)
	writeStatus(t, dir, time.Now(), 0x01000000, 0, 20)
	for s.game.Workspace().Parity.Status.State != "known" || !s.game.NavigationContext().Fresh {
		if time.Now().After(deadline) {
			t.Fatalf("live state not reached: %+v", s.game.NavigationContext())
		}
		time.Sleep(5 * time.Millisecond)
	}
	paired := collect(t, pairedSocket(t, viewer, url))
	owner := collect(t, dialScoped(t, s, ""))
	for _, f := range []*frames{paired, owner} {
		f.rpc(t, "setGoDemand", demand(1, true, "context"))
		f.rpc(t, "getGoNavigationContext", nil)
	}
	return s, dir, paired, owner
}

func writeStatus(t *testing.T, dir string, at time.Time, flags, flags2 int, fuel int) {
	t.Helper()
	raw := `{"event":"Status","timestamp":"` + stamp(at) + `","Flags":` + itoa(flags) + `,"Flags2":` + itoa(flags2) + `,"Fuel":{"FuelMain":` + itoa(fuel) + `,"FuelReservoir":0.5},"Cargo":0}`
	if err := os.WriteFile(filepath.Join(dir, "Status.json"), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
}

// nextContext waits for the next context push and decodes it.
func nextContext(t *testing.T, f *frames, within time.Duration) contextPush {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case n := <-f.names:
			if n != contextFrame {
				continue
			}
			var p contextPush
			if err := json.Unmarshal([]byte(f.payload(contextFrame)), &p); err != nil {
				t.Fatal(err)
			}
			return p
		case <-deadline:
			t.Fatalf("no navigation context push within %v", within)
		}
	}
}

func contextPushes(f *frames, d time.Duration) int { return count(f.drain(d), contextFrame) }

func current(p contextPush) string {
	if p.Navigation.Current == nil {
		return ""
	}
	return p.Navigation.Current.Name
}

// TestG3LEventDrivenFSDTransitions: charge, jump and arrival all complete in
// well under one second; every paired and local display receives each
// transition as it happens (formerly the 1 s sampler could merge or skip
// them), unchanged state is never re-sent, stale evidence does not publish,
// a hidden display receives nothing and gets only the current state on
// resume, and a reconnect reads the current state without replay.
func TestG3LEventDrivenFSDTransitions(t *testing.T) {
	s, dir, paired, owner := liveFixture(t)
	displays := map[string]*frames{"paired": paired, "owner": owner}
	for _, f := range displays {
		f.drain(150 * time.Millisecond)
	}

	start := time.Now()
	type step struct {
		act  func()
		mode string
		at   string
	}
	steps := []step{
		{func() { writeStatus(t, dir, time.Now(), 0x01000000, 1<<19, 20) }, "charge", "Fixture Alpha"},
		{func() {
			writeGameEvent(t, dir, `{"event":"StartJump","timestamp":"`+stamp(time.Now())+`","JumpType":"Hyperspace","StarSystem":"Fixture Beta","SystemAddress":9007199254740994}`)
		}, "jump", "Fixture Alpha"},
		{func() {
			writeGameEvent(t, dir, `{"event":"FSDJump","timestamp":"`+stamp(time.Now())+`","StarSystem":"Fixture Beta","SystemAddress":9007199254740994,"StarPos":[5,0,0]}`)
		}, "idle", "Fixture Beta"},
	}
	for i, st := range steps {
		st.act()
		for name, f := range displays {
			p := nextContext(t, f, 900*time.Millisecond)
			if p.Navigation.Mode != st.mode || current(p) != st.at || !p.Navigation.Fresh {
				t.Fatalf("%s step %d: mode=%q current=%q fresh=%v, want %q at %q", name, i, p.Navigation.Mode, current(p), p.Navigation.Fresh, st.mode, st.at)
			}
			if len(p.Navigation.Points) == 0 {
				t.Fatalf("%s: missing route data removed the current system from context", name)
			}
		}
	}
	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Fatalf("charge->jump->arrival delivery took %v", elapsed)
	}

	// Unchanged context (a fuel-only Status) and delayed/stale evidence (a
	// charge older than the arrival) publish nothing.
	writeStatus(t, dir, time.Now(), 0x01000000, 0, 19)
	time.Sleep(100 * time.Millisecond)
	writeStatus(t, dir, start.Add(-2*time.Second), 0x01000000, 1<<19, 19)
	for name, f := range displays {
		if n := contextPushes(f, 300*time.Millisecond); n != 0 {
			t.Fatalf("%s received %d context pushes for unchanged/stale input", name, n)
		}
	}

	// Rapid events while hidden: none delivered; resume gets the current
	// state once (no stale replay of the intermediate charge/jump).
	paired.rpc(t, "setGoDemand", demand(2, false, "context"))
	writeStatus(t, dir, time.Now(), 0x01000000, 1<<19, 19)
	nextContext(t, owner, 900*time.Millisecond) // visible display still served
	writeGameEvent(t, dir, `{"event":"StartJump","timestamp":"`+stamp(time.Now())+`","JumpType":"Hyperspace","StarSystem":"Fixture Gamma","SystemAddress":9007199254740995}`)
	writeGameEvent(t, dir, `{"event":"FSDJump","timestamp":"`+stamp(time.Now())+`","StarSystem":"Fixture Gamma","SystemAddress":9007199254740995,"StarPos":[9,0,0]}`)
	for deadline := time.Now().Add(3 * time.Second); s.game.Snapshot().CurrentSystem != "Fixture Gamma"; {
		if time.Now().After(deadline) {
			t.Fatal("arrival not ingested")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if n := contextPushes(paired, 200*time.Millisecond); n != 0 {
		t.Fatalf("hidden display received %d context pushes", n)
	}
	owner.drain(0)
	paired.rpc(t, "setGoDemand", demand(3, true, "context"))
	p := nextContext(t, paired, 900*time.Millisecond)
	if current(p) != "Fixture Gamma" || p.Navigation.Mode != "idle" {
		t.Fatalf("resume delivered stale state: %+v", p.Navigation)
	}
	if n := contextPushes(paired, 200*time.Millisecond); n != 0 {
		t.Fatalf("resume replayed %d extra context pushes", n)
	}

	// Reconnect: the explicit read returns the current state.
	paired.conn.CloseNow()
	r := owner.rpc(t, "getGoNavigationContext", nil)
	if !strings.Contains(string(r["message"]), `"Fixture Gamma"`) || !strings.Contains(string(r["message"]), `"mode":"idle"`) {
		t.Fatalf("reconnect read: %s", r["message"])
	}
}

// A display-policy change made in the launcher reaches open displays at once
// without waiting for a navigation event.
func TestG3LPolicyChangePublishesContext(t *testing.T) {
	s, _, paired, owner := liveFixture(t)
	paired.drain(150 * time.Millisecond)
	owner.drain(0)
	launcher, ctx := dialOwner(t, s)
	config := displayConfig{Charge: true, Jump: false, Restore: true, CloseMode: "tray"}
	if len(rpcMessage(t, ctx, launcher, "setGoCompanion", config)["error"]) != 0 {
		t.Fatal("policy save rejected")
	}
	p := nextContext(t, paired, 900*time.Millisecond)
	if p.Policy["jump"] != false {
		t.Fatalf("policy push: %+v", p.Policy)
	}
}

// Several open displays share one context build per change (G3K memo, now
// keyed by the change signal), and each receives each transition once.
func TestG3LContextBuildSharedAcrossDisplays(t *testing.T) {
	s, dir, paired, owner := liveFixture(t)
	displays := []*frames{paired, owner}
	for i := 0; i < 2; i++ {
		f := collect(t, dialScoped(t, s, ""))
		f.rpc(t, "setGoDemand", demand(1, true, "context"))
		f.rpc(t, "getGoNavigationContext", nil)
		displays = append(displays, f)
	}
	for _, f := range displays {
		f.drain(0)
	}
	time.Sleep(150 * time.Millisecond)
	for _, f := range displays {
		f.drain(0)
	}
	perfstats.Reset()
	writeStatus(t, dir, time.Now(), 0x01000000, 1<<19, 20)
	for _, f := range displays {
		nextContext(t, f, 900*time.Millisecond)
	}
	writeGameEvent(t, dir, `{"event":"FSDJump","timestamp":"`+stamp(time.Now())+`","StarSystem":"Fixture Beta","SystemAddress":9007199254740994,"StarPos":[5,0,0]}`)
	for _, f := range displays {
		if p := nextContext(t, f, 900*time.Millisecond); current(p) != "Fixture Beta" {
			t.Fatalf("arrival: %+v", p.Navigation)
		}
	}
	for _, f := range displays {
		if n := contextPushes(f, 200*time.Millisecond); n != 0 {
			t.Fatalf("duplicate context pushes: %d", n)
		}
	}
	c := perfstats.Take().Counters
	if c["context.push"] != int64(2*len(displays)) {
		t.Fatalf("pushes %d, want %d", c["context.push"], 2*len(displays))
	}
	if c["context.build"] >= c["context.push"] || c["context.build"] > c["context.signal"] {
		t.Fatalf("context built per display: %v", c)
	}
	t.Logf("2 transitions x %d displays: %d signals, %d builds, %d reuses, %d pushes", len(displays), c["context.signal"], c["context.build"], c["context.reuse"], c["context.push"])
}
