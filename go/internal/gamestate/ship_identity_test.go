package gamestate

import (
	"context"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/history"
)

// TestAuditShipIdentityEverywhere: Commander, account cards and the ship
// statistics header show one resolved main-ship identity. An on-foot relog
// never turns the suit into the ship, a Loadout omitting the name keeps it,
// and a fresh Frontier current ship wins under D0.
func TestAuditShipIdentityEverywhere(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-22T10:00:00Z","FID":"SYNTHETIC-A","Commander":"Fixture","Ship":"anaconda","ShipID":7}`)
	appendEvent(t, dir, `{"event":"Loadout","timestamp":"2026-11-22T10:00:01Z","Ship":"anaconda","ShipID":7,"ShipName":"Known Name","Modules":[]}`)
	appendEvent(t, dir, `{"event":"Loadout","timestamp":"2026-11-22T10:00:02Z","Ship":"anaconda","ShipID":7,"Modules":[]}`)
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-22T11:00:00Z","FID":"SYNTHETIC-A","Commander":"Fixture","Ship":"FlightSuit","ShipID":4294967297}`)
	writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
	s, _, stop := recordingService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 4 })

	check := func(wantName, wantModel string) {
		t.Helper()
		overview, err := s.CommanderOverview(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		owner := s.OwnerShipIdentity()
		cards := s.AccountCards()
		if overview["shipName"] != wantName || overview["shipType"] != wantModel {
			t.Fatalf("Commander = %v / %v, want %s / %s", overview["shipName"], overview["shipType"], wantName, wantModel)
		}
		if owner["shipName"] != wantName || owner["shipType"] != wantModel {
			t.Fatalf("statistics header = %v", owner)
		}
		if len(cards) != 1 || cards[0].ShipName != wantName || cards[0].ShipType != wantModel {
			t.Fatalf("account card = %+v", cards)
		}
	}
	check("Known Name", "Anaconda")

	store, _ := s.History()
	now := time.Now()
	if err := store.SaveExternal(context.Background(), history.ExternalFacts{Provider: "frontier", Entity: "commander", EntityID: "SYNTHETIC-A", FID: "SYNTHETIC-A", IdentityVerified: true, Fetched: now, Expires: now.Add(5 * time.Minute), Fields: map[string]any{"ship": map[string]any{"id": "7", "model": "anaconda", "name": "Frontier Name", "currentReference": true}}}); err != nil {
		t.Fatal(err)
	}
	check("Frontier Name", "Anaconda")
}
