package webapp

import "github.com/Jakoczig/eridion/go/internal/perfstats"

// recordFrame counts one outgoing socket frame and its bytes by message name
// (measurement only; see docs/testing/g3k-native-performance.md).
func recordFrame(value any, size int) {
	name := "reply"
	if m, ok := value.(map[string]any); ok {
		if n, ok := m["name"].(string); ok && n != "" {
			name = n
			if _, isReply := m["requestId"]; isReply {
				// Request names are client-controlled: count only known RPCs
				// so a client cannot grow the counter set (review R2).
				if gameplayRPC[n] || launcherRPC[n] {
					name = "rpc." + n
				} else {
					name = "rpc.other"
				}
			}
		}
	}
	perfstats.Add("ws.frames", 1)
	perfstats.Add("ws.bytes", int64(size))
	perfstats.Add("ws.frames."+name, 1)
	perfstats.Add("ws.bytes."+name, int64(size))
}

// recordShared counts a frame whose encoding was reused from another
// connection.
func recordShared(name string) { perfstats.Add("ws.shared."+name, 1) }
