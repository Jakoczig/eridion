package gamestate

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/journal"
)

// pinRecorder starts a durable service that records every provider pin.
func pinRecorder(t *testing.T, profile, dir string) (*Service, func() []string, func()) {
	t.Helper()
	var mu sync.Mutex
	var pins []string
	s, err := NewService(Options{StorageDir: profile, Directory: dir, EnableParity: true, Discovery: journal.DiscoveryOptions{Roots: []string{}}, PollInterval: 5 * time.Millisecond, ProviderAccount: func(fid string) {
		mu.Lock()
		pins = append(pins, fid)
		mu.Unlock()
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	return s, func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), pins...) }, func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("service did not stop")
		}
	}
}

// Audit F4 (Task 12): a stored default, or a legacy selection migrated to the
// default, that is not among the verified commanders once history is complete
// asks for the launcher choice and keeps providers blocked. It is never
// silently replaced by another commander.
func TestAuditUnverifiedDefaultRequiresLauncherChoice(t *testing.T) {
	for name, raw := range map[string]string{
		"stored default":           `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-GONE"}`,
		"migrated legacy selected": `{"multiple":false,"selected":"SYNTHETIC-GONE"}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir, profile := t.TempDir(), t.TempDir()
			appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-20T10:00:00Z","FID":"SYNTHETIC-A"}`)
			writeAccounts(t, profile, raw)
			s, pins, stop := pinRecorder(t, profile, dir)
			defer stop()
			waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 1 })
			if !s.DefaultRequired() || s.AccountRecovery() == "" {
				t.Fatalf("unverified default accepted: required=%v recovery=%q %+v", s.DefaultRequired(), s.AccountRecovery(), s.Accounts())
			}
			if s.PinnedAccount() != "" {
				t.Fatalf("providers pinned to an unverified commander: %q", s.PinnedAccount())
			}
			if p := pins(); len(p) == 0 || p[len(p)-1] != "" {
				t.Fatalf("provider block not applied: %v", p)
			}
			if s.Accounts().Default == "SYNTHETIC-A" {
				t.Fatal("sole commander silently replaced the stored default")
			}
			if err := s.SetDefaultAccount("SYNTHETIC-A"); err != nil || s.DefaultRequired() || s.AccountRecovery() != "" || s.PinnedAccount() != "SYNTHETIC-A" {
				t.Fatalf("launcher choice did not resolve: %v %q", err, s.PinnedAccount())
			}
		})
	}
}

// Audit F6 (Task 12): the sole-commander default is adopted only once the
// catch-up import is complete, never from a cache that has not yet seen the
// journals written while ERIDION was closed.
func TestAuditDefaultAdoptedOnlyAfterCatchUp(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-20T10:00:00Z","FID":"SYNTHETIC-A"}`)
	s, stop := durableService(t, profile, dir)
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 1 })
	stop()
	// No stored preference; meanwhile commander B played.
	if err := os.Remove(filepath.Join(profile, accountSettingsFile)); err != nil {
		t.Fatal(err)
	}
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-21T10:00:00Z","FID":"SYNTHETIC-B"}`)
	s, stop = durableService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 2 })
	if d := s.Accounts().Default; d != "" || !s.DefaultRequired() {
		t.Fatalf("default adopted from the cache before catch-up: %q required=%v", d, s.DefaultRequired())
	}
}
