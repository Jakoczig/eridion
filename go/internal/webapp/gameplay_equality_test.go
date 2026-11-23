package webapp

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// gameplayFixtureMessages gives each gameplay RPC a realistic request for the
// g2e fixture; anything not listed is called with an empty object.
var gameplayFixtureMessages = map[string]any{
	"getGoOperations":    map[string]any{"kind": "fleet", "limit": 30},
	"exportGoOperations": map[string]any{"kind": "fleet", "limit": 30},
	"getGoSystem":        map[string]any{"name": "Fixture Alpha"},
	"lookupGoSystem":     map[string]any{"name": "Fixture Alpha"},
	"getGoBGSTally":      map[string]any{"context": "missions"},
	"exportGoBGSTally":   map[string]any{"context": "missions"},
}

// Mutating gameplay actions are compared separately (they change state).
var gameplayActions = setOf("importGoItinerary", "editGoItinerary", "deleteGoItinerary", "copyGoItineraryNext", "confirmGoOrganicSale", "refreshGoCommunity", "refreshGoCommunityGoals")

// principalSpecific lists replies that legitimately describe the connection
// itself rather than gameplay data.
var principalSpecific = setOf("hostInfo")

// Audit F7 (Task 13 / D1): the main gameplay display and a paired display
// receive identical replies for every gameplay read, for the same
// host-authorized commander, and the same outcome for gameplay actions.
func TestAuditGameplayDisplaysReceiveIdenticalData(t *testing.T) {
	s, _, viewer, remoteURL, _ := g2eFixture(t)
	main := dialScoped(t, s, "")
	paired := pairedSocket(t, viewer, remoteURL)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	normalize := func(r map[string]json.RawMessage) []byte {
		delete(r, "requestId")
		delete(r, "epoch")
		b, _ := json.Marshal(r)
		return b
	}
	succeeded := 0
	for name := range gameplayRPC {
		if gameplayActions[name] || principalSpecific[name] {
			continue
		}
		message, ok := gameplayFixtureMessages[name]
		if !ok {
			message = map[string]any{}
		}
		m := normalize(rpcWithMessage(t, ctx, main, name, message))
		p := normalize(rpcWithMessage(t, ctx, paired, name, message))
		if !bytes.Equal(m, p) {
			t.Errorf("%s differs between displays:\n main:   %.600s\n paired: %.600s", name, m, p)
		}
		var r map[string]json.RawMessage
		_ = json.Unmarshal(m, &r)
		if len(r["error"]) == 0 {
			succeeded++
		}
	}
	// The comparison must cover real data, not two identical errors.
	for _, name := range []string{"getJournalSummary", "getGoShip", "getGoNavigation", "getGoExploration", "getGoWorkspace", "getGoCommanderOverview", "getGoOperations", "getGoFieldView", "getGoProviderGameplay"} {
		r := rpcWithMessage(t, ctx, main, name, gameplayFixtureMessages[name])
		if len(r["error"]) != 0 || len(r["message"]) < 3 {
			t.Errorf("fixture lacks data for %s: %s %s", name, r["error"], r["message"])
		}
	}
	if succeeded < 15 {
		t.Errorf("only %d gameplay reads returned data", succeeded)
	}

	// Actions: the same request succeeds on both displays, and the same
	// invalid request fails with the same message.
	for _, conn := range []struct {
		name string
		c    *websocket.Conn
	}{{"main", main}, {"paired", paired}} {
		r := rpcWithMessage(t, ctx, conn.c, "importGoItinerary", map[string]any{"name": "Route " + conn.name, "csv": "system\nAlpha\nBeta\n"})
		if len(r["error"]) != 0 {
			t.Errorf("%s display could not import an itinerary: %s", conn.name, r["error"])
		}
	}
	bad := map[string]any{"name": "Bad", "csv": "not a route"}
	if m, p := rpcWithMessage(t, ctx, main, "importGoItinerary", bad), rpcWithMessage(t, ctx, paired, "importGoItinerary", bad); string(m["error"]) != string(p["error"]) || len(m["error"]) == 0 {
		t.Errorf("invalid import outcome differs: main %s / paired %s", m["error"], p["error"])
	}
}
