package gamestate

import (
	"context"
	"os"
	"testing"
)

const suitLoadoutLine = `{"event":"SuitLoadout","timestamp":"2026-11-22T10:00:05Z","SuitID":1700217809818876,"SuitName":"explorationsuit_class3","SuitName_Localised":"Artemis Suit","SuitMods":[],"LoadoutID":4293000005,"LoadoutName":"Surveyor","Modules":[{"SlotName":"PrimaryWeapon1","SuitModuleID":1700217863661544,"ModuleName":"wpn_m_assaultrifle_kinetic_fauto","ModuleName_Localised":"Karma AR-50","Class":2,"WeaponMods":[]}]}`

// Audit F3 (Task 14): known Exosuit equipment is durable history evidence.
// A rebuild from history (cache upgrade, rebuildGoHistory, restore) with the
// original journal file gone still yields the known suit and loadout.
func TestAuditExosuitSurvivesHistoryRebuild(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-22T10:00:00Z","FID":"SYNTHETIC-A","Commander":"Fixture","Ship":"FlightSuit","ShipID":4294967297}`)
	appendEvent(t, dir, suitLoadoutLine)
	writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
	s, stop := durableService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 2 })
	if v := s.Workspace().Exosuit; v.State != "known" || v.Current == nil {
		t.Fatalf("live reducer did not see the loadout: %+v", v)
	}
	if err := os.Remove(sourceFile(dir)); err != nil {
		t.Fatal(err)
	}
	if err := s.rebuildDerived(context.Background()); err != nil {
		t.Fatal(err)
	}
	v := s.Workspace().Exosuit
	if v.State != "known" || v.Current == nil || v.Current.SuitID != "1700217809818876" || v.Current.Name != "Surveyor" || len(v.Current.Modules) != 1 {
		t.Fatalf("Exosuit equipment lost by a history rebuild: %+v", v)
	}
}
