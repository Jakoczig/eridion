package gamestate

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/journal"
)

func providerRecordingService(t *testing.T, profile, dir string) (*Service, func() []string, func()) {
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

// Providers stay blocked while several commanders are cached without a
// default; adopting the sole commander unblocks exactly that commander.
func TestAuditProviderBlockFollowsDefaultResolution(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-22T10:00:00Z","FID":"SYNTHETIC-A"}`)
	s, pins, stop := providerRecordingService(t, profile, dir)
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 1 })
	if got := pins(); len(got) != 1 || got[0] != "SYNTHETIC-A" {
		t.Fatalf("sole commander adoption pins = %v", got)
	}
	stop()

	dir, profile = t.TempDir(), t.TempDir()
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-22T10:00:00Z","FID":"SYNTHETIC-A"}`)
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-22T11:00:00Z","FID":"SYNTHETIC-B"}`)
	s, pins, stop = providerRecordingService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 2 })
	if got := pins(); len(got) != 0 || !s.DefaultRequired() {
		t.Fatalf("ambiguous commanders unblocked providers: %v (default required %v)", got, s.DefaultRequired())
	}
}
