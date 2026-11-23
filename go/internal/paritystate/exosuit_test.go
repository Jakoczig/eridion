package paritystate

import (
	"fmt"
	"github.com/Jakoczig/eridion/go/internal/journal"
	"github.com/Jakoczig/eridion/go/internal/sidecar"
	"testing"
)

const suitSnapshot = `,"SuitID":1700217809818876,"SuitName":"explorationsuit_class3","SuitName_Localised":"Artemis Suit","SuitMods":["suit_increasedsprintduration"],"LoadoutID":4293000005,"LoadoutName":"Surveyor","Modules":[{"SlotName":"PrimaryWeapon1","SuitModuleID":1700217863661544,"ModuleName":"wpn_m_assaultrifle_kinetic_fauto","ModuleName_Localised":"Karma AR-50","Class":2,"WeaponMods":["weapon_stability"]},{"SlotName":"SecondaryWeapon","SuitModuleID":1700216182854765,"ModuleName":"wpn_s_pistol_plasma_charged","ModuleName_Localised":"Manticore Tormentor","Class":1,"WeaponMods":[]}]`

func TestAuditExosuitKnownEquipment(t *testing.T) {
	m := session(t)
	if v := m.Exosuit("a"); v.State != "unknown" || v.Current != nil {
		t.Fatalf("equipment invented: %+v", v)
	}
	m.ApplyEvent("a", event(t, 2, "SuitLoadout", suitSnapshot))
	v := m.Exosuit("a")
	if v.State != "known" || v.Current == nil || v.Current.SuitID != "1700217809818876" || v.Current.SuitLabel != "Artemis Suit" || v.Current.Name != "Surveyor" || len(v.Current.Modules) != 2 {
		t.Fatalf("snapshot = %+v", v)
	}
	if v.Current.Grade != nil {
		t.Fatal("grade inferred from the class suffix", *v.Current.Grade)
	}
	if w := v.Current.Modules[0]; w.Slot != "PrimaryWeapon1" || w.ModuleID != "1700217863661544" || w.Label != "Karma AR-50" || w.Class == nil || *w.Class != 2 || len(w.Mods) != 1 {
		t.Fatalf("weapon = %+v", w)
	}
	// A partial delta changes only its own slot.
	m.ApplyEvent("a", event(t, 3, "LoadoutRemoveModule", `,"SuitID":1700217809818876,"LoadoutID":4293000005,"LoadoutName":"Surveyor","SlotName":"SecondaryWeapon","SuitModuleID":1700216182854765`))
	m.ApplyEvent("a", event(t, 4, "LoadoutEquipModule", `,"SuitID":1700217809818876,"LoadoutID":4293000005,"LoadoutName":"Surveyor","SlotName":"PrimaryWeapon2","SuitModuleID":1700000000000001,"ModuleName":"wpn_m_sniper_plasma_charged","ModuleName_Localised":"Manticore Executioner","Class":3,"WeaponMods":[]`))
	m.ApplyEvent("a", event(t, 5, "RenameSuitLoadout", `,"SuitID":1700217809818876,"LoadoutID":4293000005,"LoadoutName":"Long range"`))
	m.ApplyEvent("a", event(t, 6, "UpgradeSuit", `,"Name":"explorationsuit_class3","SuitID":1700217809818876,"Class":4,"Cost":1`))
	v = m.Exosuit("a")
	slots := map[string]bool{}
	for _, mod := range v.Current.Modules {
		slots[mod.Slot] = true
	}
	if !slots["PrimaryWeapon1"] || slots["SecondaryWeapon"] || !slots["PrimaryWeapon2"] || v.Current.Name != "Long range" || v.Current.Grade == nil || *v.Current.Grade != 4 {
		t.Fatalf("deltas = %+v", v.Current)
	}
	// Create/delete keep the known loadout list exact.
	m.ApplyEvent("a", event(t, 7, "CreateSuitLoadout", `,"SuitID":1,"SuitName":"flightsuit","LoadoutID":4293000006,"LoadoutName":"Spare","Modules":[]`))
	if len(m.Exosuit("a").Loadouts) != 2 {
		t.Fatalf("created loadout missing: %+v", m.Exosuit("a").Loadouts)
	}
	m.ApplyEvent("a", event(t, 8, "DeleteSuitLoadout", `,"SuitID":1,"SuitName":"flightsuit","LoadoutID":4293000006,"LoadoutName":"Spare"`))
	if len(m.Exosuit("a").Loadouts) != 1 {
		t.Fatalf("deleted loadout retained: %+v", m.Exosuit("a").Loadouts)
	}
	// The view is detached.
	got := m.Exosuit("a")
	got.Current.Modules[0].Label = "mutated"
	if m.Exosuit("a").Current.Modules[0].Label == "mutated" {
		t.Fatal("mutable exosuit view")
	}
	// Known equipment survives shutdown and the next session (offline).
	m.ApplyEvent("a", event(t, 9, "Shutdown", ""))
	if v := m.Exosuit("a"); v.Current == nil || v.Current.Name != "Long range" {
		t.Fatalf("shutdown erased known equipment: %+v", v)
	}
	next, _ := journal.ParseLine([]byte(`{"timestamp":"2026-11-14T11:00:00Z","event":"LoadGame","FID":"private","Ship":"FlightSuit"}`), journal.SourcePosition{FileID: "g", Offset: 100, Sequence: 1})
	m.ApplyEvent("a", next)
	if v := m.Exosuit("a"); v.Current == nil || v.Current.Name != "Long range" || v.Controlled != "suit" {
		t.Fatalf("new session lost equipment or context: %+v", v)
	}
	// Another commander never sees it.
	other, _ := journal.ParseLine([]byte(`{"timestamp":"2026-11-14T11:00:01Z","event":"LoadGame","FID":"other","Ship":"FlightSuit"}`), journal.SourcePosition{FileID: "g", Offset: 200, Sequence: 2})
	m.ApplyEvent("b", other)
	if v := m.Exosuit("b"); v.Current != nil {
		t.Fatalf("equipment crossed accounts: %+v", v)
	}
}

func TestAuditOnFootTelemetry(t *testing.T) {
	m := session(t)
	status := func(seq uint64, flags, flags2 int, extra string) sidecar.Change {
		stamp := event(t, seq, "Anything", "").Timestamp
		return sidecar.Change{Kind: "Status", Valid: true, Timestamp: stamp, Raw: []byte(fmt.Sprintf(`{"timestamp":%q,"event":"Status","Flags":%d,"Flags2":%d%s}`, stamp.Format("2006-01-02T15:04:05Z"), flags, flags2, extra))}
	}
	m.ApplySidecar("a", status(2, 0, 1|1<<16, `,"Oxygen":0.5,"Health":0.75,"Temperature":293.5,"SelectedWeapon":"$humanoid_fists_name;","SelectedWeapon_Localised":"Unarmed","Gravity":0.17,"BodyName":"Fixture 1 a","Fuel":{"FuelMain":3},"Cargo":4`))
	s := m.Status("a")
	if s.OnFoot == nil || s.OnFoot.Health == nil || *s.OnFoot.Health != 0.75 || s.OnFoot.Oxygen == nil || *s.OnFoot.Oxygen != 0.5 || s.OnFoot.SelectedWeaponLabel != "Unarmed" || s.OnFoot.GravityG == nil {
		t.Fatalf("on-foot telemetry = %+v", s.OnFoot)
	}
	if s.FuelMainT != nil || s.CargoT != nil {
		t.Fatal("suit context projected ship fuel/cargo")
	}
	// Absent values stay unknown rather than zero.
	m.ApplySidecar("a", status(3, 0, 1, ``))
	if s := m.Status("a"); s.OnFoot == nil || s.OnFoot.Health != nil || s.OnFoot.Oxygen != nil {
		t.Fatalf("absent values invented: %+v", s.OnFoot)
	}
	// Back in the main ship: no on-foot claim.
	m.ApplySidecar("a", status(4, 1<<24, 0, `,"Fuel":{"FuelMain":3}`))
	if s := m.Status("a"); s.OnFoot != nil || s.FuelMainT == nil {
		t.Fatalf("embarked status = %+v", s)
	}
}
