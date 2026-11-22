package webapp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// D1: operations reads and previewed exports are gameplay on every trusted
// display; source backfill is launcher maintenance.
func TestG2LPairedOperationsMatchMainDisplay(t *testing.T) {
	_, _, viewer, url, _ := g2eFixture(t)
	c := pairedSocket(t, viewer, url)
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, name := range []string{"getGoOperations", "exportGoOperations"} {
		if r := callRPC(t, ctx, c, name); strings.Contains(string(r["error"]), "owner") || string(r["error"]) == `"`+launcherOnlyError+`"` {
			t.Fatalf("paired display denied %s: %s", name, r["error"])
		}
	}
	if r := callRPC(t, ctx, c, "backfillGoOperationsSource"); string(r["error"]) != `"`+launcherOnlyError+`"` {
		t.Fatalf("paired display reached launcher maintenance: %s", r["error"])
	}
}

func TestG2LOwnerExportNeedsPreviewAcknowledgment(t *testing.T) {
	s, _, _, _, _ := g2eFixture(t)
	if _, e := s.operationsRPC(context.Background(), "exportGoOperations", json.RawMessage(`{"kind":"fleet"}`)); e == nil {
		t.Fatal("private export without acknowledgment")
	}
	preview, e := s.operationsRPC(context.Background(), "getGoOperations", json.RawMessage(`{"kind":"fleet"}`))
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(preview)
	var hash struct{ PreviewHash string }
	json.Unmarshal(b, &hash)
	value, e := s.operationsRPC(context.Background(), "exportGoOperations", json.RawMessage(fmt.Sprintf(`{"kind":"fleet","acknowledge":true,"previewHash":%q}`, hash.PreviewHash)))
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(value.(map[string]string)["csv"], "observed_utc") {
		t.Fatal(value)
	}
}
