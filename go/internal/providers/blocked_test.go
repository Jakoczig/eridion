package providers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/journal"
)

// TestAuditProvidersBlockedWithoutCommander (Task 12 follow-up): while no
// default/selected commander is resolved, the provider manager is explicitly
// blocked: no account namespace is activated from the journal, nothing is
// queued or sent, reads are refused and the status says why. Pinning a real
// commander lifts the block. No sentinel FID is used.
func TestAuditProvidersBlockedWithoutCommander(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	calls := 0
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: http.Header{}}, nil
	})}
	m := New(context.Background(), t.TempDir(), Options{Client: client, Vault: &testVault{}, Now: func() time.Time { return now }, Manual: true})
	defer m.Close()
	session(t, m)
	if err := m.Configure("eddn", Config{Upload: true, Categories: []string{"travel"}}); err != nil {
		t.Fatal(err)
	}
	m.PinAccount("") // launcher choice pending
	if !m.Blocked() {
		t.Fatal("empty pin did not block providers")
	}
	m.Observe(event(t, `{"event":"Fileheader","gameversion":"4.1.0.0","build":"fixture"}`), journal.Live)
	m.Observe(event(t, `{"timestamp":"2026-11-14T12:00:00Z","event":"LoadGame","FID":"SYNTHETIC-A","Commander":"Fixture CMDR"}`), journal.Live)
	jump := event(t, `{"timestamp":"2026-11-14T12:00:00Z","event":"FSDJump","StarSystem":"Fixture","SystemAddress":9007199254740993,"StarPos":[1,2,3]}`)
	m.Observe(jump, journal.Live)
	m.dispatch("eddn")
	if calls != 0 {
		t.Fatalf("blocked manager sent %d requests", calls)
	}
	m.mu.Lock()
	activated, queued, pin := m.accountFID, len(m.states["eddn"].queue), m.pinnedAccount
	m.mu.Unlock()
	if activated != "" || queued != 0 || pin != "" {
		t.Fatalf("blocked manager activated %q / queued %d / pin %q", activated, queued, pin)
	}
	if _, err := m.Read(context.Background(), "edsm", "flightlog", ""); err == nil || !strings.Contains(err.Error(), "launcher") {
		t.Fatalf("blocked read = %v", err)
	}
	if err := m.ActivateAccount("SYNTHETIC-A", false); err == nil {
		t.Fatal("blocked manager activated an account")
	}
	for _, s := range m.Status() {
		if s.State != "awaiting_commander" {
			t.Fatalf("status %s = %s, want awaiting_commander", s.Capability.ID, s.State)
		}
	}

	// The launcher (or single-commander adoption) pins the commander.
	m.PinAccount("SYNTHETIC-A")
	if m.Blocked() {
		t.Fatal("pinning a commander did not lift the block")
	}
	m.Observe(event(t, `{"timestamp":"2026-11-14T12:00:00Z","event":"LoadGame","FID":"SYNTHETIC-A","Commander":"Fixture CMDR"}`), journal.Live)
	if err := m.Configure("eddn", Config{Upload: true, Categories: []string{"travel"}}); err != nil {
		t.Fatal(err)
	}
	m.Observe(jump, journal.Live)
	m.dispatch("eddn")
	if calls != 1 {
		t.Fatalf("unblocked live upload calls = %d, want 1", calls)
	}
}
