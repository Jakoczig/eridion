package webapp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/providers"
)

func searchServer(t *testing.T, starPos bool, id64 string) (*Server, *atomic.Int32) {
	t.Helper()
	dir := t.TempDir()
	writeGameEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-01-01T00:00:01Z","FID":"SYNTHETIC-A"}`)
	location := `{"event":"Location","timestamp":"2026-01-01T00:00:02Z","StarSystem":"Fixture Coordless","SystemAddress":9007199254740993`
	if starPos {
		location += `,"StarPos":[7,8,9]`
	}
	writeGameEvent(t, dir, location+`}`)
	var calls atomic.Int32
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		raw := `{"id":77,"id64":` + id64 + `,"name":"Fixture Coordless","bodies":[],"stations":[]}`
		if r.URL.Path == "/api-v1/system" {
			raw = `{"id":77,"id64":` + id64 + `,"name":"Fixture Coordless","coords":{"x":3,"y":4,"z":5}}`
		}
		if r.URL.Path == "/api-system-v1/stations" {
			raw = `{"id":77,"name":"Fixture Coordless","stations":[]}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(raw))}, nil
	})}
	s, e := StartLocalWithOptions(context.Background(), fixtureFS(), Options{StorageDir: t.TempDir(), JournalDir: dir, EnableJournal: true, EnableNavigation: true, EnableExploration: true, EnableDashboard: true, EnableParity: true, CommunityClient: client, ProviderOptions: providers.Options{Vault: &accountTestVault{}, Manual: true}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close(context.Background()) })
	deadline := time.Now().Add(5 * time.Second)
	for v := s.game.Snapshot(); v.ImportInProgress || v.EventsRead != 2; v = s.game.Snapshot() {
		if time.Now().After(deadline) {
			t.Fatal("journal replay not ready")
		}
		time.Sleep(time.Millisecond)
	}
	if err := s.community.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	return s, &calls
}

func searchPoint(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(v)
	var out struct {
		Point map[string]any `json:"point"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out.Point
}

// Task 4 / F05: an explicit lookup for a journal-witnessed system without
// StarPos resolves its coordinates through the existing consented EDSM path,
// bound to the witnessed address; a different ID never merges; witnessed
// coordinates are never replaced and need no lookup.
func TestAuditSearchMissingCoordinates(t *testing.T) {
	s, calls := searchServer(t, false, "9007199254740993")
	ctx := context.Background()
	if p := searchPoint(t, s.systemView(ctx, "Fixture Coordless", false)); p["coordinates"] != nil || calls.Load() != 0 {
		t.Fatalf("passive read resolved or requested: %v (%d calls)", p, calls.Load())
	}
	p := searchPoint(t, s.systemView(ctx, "Fixture Coordless", true))
	coords, _ := p["coordinates"].([]any)
	if len(coords) != 3 || coords[0] != 3.0 || p["address"] != "9007199254740993" || p["coordinatesSource"] != "EDSM" {
		t.Fatalf("missing coordinates not resolved for the witnessed identity: %v", p)
	}

	s, _ = searchServer(t, false, "123456")
	if p := searchPoint(t, s.systemView(ctx, "Fixture Coordless", true)); p["coordinates"] != nil || p["address"] != "9007199254740993" {
		t.Fatalf("identity mismatch merged: %v", p)
	}

	s, calls = searchServer(t, true, "9007199254740993")
	p = searchPoint(t, s.systemView(ctx, "Fixture Coordless", false))
	coords, _ = p["coordinates"].([]any)
	if len(coords) != 3 || coords[0] != 7.0 || calls.Load() != 0 {
		t.Fatalf("witnessed coordinates missing or a lookup was made: %v (%d calls)", p, calls.Load())
	}
	p = searchPoint(t, s.systemView(ctx, "Fixture Coordless", true))
	coords, _ = p["coordinates"].([]any)
	if len(coords) != 3 || coords[0] != 7.0 || p["coordinatesSource"] == "EDSM" {
		t.Fatalf("witnessed coordinates replaced: %v", p)
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Audit F8 / G2E: an already-open gameplay display learns of an EDSM consent
// change made in the launcher without waiting for its 15-second poll, so it
// can drop community data immediately.
func TestAuditCommunityConsentChangeReachesOpenDisplays(t *testing.T) {
	s, _ := searchServer(t, true, "9007199254740993")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	main := dialScoped(t, s, "")
	// Mark the connection as a community consumer, as the map does.
	if r := rpcWithMessage(t, ctx, main, "getGoCommunity", map[string]any{}); len(r["error"]) != 0 {
		t.Fatal(string(r["error"]))
	}
	if err := s.community.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	for {
		_, raw, err := main.Read(ctx)
		if err != nil {
			t.Fatal("no goCommunityChanged after the consent change:", err)
		}
		var frame struct {
			Name    string `json:"name"`
			Message struct {
				Enabled bool `json:"enabled"`
			} `json:"message"`
		}
		if json.Unmarshal(raw, &frame) == nil && frame.Name == "goCommunityChanged" {
			if frame.Message.Enabled {
				t.Fatal("change reported as enabled")
			}
			return
		}
	}
}
