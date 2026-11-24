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
				name = "rpc." + n
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
