package webapp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/providers"
	"github.com/coder/websocket"
)

// dialScoped opens an owner-session socket the way a page does: the launcher
// window connects with scope=launcher, every gameplay window without it.
func dialScoped(t *testing.T, s *Server, scope string) *websocket.Conn {
	t.Helper()
	client := ownerClient(t, s)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := strings.Replace(s.URL(), "http:", "ws:", 1)
	if scope != "" {
		url += "/?scope=" + scope
	}
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPClient: client, HTTPHeader: http.Header{"Origin": []string{s.URL()}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

// readEpoch waits for the host's accountEpoch frame.
func readEpoch(t *testing.T, ctx context.Context, conn *websocket.Conn) uint64 {
	t.Helper()
	for {
		_, raw, err := conn.Read(ctx)
		if err != nil {
			t.Fatal("no accountEpoch frame:", err)
		}
		var frame struct {
			Name    string `json:"name"`
			Message struct {
				Epoch uint64 `json:"epoch"`
			} `json:"message"`
		}
		if json.Unmarshal(raw, &frame) == nil && frame.Name == "accountEpoch" {
			return frame.Message.Epoch
		}
	}
}

func epochRPC(t *testing.T, ctx context.Context, conn *websocket.Conn, id, name string, epoch uint64, message any) map[string]json.RawMessage {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"requestId": id, "name": name, "epoch": epoch, "message": message})
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
	for {
		_, raw, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var response map[string]json.RawMessage
		if json.Unmarshal(raw, &response) == nil && string(response["requestId"]) == `"`+id+`"` {
			return response
		}
	}
}

func twoCommanderServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	writeGameEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-01-01T00:00:01Z","FID":"SYNTHETIC-A"}`)
	writeGameEvent(t, dir, `{"event":"Location","timestamp":"2026-01-01T00:00:02Z","StarSystem":"Fixture A","SystemAddress":1}`)
	writeGameEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-01-01T00:00:03Z","FID":"SYNTHETIC-B"}`)
	writeGameEvent(t, dir, `{"event":"Location","timestamp":"2026-01-01T00:00:04Z","StarSystem":"Fixture B","SystemAddress":2}`)
	s, e := StartLocalWithOptions(context.Background(), fixtureFS(), Options{StorageDir: t.TempDir(), JournalDir: dir, EnableJournal: true, EnableNavigation: true, EnableExploration: true, EnableDashboard: true, EnableParity: true, ProviderOptions: providers.Options{Vault: &accountTestVault{}, Manual: true}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close(context.Background()) })
	deadline := time.Now().Add(3 * time.Second)
	for v := s.game.Snapshot(); v.ImportInProgress || v.EventsRead != 4; v = s.game.Snapshot() {
		if time.Now().After(deadline) {
			t.Fatal("journal replay not ready")
		}
		time.Sleep(time.Millisecond)
	}
	raw, _ := json.Marshal(map[string]any{"multiple": true, "selected": "SYNTHETIC-A"})
	if _, e := s.accountRPC(context.Background(), "setGoAccounts", raw); e != nil {
		t.Fatal(e)
	}
	return s
}

// Audit F2 (Task 13): a host commander switch invalidates the main display
// too. Its socket is closed (pending client callbacks are cancelled), nothing
// is answered under the old commander, and a request stamped with the old
// account epoch is rejected on any connection, so a queued itinerary import
// cannot run under the new commander.
func TestAuditCommanderSwitchInvalidatesMainDisplay(t *testing.T) {
	s := twoCommanderServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	main := dialScoped(t, s, "")
	launcher := dialScoped(t, s, "launcher")
	before := readEpoch(t, ctx, main)
	if got := readEpoch(t, ctx, launcher); got != before {
		t.Fatalf("connections disagree on the epoch: %d / %d", before, got)
	}

	if r := epochRPC(t, ctx, launcher, "switch", "setGoAccounts", before, map[string]any{"multiple": true, "selected": "SYNTHETIC-B"}); len(r["error"]) != 0 {
		t.Fatal("launcher switch failed", string(r["error"]))
	}

	// The main display connection opened under A is closed.
	_ = main.Write(ctx, websocket.MessageText, []byte(`{"name":"getJournalSummary","requestId":"after-switch"}`))
	for {
		_, raw, err := main.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				t.Fatal("main display socket was not closed on the commander switch")
			}
			break
		}
		if strings.Contains(string(raw), "after-switch") {
			t.Fatal("main display answered under the previous commander")
		}
	}

	// A queued action stamped with A's epoch never runs under B.
	r := epochRPC(t, ctx, launcher, "queued", "importGoItinerary", before, map[string]any{"name": "Queued under A", "csv": "system\nAlpha\nBeta\n"})
	if !strings.Contains(string(r["error"]), "Commander changed") {
		t.Fatalf("stale-epoch import was not rejected: %s %s", r["error"], r["message"])
	}
	db, _ := s.game.History()
	// No itinerary at all for B (ErrNoRows), or none named after A's queue.
	if page, err := db.Itinerary("SYNTHETIC-B", ""); err == nil {
		if raw, _ := json.Marshal(page); strings.Contains(string(raw), "Queued under A") {
			t.Fatal("old-commander import ran under the new commander", string(raw))
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}

	// The reconnected main display works under the new epoch.
	next := dialScoped(t, s, "")
	after := readEpoch(t, ctx, next)
	if after == before {
		t.Fatal("commander switch did not advance the account epoch")
	}
	if r := epochRPC(t, ctx, next, "fresh", "getJournalSummary", after, nil); len(r["error"]) != 0 {
		t.Fatal("fresh request rejected", string(r["error"]))
	}
	var stamped uint64
	if r := epochRPC(t, ctx, next, "stamp", "getLoadingStatus", after, nil); json.Unmarshal(r["epoch"], &stamped) != nil || stamped != after {
		t.Fatalf("reply not stamped with its epoch: %s", r["epoch"])
	}
}

// Audit F5 (Task 13 / contract 4): launcher management is enforced per
// connection. The main gameplay display shares the owner session but cannot
// call settings, credential, history or pairing RPCs; the launcher can.
func TestAuditMainDisplayCannotCallLauncherRPCs(t *testing.T) {
	s := twoCommanderServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	main := dialScoped(t, s, "")
	for name := range launcherRPC {
		r := rpcWithMessage(t, ctx, main, name, map[string]any{})
		var message string
		_ = json.Unmarshal(r["error"], &message)
		if message != launcherOnlyError || r["message"] != nil {
			t.Fatalf("main gameplay display reached launcher RPC %s: %s", name, r["error"])
		}
	}
	launcher := dialScoped(t, s, "launcher")
	for _, name := range []string{"getGoProviders", "getGoAccounts", "getGoLauncher"} {
		r := rpcWithMessage(t, ctx, launcher, name, map[string]any{})
		var message string
		_ = json.Unmarshal(r["error"], &message)
		if message == launcherOnlyError {
			t.Fatalf("launcher denied %s", name)
		}
	}
}

// Owner HTTP management (pairing, journal directory) also needs the launcher
// scope; a gameplay page's request with the owner cookie alone is refused.
func TestAuditOwnerHTTPNeedsLauncherScope(t *testing.T) {
	s := twoCommanderServer(t)
	client := ownerClient(t, s)
	for _, path := range []string{"/owner/journal", "/owner/remote"} {
		req, _ := http.NewRequest("POST", s.URL()+path, strings.NewReader(`{"action":"status"}`))
		req.Header.Set("Origin", s.URL())
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("%s without launcher scope = %d", path, res.StatusCode)
		}
	}
	if status, _, _ := remotePost(t, client, s.URL()+"/owner/journal", s.URL(), map[string]any{"action": "status"}); status != http.StatusOK {
		t.Fatalf("launcher journal status = %d", status)
	}
}

// A session boundary of the same commander (the display briefly has no
// commander, e.g. a new Fileheader) is not a commander switch; a different
// known commander is, even across such a gap.
func TestAuditAccountEpochAdvancesOnlyForADifferentCommander(t *testing.T) {
	var a accountEpochState
	a.init()
	for _, step := range []struct {
		fid     string
		advance bool
	}{{"", false}, {"SYNTHETIC-A", false}, {"", false}, {"SYNTHETIC-A", false}, {"SYNTHETIC-B", true}, {"", false}, {"SYNTHETIC-A", true}, {"SYNTHETIC-A", false}} {
		if got := a.observe(step.fid); got != step.advance {
			t.Fatalf("observe(%q) advanced=%v, want %v", step.fid, got, step.advance)
		}
	}
	if a.current() != 3 {
		t.Fatalf("epoch = %d, want 3", a.current())
	}
}
