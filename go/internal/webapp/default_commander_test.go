package webapp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/providers"
)

// Default B with reader-active A pins B for providers and displays across a
// restart, and single mode with a default never asks for a choice again.
func TestAuditDefaultCommanderRPC(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	writeGameEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-01-01T00:00:01Z","FID":"SYNTHETIC-B"}`)
	writeGameEvent(t, dir, `{"event":"Location","timestamp":"2026-01-01T00:00:02Z","StarSystem":"B"}`)
	writeGameEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-01-01T00:00:03Z","FID":"SYNTHETIC-A"}`)
	writeGameEvent(t, dir, `{"event":"Location","timestamp":"2026-01-01T00:00:04Z","StarSystem":"A"}`)
	start := func() *Server {
		s, e := StartLocalWithOptions(context.Background(), fixtureFS(), Options{StorageDir: profile, JournalDir: dir, EnableJournal: true, EnableNavigation: true, EnableExploration: true, EnableDashboard: true, EnableParity: true, ProviderOptions: providers.Options{Vault: &accountTestVault{}, Manual: true}})
		if e != nil {
			t.Fatal(e)
		}
		deadline := time.Now().Add(3 * time.Second)
		for v := s.game.Snapshot(); v.ImportInProgress || v.EventsRead != 4; v = s.game.Snapshot() {
			if time.Now().After(deadline) {
				t.Fatal("journal replay not ready")
			}
			time.Sleep(time.Millisecond)
		}
		return s
	}
	s := start()
	if !s.game.DefaultRequired() || !s.accountChoiceRequired() {
		t.Fatal("ambiguous first run did not request one launcher default")
	}
	raw, _ := json.Marshal(map[string]any{"multiple": false, "default": "SYNTHETIC-UNVERIFIED"})
	if _, e := s.accountRPC(context.Background(), "setGoAccounts", raw); e == nil {
		t.Fatal("unverified default accepted")
	}
	raw, _ = json.Marshal(map[string]any{"multiple": false, "default": "SYNTHETIC-B"})
	v, e := s.accountRPC(context.Background(), "setGoAccounts", raw)
	if e != nil {
		t.Fatal(e)
	}
	view := v.(map[string]any)
	if view["defaultRequired"] != false || view["choiceRequired"] != false || s.game.FieldCommander() != "SYNTHETIC-B" {
		t.Fatalf("default not applied: %v", view)
	}
	if e = s.providers.ActivateAccount("SYNTHETIC-A", false); e == nil {
		t.Fatal("provider namespace followed reader-active A")
	}
	s.Close(context.Background())

	s = start()
	defer s.Close(context.Background())
	if s.accountChoiceRequired() || s.game.FieldCommander() != "SYNTHETIC-B" {
		t.Fatal("restart lost default or asked again")
	}
	if e = s.providers.ActivateAccount("SYNTHETIC-A", false); e == nil {
		t.Fatal("startup provider pin did not use the default")
	}
	if e = s.providers.ActivateAccount("SYNTHETIC-B", false); e != nil {
		t.Fatal(e)
	}
}
