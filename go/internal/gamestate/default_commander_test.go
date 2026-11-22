package gamestate

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/journal"
	"github.com/Jakoczig/eridion/go/internal/privatefs"
)

func writeAccounts(t *testing.T, profile, raw string) {
	t.Helper()
	if err := privatefs.Prepare(profile); err != nil {
		t.Fatal(err)
	}
	if err := privatefs.Write(profile, accountSettingsFile, []byte(raw)); err != nil {
		t.Fatal(err)
	}
}

// recordingService starts a durable service and records every display
// account it publishes, so a transient wrong-account frame is detectable.
func recordingService(t *testing.T, profile, dir string) (*Service, func() []string, func()) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	s, err := NewService(Options{StorageDir: profile, Directory: dir, EnableParity: true, Discovery: journal.DiscoveryOptions{Roots: []string{}}, PollInterval: 5 * time.Millisecond, DisplayAccount: func(fid string, _ bool) {
		mu.Lock()
		seen = append(seen, fid)
		mu.Unlock()
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	return s, func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), seen...) }, func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("service did not stop")
		}
	}
}

func TestAuditDefaultCommander(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-20T10:00:00Z","FID":"SYNTHETIC-B"}`)
	appendEvent(t, dir, `{"event":"Location","timestamp":"2026-11-20T10:00:01Z","StarSystem":"Fixture B","SystemAddress":2}`)
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-22T10:00:00Z","FID":"SYNTHETIC-A"}`)
	appendEvent(t, dir, `{"event":"Location","timestamp":"2026-11-22T10:00:01Z","StarSystem":"Fixture A","SystemAddress":1}`)
	writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-B"}`)

	s, seen, stop := recordingService(t, profile, dir)
	if s.FieldCommander() != "SYNTHETIC-B" || s.PinnedAccount() != "SYNTHETIC-B" {
		t.Fatal("cold start did not resolve the default before import")
	}
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 4 })
	if v := s.Snapshot(); v.CurrentSystem != "Fixture B" {
		t.Fatalf("reader-active A replaced default B: %+v", v)
	}
	for _, fid := range seen() {
		if fid != "SYNTHETIC-B" {
			t.Fatalf("transient display frame for %q: %v", fid, seen())
		}
	}
	// A later A event does not change the default.
	appendEvent(t, dir, `{"event":"FSDJump","timestamp":"2026-11-22T10:05:00Z","StarSystem":"Fixture A2","SystemAddress":3}`)
	waitState(t, s, func(v Summary) bool { return v.EventsRead == 5 })
	if s.Accounts().Default != "SYNTHETIC-B" || s.FieldCommander() != "SYNTHETIC-B" {
		t.Fatal("journal activity changed the default", s.Accounts())
	}
	// A temporary multi-mode selection is independent of the default.
	if err := s.SetAccounts(true, "SYNTHETIC-A"); err != nil {
		t.Fatal(err)
	}
	if s.FieldCommander() != "SYNTHETIC-A" || s.Accounts().Default != "SYNTHETIC-B" {
		t.Fatal("multi-mode selection overwrote the default", s.Accounts())
	}
	if err := s.SetAccounts(false, ""); err != nil {
		t.Fatal(err)
	}
	if s.FieldCommander() != "SYNTHETIC-B" {
		t.Fatal("leaving multi mode lost the default")
	}
	stop()

	// Offline restart: the journal is unavailable, the cached default still works.
	if err := os.Remove(sourceFile(dir)); err != nil {
		t.Fatal(err)
	}
	s, seen, stop = recordingService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress })
	if s.FieldCommander() != "SYNTHETIC-B" || s.Snapshot().CurrentSystem != "Fixture B" || s.DefaultRequired() {
		t.Fatalf("offline restart lost default B: %q %+v", s.FieldCommander(), s.Snapshot())
	}
	for _, fid := range seen() {
		if fid != "SYNTHETIC-B" {
			t.Fatalf("offline restart published %q", fid)
		}
	}
}

func TestAuditDefaultCommanderMigrationAndRecovery(t *testing.T) {
	t.Run("legacy explicit selection becomes default", func(t *testing.T) {
		profile := t.TempDir()
		writeAccounts(t, profile, `{"multiple":true,"selected":"SYNTHETIC-A"}`)
		got, err := LoadAccountSettings(profile)
		if err != nil || got.Default != "SYNTHETIC-A" || got.Selected != "SYNTHETIC-A" {
			t.Fatalf("%+v %v", got, err)
		}
	})
	t.Run("absent legacy settings", func(t *testing.T) {
		got, err := LoadAccountSettings(t.TempDir())
		if err != nil || got != (AccountSettings{}) {
			t.Fatalf("%+v %v", got, err)
		}
	})
	for name, raw := range map[string]string{"corrupt": `{"multiple":`, "unknown field": `{"multiple":false,"admin":true}`, "future version": `{"version":9}`, "unsafe fid": `{"defaultFID":"bad\nfid"}`} {
		t.Run(name, func(t *testing.T) {
			dir, profile := t.TempDir(), t.TempDir()
			appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-20T10:00:00Z","FID":"SYNTHETIC-A"}`)
			writeAccounts(t, profile, raw)
			s, stop := durableService(t, profile, dir)
			defer stop()
			waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 1 })
			if s.Accounts().Default != "" || !s.DefaultRequired() || s.AccountRecovery() == "" {
				t.Fatalf("damaged preference silently replaced: %+v", s.Accounts())
			}
			kept, err := privatefs.Read(profile, accountRecoveryFile)
			if err != nil || string(kept) != raw {
				t.Fatalf("damaged preference not preserved: %q %v", kept, err)
			}
			if err = s.SetDefaultAccount("SYNTHETIC-A"); err != nil || s.DefaultRequired() || s.AccountRecovery() != "" {
				t.Fatalf("launcher recovery failed: %v", err)
			}
		})
	}
	t.Run("sole commander adopted and persisted", func(t *testing.T) {
		dir, profile := t.TempDir(), t.TempDir()
		appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-20T10:00:00Z","FID":"SYNTHETIC-A"}`)
		s, stop := durableService(t, profile, dir)
		defer stop()
		waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 1 })
		got, err := LoadAccountSettings(profile)
		if err != nil || got.Default != "SYNTHETIC-A" || s.DefaultRequired() {
			t.Fatalf("%+v %v", got, err)
		}
	})
	t.Run("ambiguous first-run import requires one launcher choice", func(t *testing.T) {
		dir, profile := t.TempDir(), t.TempDir()
		appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-20T10:00:00Z","FID":"SYNTHETIC-A"}`)
		appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-21T10:00:00Z","FID":"SYNTHETIC-B"}`)
		s, stop := durableService(t, profile, dir)
		defer stop()
		waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 2 })
		if s.Accounts().Default != "" || !s.DefaultRequired() {
			t.Fatal("ambiguous identities were guessed from import order", s.Accounts())
		}
		if err := s.SetDefaultAccount("NOT-VERIFIED"); err == nil {
			t.Fatal("unverified default accepted")
		}
		if err := s.SetDefaultAccount("SYNTHETIC-A"); err != nil || s.DefaultRequired() || s.FieldCommander() != "SYNTHETIC-A" {
			t.Fatal("explicit default failed", err)
		}
		if _, err := os.Stat(filepath.Join(profile, accountSettingsFile)); err != nil {
			t.Fatal("default not persisted", err)
		}
	})
}
