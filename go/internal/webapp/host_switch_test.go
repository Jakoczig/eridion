package webapp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/providers"
	"github.com/coder/websocket"
)

// Task 13 / contract 4 (owner decision 2026-11-23): a host-authorized
// commander switch moves every trusted display to the new commander without
// re-pairing, and invalidates everything delivered or requested under the
// previous commander.
func TestHostCommanderSwitchMovesTrustedDisplays(t *testing.T) {
	s, _, client, remoteURL, code := remoteFixture(t)
	if status, _, _ := remotePost(t, client, remoteURL+"/auth/pair", remoteURL, map[string]any{"code": code}); status != 200 {
		t.Fatal("pair", status)
	}
	s.remote.selectAccount("SYNTHETIC-A")
	old := pairedSocket(t, client, remoteURL)
	defer old.CloseNow()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if r := callRPC(t, ctx, old, "getLoadingStatus"); len(r["error"]) != 0 {
		t.Fatal("A grant unusable before the switch")
	}

	s.remote.hostSelectAccount("SYNTHETIC-B", false)

	// The previous-commander socket is closed: nothing requested under A is answered.
	_ = old.Write(ctx, websocket.MessageText, []byte(`{"name":"getJournalSummary","requestId":"after-switch"}`))
	for {
		_, raw, e := old.Read(ctx)
		if e != nil {
			if ctx.Err() != nil {
				t.Fatal("previous-commander socket was not actively closed", e)
			}
			break
		}
		if strings.Contains(string(raw), "after-switch") {
			t.Fatal("previous-commander socket answered after the host switch")
		}
	}
	// The same grant reconnects for B with no new pairing or prompt.
	if status, _, _ := remotePost(t, client, remoteURL+"/auth/session", remoteURL, nil); status != http.StatusOK {
		t.Fatal("trusted display did not follow the host switch", status)
	}
	next := pairedSocket(t, client, remoteURL)
	defer next.CloseNow()
	if r := callRPC(t, ctx, next, "getLoadingStatus"); len(r["error"]) != 0 {
		t.Fatal("followed grant unusable for B")
	}
	s.remote.mu.Lock()
	if len(s.remote.state.Devices) != 1 || s.remote.state.Devices[0].Account != tokenHash("SYNTHETIC-B") {
		t.Error("grant not moved to the host-selected commander")
	}
	s.remote.mu.Unlock()

	// And back again.
	s.remote.hostSelectAccount("SYNTHETIC-A", false)
	if status, _, _ := remotePost(t, client, remoteURL+"/auth/session", remoteURL, nil); status != http.StatusOK {
		t.Fatal("trusted display did not follow the switch back", status)
	}
}

// A grant left on another commander by an earlier build (the switch used to
// strand it) is moved by the next host-authorized selection, even when the
// host account itself did not change.
func TestHostSelectionMigratesStrandedGrant(t *testing.T) {
	s, _, _, _, _ := remoteFixture(t)
	m := s.remote
	token := strings.Repeat("c", 64)
	m.mu.Lock()
	m.state.Version, m.state.Account = 2, tokenHash("SYNTHETIC-B")
	m.state.Devices = []device{{Account: tokenHash("SYNTHETIC-A"), ID: strings.Repeat("d", 32), Name: "Stranded laptop", Expires: time.Now().Add(time.Hour), Hash: tokenHash(token)}}
	m.mu.Unlock()
	m.hostSelectAccount("SYNTHETIC-B", false)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.findLocked(token); !ok {
		t.Fatal("stranded grant not migrated by the host selection")
	}
}

// Grants whose original commander is unknown stay quarantined: a host switch
// moves trusted grants, it never proves ownership of ambiguous legacy ones.
func TestHostSwitchKeepsAmbiguousLegacyGrantQuarantined(t *testing.T) {
	s, _, _, _, _ := remoteFixture(t)
	m := s.remote
	token := strings.Repeat("e", 64)
	m.mu.Lock()
	m.legacyGrants = true
	m.state.Account = ""
	m.state.Devices = []device{{ID: strings.Repeat("f", 32), Name: "Legacy laptop", Expires: time.Now().Add(time.Hour), Hash: tokenHash(token)}}
	m.mu.Unlock()
	m.hostSelectAccount("SYNTHETIC-A", false)
	m.hostSelectAccount("SYNTHETIC-B", false)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state.Devices[0].Account != "" {
		t.Fatal("host switch assigned an ambiguous legacy grant")
	}
	if _, ok := m.findLocked(token); ok {
		t.Fatal("ambiguous legacy grant became active")
	}
}

// End to end: the launcher's commander selection (setGoAccounts) is the
// host-authorized switch that moves the paired display.
func TestLauncherSelectionMovesPairedDisplay(t *testing.T) {
	dir := t.TempDir()
	writeGameEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-01-01T00:00:01Z","FID":"SYNTHETIC-A"}`)
	writeGameEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-01-01T00:00:03Z","FID":"SYNTHETIC-B"}`)
	s, e := StartLocalWithOptions(context.Background(), fixtureFS(), Options{StorageDir: t.TempDir(), JournalDir: dir, EnableJournal: true, EnableNavigation: true, EnableExploration: true, EnableDashboard: true, EnableParity: true, ProviderOptions: providers.Options{Vault: &accountTestVault{}, Manual: true}})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close(context.Background())
	deadline := time.Now().Add(3 * time.Second)
	for v := s.game.Snapshot(); v.ImportInProgress || v.EventsRead != 2; v = s.game.Snapshot() {
		if time.Now().After(deadline) {
			t.Fatal("journal replay not ready")
		}
		time.Sleep(time.Millisecond)
	}
	selectCommander := func(fid string) {
		raw, _ := json.Marshal(map[string]any{"multiple": true, "selected": fid})
		if _, e := s.accountRPC(context.Background(), "setGoAccounts", raw); e != nil {
			t.Fatal(e)
		}
	}
	selectCommander("SYNTHETIC-A")
	owner := ownerClient(t, s)
	ownerAction(t, s, owner, map[string]any{"action": "enable", "address": "127.0.0.1", "port": 0})
	pairing := ownerAction(t, s, owner, map[string]any{"action": "pair"})
	var code string
	json.Unmarshal(pairing["code"], &code)
	s.remote.mu.Lock()
	_, err := s.remote.pairLocked(code, "Paired laptop")
	s.remote.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	selectCommander("SYNTHETIC-B")
	s.remote.mu.Lock()
	defer s.remote.mu.Unlock()
	if len(s.remote.state.Devices) != 1 || s.remote.state.Devices[0].Account != tokenHash("SYNTHETIC-B") || s.remote.state.Account != tokenHash("SYNTHETIC-B") {
		t.Fatal("launcher selection did not move the paired display to B")
	}
}
