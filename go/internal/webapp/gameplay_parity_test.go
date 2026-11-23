package webapp

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func rpcWithMessage(t *testing.T, ctx context.Context, conn *websocket.Conn, name string, message any) map[string]json.RawMessage {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"requestId": "parity-" + name, "name": name, "message": message})
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
	for {
		_, raw, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var response map[string]json.RawMessage
		if json.Unmarshal(raw, &response) == nil && string(response["requestId"]) == `"parity-`+name+`"` {
			return response
		}
	}
}

func authorizationError(response map[string]json.RawMessage) bool {
	var message string
	_ = json.Unmarshal(response["error"], &message)
	return message == launcherOnlyError || strings.Contains(message, "owner") || strings.Contains(message, "Owner")
}

// TestAuditGameplayParity (Task 13 / F16, confirmed D1): every successfully
// paired display has the same gameplay RPCs as the main display; only launcher
// management stays on the local launcher.
func TestAuditGameplayParity(t *testing.T) {
	s, _, client, remoteURL, code := remoteFixture(t)
	if status, _, _ := remotePost(t, client, remoteURL+"/auth/pair", remoteURL, map[string]any{"code": code}); status != 200 {
		t.Fatal("pair", status)
	}
	viewer := pairedSocket(t, client, remoteURL)
	// The main gameplay display is an owner-session connection without the
	// launcher scope (dialOwner acts as the launcher).
	owner := dialScoped(t, s, "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for name := range gameplayRPC {
		v := rpcWithMessage(t, ctx, viewer, name, map[string]any{})
		if authorizationError(v) {
			t.Fatalf("paired display denied gameplay RPC %s: %s", name, v["error"])
		}
		o := rpcWithMessage(t, ctx, owner, name, map[string]any{})
		if authorizationError(o) {
			t.Fatalf("main display denied gameplay RPC %s: %s", name, o["error"])
		}
	}
	for name := range launcherRPC {
		for display, conn := range map[string]*websocket.Conn{"paired": viewer, "main": owner} {
			v := rpcWithMessage(t, ctx, conn, name, map[string]any{})
			var message string
			_ = json.Unmarshal(v["error"], &message)
			if message != launcherOnlyError || v["message"] != nil {
				t.Fatalf("%s display reached launcher RPC %s: %s", display, name, v["error"])
			}
		}
	}
}

// Every RPC the socket dispatches is classified exactly once, so a new RPC
// cannot silently become owner-only gameplay or open management to displays.
func TestAuditRPCClassificationComplete(t *testing.T) {
	raw, err := os.ReadFile("socket.go")
	if err != nil {
		t.Fatal(err)
	}
	// Windows checkouts may use CRLF; parse line structure independently.
	src := strings.ReplaceAll(string(raw), "\r\n", "\n")
	start := strings.Index(src, "switch request.Name {")
	end := strings.Index(src[start:], "default:\n\t\t\tresponse[\"error\"] = \"Not implemented")
	if start < 0 || end < 0 {
		t.Fatal("dispatch switch not found")
	}
	names := regexp.MustCompile(`"([a-zA-Z]+)"`).FindAllStringSubmatch(strings.Join(regexp.MustCompile(`(?m)^\t\tcase ([^\n]+):$`).FindAllString(src[start:start+end], -1), "\n"), -1)
	if len(names) < 40 {
		t.Fatalf("parsed only %d RPC names", len(names))
	}
	for _, m := range names {
		name := m[1]
		if gameplayRPC[name] == launcherRPC[name] {
			t.Fatalf("RPC %s is not classified exactly once", name)
		}
	}
	for name := range gameplayRPC {
		if launcherRPC[name] {
			t.Fatalf("RPC %s classified twice", name)
		}
	}
}

// The gameplay provider DTO carries no configuration, credentials or errors.
func TestAuditGameplayProviderDTO(t *testing.T) {
	s, _, client, remoteURL, code := remoteFixture(t)
	if status, _, _ := remotePost(t, client, remoteURL+"/auth/pair", remoteURL, map[string]any{"code": code}); status != 200 {
		t.Fatal("pair", status)
	}
	_ = s
	viewer := pairedSocket(t, client, remoteURL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	reply := rpcWithMessage(t, ctx, viewer, "getGoProviderGameplay", map[string]any{})
	if authorizationError(reply) {
		t.Fatalf("provider gameplay DTO denied: %s", reply["error"])
	}
	for _, forbidden := range []string{"config", "credential", "lastError", "queue", "apiKey", "token"} {
		if strings.Contains(strings.ToLower(string(reply["message"])), strings.ToLower(forbidden)) {
			t.Fatalf("provider gameplay DTO leaked %s: %s", forbidden, reply["message"])
		}
	}
}
