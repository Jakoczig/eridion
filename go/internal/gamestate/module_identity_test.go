package gamestate

import (
	"context"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/history"
	"github.com/Jakoczig/eridion/go/internal/paritystate"
)

func workspaceModule(t *testing.T, s *Service, slot string) paritystate.Module {
	t.Helper()
	for _, m := range s.Workspace().Parity.Modules {
		if m.Slot == slot {
			return m
		}
	}
	t.Fatalf("no module in %s", slot)
	return paritystate.Module{}
}

// Task 16: a Frontier numeric module ID identifies a journal module only
// after commander, ship and slot are verified: fresh current-ship evidence
// for the displayed commander, the same ship ID as the journal's verified
// ship, the same slot and symbol.
func TestAuditFrontierModuleIDsReconcileVerifiedShipOnly(t *testing.T) {
	for _, tc := range []struct {
		name, shipID, want string
	}{{"same ship", "7", "exact_id"}, {"other ship", "8", "ambiguous_variant"}} {
		t.Run(tc.name, func(t *testing.T) {
			dir, profile := t.TempDir(), t.TempDir()
			appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-24T10:00:00Z","FID":"SYNTHETIC-A","Commander":"Fixture","Ship":"anaconda","ShipID":7}`)
			appendEvent(t, dir, `{"event":"Loadout","timestamp":"2026-11-24T10:00:01Z","Ship":"anaconda","ShipID":7,"ShipName":"Fixture","Modules":[{"Slot":"HugeHardpoint1","Item":"hpt_beamlaser_fixed_huge","On":true,"Priority":0,"Health":1,"Engineering":{"BlueprintName":"Weapon_Overcharged","Level":5,"Quality":1,"Modifiers":[]}}]}`)
			writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
			s, _, stop := recordingService(t, profile, dir)
			defer stop()
			waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 2 })
			if m := workspaceModule(t, s, "HugeHardpoint1"); m.Identity != "ambiguous_variant" {
				t.Fatalf("before Frontier evidence: %+v", m)
			}
			store, _ := s.History()
			now := time.Now()
			if err := store.SaveExternal(context.Background(), history.ExternalFacts{Provider: "frontier", Entity: "commander", EntityID: "SYNTHETIC-A", FID: "SYNTHETIC-A", IdentityVerified: true, Fetched: now, Expires: now.Add(5 * time.Minute), Fields: map[string]any{"ship": map[string]any{"id": tc.shipID, "model": "anaconda", "currentReference": true, "modules": []map[string]any{{"slot": "HugeHardpoint1", "item": "Hpt_BeamLaser_Fixed_Huge", "id": "129046189"}}}}}); err != nil {
				t.Fatal(err)
			}
			s.invalidateFrontierModules()
			if m := workspaceModule(t, s, "HugeHardpoint1"); m.Identity != tc.want || (tc.want == "exact_id" && m.Name != "Overloaded Beam Laser") {
				t.Fatalf("after Frontier evidence for ship %s: %+v", tc.shipID, m)
			}
		})
	}
}
