package providers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/journal"
)

func decodeNumbers(t *testing.T, raw string) map[string]any {
	t.Helper()
	d := json.NewDecoder(strings.NewReader(raw))
	d.UseNumber()
	var v map[string]any
	if err := d.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// Task 17 / F24: Inara pilot ranks are normalized into the shared rank
// representation with their attributable progress, and out-of-range levels
// are dropped (pilot categories 0-13, Empire/Federation navy 0-14).
func TestAuditInaraRankNormalization(t *testing.T) {
	data := decodeNumbers(t, `{"events":[{"eventStatus":200,"eventData":{"commanderName":"Fixture CMDR","commanderRanksPilot":[
		{"rankName":"combat","rankValue":5,"rankProgress":0.35},
		{"rankName":"exploration","rankValue":13,"rankProgress":1},
		{"rankName":"trade","rankValue":14,"rankProgress":0.1},
		{"rankName":"empire","rankValue":14,"rankProgress":0.5},
		{"rankName":"federation","rankValue":15},
		{"rankName":"cqc","rankValue":-1},
		{"rankName":"soldier","rankValue":2,"rankProgress":1.5}]}}]}`)
	x, err := normalizedCommander("inara", "profile", "SYNTHETIC-A", "Fixture CMDR", data, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	levels, _ := json.Marshal(x.Fields["ranks"])
	progress, _ := json.Marshal(x.Fields["rankProgress"])
	if string(levels) != `[{"name":"combat","value":"5"},{"name":"exploration","value":"13"},{"name":"empire","value":"14"},{"name":"soldier","value":"2"}]` {
		t.Fatalf("levels = %s", levels)
	}
	if string(progress) != `[{"name":"combat","value":"35"},{"name":"exploration","value":"100"},{"name":"empire","value":"50"}]` {
		t.Fatalf("progress = %s", progress)
	}
}

// Task 17 / F24: the Inara rank upload uses the documented bounds: pilot
// ranks 0-13, Empire/Federation navy ranks 0-14.
func TestAuditInaraRankUploadBounds(t *testing.T) {
	m, _ := testManager(t, func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network"); return nil, nil })
	cred := Credential{APIKey: "fixture-key", Commander: "Fixture CMDR"}
	encode := func(raw string) error {
		e := event(t, raw)
		m.mu.Lock()
		defer m.mu.Unlock()
		_, _, _, err := m.encode("inara", e, object(e.Raw), cred)
		return err
	}
	for raw, ok := range map[string]bool{
		`{"timestamp":"2026-11-14T12:00:00Z","event":"Rank","Combat":13,"Empire":14}`: true,
		`{"timestamp":"2026-11-14T12:00:00Z","event":"Rank","Federation":14}`:         true,
		`{"timestamp":"2026-11-14T12:00:00Z","event":"Rank","Combat":14}`:             false,
		`{"timestamp":"2026-11-14T12:00:00Z","event":"Rank","Exobiologist":14}`:       false,
		`{"timestamp":"2026-11-14T12:00:00Z","event":"Rank","Empire":15}`:             false,
	} {
		if err := encode(raw); (err == nil) != ok {
			t.Errorf("%s: err=%v, want ok=%v", raw, err, ok)
		}
	}
	_ = journal.Live
}
