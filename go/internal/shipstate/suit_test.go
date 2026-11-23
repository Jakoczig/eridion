package shipstate

import (
	"github.com/Jakoczig/eridion/go/internal/journal"
	"testing"
)

// TestAuditSuitIsNotShip: an on-foot or other non-ship LoadGame never turns
// a suit, SRV, fighter or taxi into the commander's ship. A verified real ship
// stays last-known; without one the ship is unknown.
func TestAuditSuitIsNotShip(t *testing.T) {
	feeder := func(m *Model) func(string) {
		seq := uint64(0)
		return func(raw string) {
			seq++
			e, err := journal.ParseLine([]byte(raw), journal.SourcePosition{FileID: "fixture", Offset: int64(seq * 100), Sequence: seq})
			if err != nil {
				t.Fatal(err)
			}
			m.ApplyEvent("A", e)
		}
	}
	for _, vehicle := range []struct{ symbol, kind string }{
		{"FlightSuit", "suit"}, {"ExplorationSuit_Class3", "suit"}, {"testbuggy", "srv"}, {"independent_fighter", "fighter"}, {"adder_taxi", "taxi"},
	} {
		m := NewModel()
		feed := feeder(m)
		feed(`{"event":"LoadGame","FID":"SYNTHETIC-A","Ship":"anaconda","ShipID":7,"ShipName":"REAL SHIP"}`)
		feed(`{"event":"Loadout","Ship":"anaconda","ShipID":7,"ShipName":"REAL SHIP","HullValue":1000,"CargoCapacity":64}`)
		feed(`{"event":"LoadGame","FID":"SYNTHETIC-A","Ship":"` + vehicle.symbol + `","ShipID":4294967297,"FuelLevel":1}`)
		v := m.View("A")
		if v.ShipType != "anaconda" || v.ShipName != "REAL SHIP" || v.Controlled != vehicle.kind {
			t.Fatalf("%s: view = %+v", vehicle.symbol, v)
		}
		if v.FuelMainT != nil {
			t.Fatalf("%s: vehicle fuel attributed to the ship", vehicle.symbol)
		}
		if m.LastKnownIdentity("A") != "7" || m.Identity("A") != "" {
			t.Fatalf("%s: identity live=%q last=%q", vehicle.symbol, m.Identity("A"), m.LastKnownIdentity("A"))
		}
		// A taxi/SRV Loadout-like record cannot replace the ship either.
		feed(`{"event":"Loadout","Ship":"` + vehicle.symbol + `","ShipID":4294967297}`)
		if v := m.View("A"); v.ShipType != "anaconda" {
			t.Fatalf("%s: Loadout replaced the ship: %+v", vehicle.symbol, v)
		}
		// Boarding the real ship again restores live identity.
		feed(`{"event":"Loadout","Ship":"anaconda","ShipID":7,"ShipName":"REAL SHIP"}`)
		if v := m.View("A"); v.Controlled != "ship" || m.Identity("A") != "7" {
			t.Fatalf("%s: return to ship: %+v", vehicle.symbol, v)
		}
	}

	// No prior real-ship evidence: unknown, not the suit.
	m := NewModel()
	feeder(m)(`{"event":"LoadGame","FID":"SYNTHETIC-A","Ship":"FlightSuit","ShipID":4294967297}`)
	if v := m.View("A"); v.ShipType != "" || v.Controlled != "suit" {
		t.Fatalf("suit became ship: %+v", v)
	}
	// Another commander's ship is never carried over.
	m = NewModel()
	feed := feeder(m)
	feed(`{"event":"LoadGame","FID":"SYNTHETIC-A","Ship":"anaconda","ShipID":7}`)
	e, _ := journal.ParseLine([]byte(`{"event":"LoadGame","FID":"SYNTHETIC-B","Ship":"FlightSuit"}`), journal.SourcePosition{FileID: "fixture", Offset: 1000, Sequence: 2})
	m.ApplyEvent("B", e)
	if v := m.View("B"); v.ShipType != "" {
		t.Fatalf("ship leaked across commanders: %+v", v)
	}
	// An unfamiliar genuine hull is not rejected.
	m = NewModel()
	feeder(m)(`{"event":"LoadGame","FID":"SYNTHETIC-A","Ship":"unfamiliar_future_hull","ShipID":9}`)
	if v := m.View("A"); v.ShipType != "unfamiliar_future_hull" || v.Controlled != "ship" {
		t.Fatalf("future hull rejected: %+v", v)
	}
}

// Task 16: a later Loadout of the same ship that omits ShipName keeps the
// known custom name; an explicit empty name clears it; another ship never
// inherits it.
func TestAuditShipNamePreserved(t *testing.T) {
	m := NewModel()
	seq := uint64(0)
	feed := func(raw string) {
		seq++
		e, err := journal.ParseLine([]byte(raw), journal.SourcePosition{FileID: "fixture", Offset: int64(seq * 100), Sequence: seq})
		if err != nil {
			t.Fatal(err)
		}
		m.ApplyEvent("A", e)
	}
	feed(`{"event":"LoadGame","FID":"SYNTHETIC-A","Ship":"asp","ShipID":7}`)
	feed(`{"event":"Loadout","Ship":"asp","ShipID":7,"ShipName":"Known Name"}`)
	feed(`{"event":"Loadout","Ship":"asp","ShipID":7}`)
	if v := m.View("A"); v.ShipName != "Known Name" {
		t.Fatalf("name lost when omitted: %+v", v)
	}
	feed(`{"event":"Loadout","Ship":"asp","ShipID":7,"ShipName":""}`)
	if v := m.View("A"); v.ShipName != "" {
		t.Fatalf("explicit clear ignored: %+v", v)
	}
	feed(`{"event":"Loadout","Ship":"asp","ShipID":7,"ShipName":"Again"}`)
	feed(`{"event":"Loadout","Ship":"python","ShipID":8}`)
	if v := m.View("A"); v.ShipName != "" {
		t.Fatalf("another ship inherited the name: %+v", v)
	}
}
