package navcontext

import (
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/journal"
	"github.com/Jakoczig/eridion/go/internal/paritystate"
)

// G3L: Status reports only visible mode changes, and Deadline names the one
// instant the view changes by time alone, so delivery needs no sampling.
func TestG3LStatusChangeAndDeadline(t *testing.T) {
	m := New()
	feed(t, m, "a", `{"event":"LoadGame","timestamp":"2026-01-01T00:00:01Z","FID":"a"}`, journal.Live)
	feed(t, m, "a", `{"event":"Location","timestamp":"2026-01-01T00:00:02Z","StarSystem":"Fixture","SystemAddress":1}`, journal.Live)
	at, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:03Z")
	if !m.Deadline("a", at).IsZero() {
		t.Fatal("idle view has a time deadline")
	}
	v := paritystate.StatusView{State: "known", ObservedAt: "2026-01-01T00:00:03Z", Flags: &paritystate.StatusFlags{InMainShip: true}, Flags2: &paritystate.StatusFlags2{FSDHyperdriveCharging: true}}
	if !m.Status("a", v) {
		t.Fatal("charge not reported as a change")
	}
	if m.Status("a", v) {
		t.Fatal("repeated identical Status reported as a change")
	}
	v.ObservedAt = "2026-01-01T00:00:03.5Z"
	if m.Status("a", v) {
		t.Fatal("unchanged charge at a later time reported as a change")
	}
	// The continuing charge extended its window to the latest observation.
	ext := at.Add(500 * time.Millisecond)
	if d := m.Deadline("a", at.Add(time.Second)); !d.Equal(ext.Add(modeWindow)) {
		t.Fatalf("deadline %v", d)
	}
	if m.View("a", ext.Add(modeWindow-time.Nanosecond)).Mode != "charge" || m.View("a", ext.Add(modeWindow)).Mode != "idle" {
		t.Fatal("mode window boundary")
	}
	if !m.Deadline("a", ext.Add(modeWindow)).IsZero() {
		t.Fatal("expired mode keeps a deadline (would loop)")
	}
	if d := m.Deadline("a", at); !d.Equal(ext) {
		t.Fatalf("future mode deadline %v", d)
	}
	v.ObservedAt = "2026-01-01T00:00:04Z"
	v.Flags2.FSDHyperdriveCharging = false
	if !m.Status("a", v) || !m.Deadline("a", at.Add(2*time.Second)).IsZero() {
		t.Fatal("cancel not reported or deadline kept")
	}
	stale := v
	stale.ObservedAt = "2026-01-01T00:00:02.5Z"
	stale.Flags2 = &paritystate.StatusFlags2{FSDHyperdriveCharging: true}
	if m.Status("a", stale) {
		t.Fatal("delayed Status older than the last observation changed the mode")
	}
}
