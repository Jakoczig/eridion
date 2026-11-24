package webapp

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/gamestate"
	"github.com/Jakoczig/eridion/go/internal/journal"
	"github.com/Jakoczig/eridion/go/internal/perfstats"
	"github.com/coder/websocket"
)

// frames reads a connection continuously (a timed Read would close it) and
// records pushed frame names and the last payload per name.
type frames struct {
	conn    *websocket.Conn
	names   chan string
	replies chan map[string]json.RawMessage
	mu      sync.Mutex
	last    map[string]json.RawMessage
}

func collect(t *testing.T, conn *websocket.Conn) *frames {
	f := &frames{conn: conn, names: make(chan string, 4096), replies: make(chan map[string]json.RawMessage, 64), last: map[string]json.RawMessage{}}
	go func() {
		for {
			_, raw, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			var m map[string]json.RawMessage
			if json.Unmarshal(raw, &m) != nil {
				continue
			}
			if _, reply := m["requestId"]; reply {
				f.replies <- m
				continue
			}
			var name string
			_ = json.Unmarshal(m["name"], &name)
			f.mu.Lock()
			f.last[name] = m["message"]
			f.mu.Unlock()
			f.names <- name
		}
	}()
	return f
}

func (f *frames) payload(name string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return string(f.last[name])
}

// rpc sends name/message and returns its reply.
func (f *frames) rpc(t *testing.T, name string, message any) map[string]json.RawMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id := "perf-" + name
	data, _ := json.Marshal(map[string]any{"requestId": id, "name": name, "message": message})
	if err := f.conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case m := <-f.replies:
			if string(m["requestId"]) == `"`+id+`"` {
				return m
			}
		case <-ctx.Done():
			t.Fatal("no reply to " + name)
		}
	}
}

// drain waits d and returns the names pushed meanwhile (and earlier).
func (f *frames) drain(d time.Duration) []string {
	time.Sleep(d)
	out := []string{}
	for {
		select {
		case n := <-f.names:
			out = append(out, n)
		default:
			return out
		}
	}
}

func count(names []string, name string) int {
	n := 0
	for _, x := range names {
		if x == name {
			n++
		}
	}
	return n
}

func demandFixture(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	writeGameEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-24T10:00:00Z","FID":"SYNTHETIC-A","Commander":"Fixture","Ship":"anaconda","ShipID":7}`)
	writeGameEvent(t, dir, `{"event":"Location","timestamp":"2026-11-24T10:00:01Z","StarSystem":"Fixture Alpha","SystemAddress":9007199254740993,"StarPos":[0,0,0]}`)
	s, err := StartLocalWithOptions(context.Background(), fixtureFS(), Options{StorageDir: t.TempDir(), JournalDir: dir, EnableJournal: true, EnableNavigation: true, EnableExploration: true, EnableDashboard: true, EnableParity: true, Discovery: journal.DiscoveryOptions{Roots: []string{}}, PollInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close(context.Background()) })
	deadline := time.Now().Add(5 * time.Second)
	for s.game.Snapshot().CurrentSystem != "Fixture Alpha" || s.game.Snapshot().ImportInProgress {
		if time.Now().After(deadline) {
			t.Fatal("fixture import")
		}
		time.Sleep(5 * time.Millisecond)
	}
	return s, dir
}

// jump appends an FSDJump and waits until the service ingested it.
func jump(t *testing.T, s *Server, dir, name string, n int) {
	t.Helper()
	writeGameEvent(t, dir, `{"event":"FSDJump","timestamp":"2026-11-24T10:00:0`+itoa(n)+`Z","StarSystem":"`+name+`","SystemAddress":`+itoa(9007199254740994+n)+`,"StarPos":[`+itoa(n)+`,0,0]}`)
	deadline := time.Now().Add(5 * time.Second)
	for s.game.Snapshot().CurrentSystem != name {
		if time.Now().After(deadline) {
			t.Fatalf("journal event %s not ingested", name)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func waitDemand(t *testing.T, s *Server, want uint8) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for s.game.Demand() != want {
		if time.Now().After(deadline) {
			t.Fatalf("host demand = %04b, want %04b", s.game.Demand(), want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func demand(generation int, visible bool, topics ...string) map[string]any {
	return map[string]any{"generation": json.Number(itoa(generation)), "visible": visible, "topics": topics}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

// TestPerfDemandLifecycle (G3K PERF-06).
func TestPerfDemandLifecycle(t *testing.T) {
	s, dir := demandFixture(t)
	a, b := collect(t, dialScoped(t, s, "")), collect(t, dialScoped(t, s, ""))
	// Undeclared (legacy) connections keep everything prepared.
	waitDemand(t, s, gamestate.DomainAll)

	// Identical demand: both want navigation; one encoding is shared.
	a.rpc(t, "setGoDemand", demand(1, true, "navigation"))
	b.rpc(t, "setGoDemand", demand(1, true, "navigation"))
	waitDemand(t, s, gamestate.DomainNavigation)
	a.drain(100 * time.Millisecond)
	b.drain(100 * time.Millisecond)
	perfstats.Reset()
	jump(t, s, dir, "Fixture Beta", 2)
	na, nb := a.drain(300*time.Millisecond), b.drain(300*time.Millisecond)
	if count(na, "goNavigationChanged") != 1 || count(nb, "goNavigationChanged") != 1 {
		t.Fatalf("each display gets the navigation change once: %v / %v", na, nb)
	}
	if count(na, "goExplorationChanged")+count(nb, "goExplorationChanged")+count(na, "goShipChanged")+count(na, "goWorkspaceChanged") != 0 {
		t.Fatalf("undemanded topics were pushed: %v / %v", na, nb)
	}
	if perfstats.Take().Counters["ws.shared.goNavigationChanged"] != 1 {
		t.Fatalf("identical demand encoded twice: %v", perfstats.Take().Counters)
	}

	// Different demand: b wants the ship only.
	b.rpc(t, "setGoDemand", demand(2, true, "ship"))
	waitDemand(t, s, gamestate.DomainNavigation|gamestate.DomainShip)
	b.drain(200 * time.Millisecond)
	jump(t, s, dir, "Fixture Gamma", 3)
	if nb := b.drain(300 * time.Millisecond); count(nb, "goNavigationChanged") != 0 {
		t.Fatalf("navigation pushed to a ship-only display: %v", nb)
	}

	// A late (older) generation is ignored.
	if r := b.rpc(t, "setGoDemand", demand(1, true, "navigation")); !strings.Contains(string(r["message"]), `"applied":false`) {
		t.Fatalf("late generation applied: %s", r["message"])
	}

	// Hidden: no pushes; visible again: the current state at once.
	a.rpc(t, "setGoDemand", demand(2, false, "navigation"))
	a.drain(100 * time.Millisecond)
	jump(t, s, dir, "Fixture Delta", 4)
	if na := a.drain(300 * time.Millisecond); count(na, "goNavigationChanged")+count(na, "journalSummaryChanged") > 0 {
		t.Fatalf("hidden display received updates: %v", na)
	}
	a.rpc(t, "setGoDemand", demand(3, true, "navigation"))
	a.drain(400 * time.Millisecond)
	if !strings.Contains(a.payload("journalSummaryChanged"), "Fixture Delta") {
		t.Fatalf("resume did not deliver the held summary: %s", a.payload("journalSummaryChanged"))
	}
	if !strings.Contains(a.payload("goNavigationChanged"), "Fixture Delta") {
		t.Fatalf("resume did not deliver the current state: %s", a.payload("goNavigationChanged"))
	}

	// Last release: nothing prepared in normal mode.
	a.rpc(t, "setGoDemand", demand(4, false))
	b.rpc(t, "setGoDemand", demand(3, true))
	waitDemand(t, s, 0)
	// Low latency keeps the bounded warm set prepared with no display.
	s.companion.mu.Lock()
	s.companion.config.LowLatency = true
	s.companion.mu.Unlock()
	s.recomputeDemand()
	waitDemand(t, s, gamestate.DomainAll)
	// Closing every connection keeps the warm set; off releases it.
	a.conn.CloseNow()
	b.conn.CloseNow()
	time.Sleep(100 * time.Millisecond)
	waitDemand(t, s, gamestate.DomainAll)
	s.companion.mu.Lock()
	s.companion.config.LowLatency = false
	s.companion.mu.Unlock()
	s.recomputeDemand()
	waitDemand(t, s, 0)
	// Gameplay continues without demand: the journal is still ingested.
	jump(t, s, dir, "Fixture Epsilon", 5) // ingestion continues without display demand
	if s.game.Navigation().CurrentSystem != "Fixture Epsilon" {
		t.Fatal("an undemanded domain read was stale")
	}
}

func TestPerfDemandValidation(t *testing.T) {
	for _, raw := range []string{`{"generation":0,"visible":true,"topics":[]}`, `{"generation":"x","visible":true}`, `{"generation":1,"visible":true,"topics":["launcher"]}`, `{"generation":1,"visible":true,"topics":["a","b","c","d","e","f","g","h","i","j","k","l","m","n","o","p","q"]}`, `{"generation":1,"fid":"OTHER"}`} {
		if _, err := parseDemand(json.RawMessage(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	d, err := parseDemand(json.RawMessage(`{"generation":"18446744073709551615","visible":true,"topics":["navigation","workspace"]}`))
	if err != nil || d.generation != 18446744073709551615 || !d.allows(topicWorkspace) || d.allows(topicShip) {
		t.Fatalf("%+v %v", d, err)
	}
}

// The launcher setting "Low latency: keep views ready" (default off) sets
// the warm demand once saved; with no display, normal mode prepares nothing.
func TestPerfLowLatencySetting(t *testing.T) {
	s, _ := demandFixture(t)
	waitDemand(t, s, 0)
	if s.lowLatency() {
		t.Fatal("low latency must default to off")
	}
	if err := s.saveDisplayConfig([]byte(`{"closeMode":"tray","restore":true,"lowLatency":true,"devices":[]}`)); err != nil {
		t.Fatal(err)
	}
	waitDemand(t, s, gamestate.DomainAll)
	if err := s.saveDisplayConfig([]byte(`{"closeMode":"bogus","lowLatency":false}`)); err == nil {
		t.Fatal("invalid settings accepted")
	}
	waitDemand(t, s, gamestate.DomainAll) // a failed save keeps the effective mode
	if err := s.saveDisplayConfig([]byte(`{"closeMode":"tray","restore":true,"lowLatency":false,"devices":[]}`)); err != nil {
		t.Fatal(err)
	}
	waitDemand(t, s, 0)
}

// The initial snapshot a connection receives has no published revision; its
// encoding must never be shared with a later connection (whose initial
// snapshot may differ).
func TestPerfInitialSnapshotNotShared(t *testing.T) {
	s, dir := demandFixture(t)
	a := collect(t, dialScoped(t, s, ""))
	a.drain(200 * time.Millisecond)
	s.game.SetDemand(0) // nothing prepared: the next jump is not published
	jump(t, s, dir, "Fixture Later", 6)
	b := collect(t, dialScoped(t, s, ""))
	b.drain(300 * time.Millisecond)
	if !strings.Contains(b.payload("goNavigationChanged"), "Fixture Later") {
		t.Fatalf("a new connection received another connection's older initial snapshot: %s", b.payload("goNavigationChanged"))
	}
	_ = a
}

// Review R1: concurrent demand changes are applied to the game service in
// the order their unions were computed; the host never ends on a stale mask.
func TestPerfDemandConcurrentApplyConsistent(t *testing.T) {
	s, _ := demandFixture(t)
	conns := make([]*websocket.Conn, 4)
	for i := range conns {
		conns[i] = dialScoped(t, s, "")
	}
	waitDemand(t, s, gamestate.DomainAll)
	topics := [][]string{{"navigation"}, {"ship"}, {"navigation", "exploration"}, {"workspace"}, {}}
	var wg sync.WaitGroup
	for round := 1; round <= 40; round++ {
		for i, c := range conns {
			wg.Add(1)
			go func(c *websocket.Conn, gen, k int) {
				defer wg.Done()
				d, _ := parseDemand(json.RawMessage(`{"generation":"` + itoa(gen) + `","visible":true,"topics":` + func() string { b, _ := json.Marshal(topics[k%len(topics)]); return string(b) }() + `}`))
				s.updateDemand(c, d, false)
			}(c, round, round+i)
		}
		wg.Wait()
		s.demand.mu.Lock()
		want := s.demand.last
		s.demand.mu.Unlock()
		if got := s.game.Demand(); got != want {
			t.Fatalf("round %d: host demand %04b, registry %04b", round, got, want)
		}
	}
}

// Review R2: client-chosen request names never create new counters.
func TestPerfFrameCounterNamesBounded(t *testing.T) {
	perfstats.Reset()
	for i := 0; i < 50; i++ {
		recordFrame(map[string]any{"requestId": "x", "name": "random-" + itoa(i)}, 10)
	}
	recordFrame(map[string]any{"requestId": "x", "name": "getGoShip"}, 10)
	c := perfstats.Take().Counters
	if c["ws.frames.rpc.other"] != 50 || c["ws.frames.rpc.getGoShip"] != 1 || len(c) > 6 {
		t.Fatalf("unbounded counter names: %v", c)
	}
}
