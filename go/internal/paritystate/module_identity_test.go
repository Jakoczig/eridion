package paritystate

import (
	"encoding/json"
	"testing"
)

func moduleBySlot(mods []Module, slot string) *Module {
	for i := range mods {
		if mods[i].Slot == slot {
			return &mods[i]
		}
	}
	return nil
}

// Task 16 / F20: journal Loadout modules resolve through the ID-first
// catalogue. A shared symbol is the standard module only when no engineering
// is reported; engineered, it stays ambiguous with no stock statistics.
func TestAuditLoadoutModuleIdentity(t *testing.T) {
	mods, _ := parseModules(map[string]any{"Modules": []any{
		map[string]any{"Slot": "HugeHardpoint1", "Item": "hpt_beamlaser_fixed_huge"},
		map[string]any{"Slot": "HugeHardpoint2", "Item": "hpt_beamlaser_fixed_huge", "Engineering": map[string]any{"BlueprintName": "Weapon_Overcharged", "Level": 5, "Modifiers": []any{}}},
		map[string]any{"Slot": "FrameShiftDrive", "Item": "int_hyperdrive_overcharge_size5_class5"},
		map[string]any{"Slot": "Slot01", "Item": "int_future_module_size9"},
	}})
	std, eng, sco, future := moduleBySlot(mods, "HugeHardpoint1"), moduleBySlot(mods, "HugeHardpoint2"), moduleBySlot(mods, "FrameShiftDrive"), moduleBySlot(mods, "Slot01")
	if std.Identity != "standard_unengineered" || std.Name != "Beam Laser" || std.Reference == nil || std.ModuleID != "128049431" {
		t.Fatalf("standard = %+v", std)
	}
	if eng.Identity != "ambiguous_variant" || eng.Reference != nil || len(eng.Candidates) != 2 || eng.Name != "Beam Laser or Overloaded Beam Laser" {
		t.Fatalf("engineered shared symbol = %+v", eng)
	}
	if sco.Identity != "unique_symbol" || sco.ModuleID == "" {
		t.Fatalf("SCO drive = %+v", sco)
	}
	if future.Identity != "unknown" || future.Reference != nil {
		t.Fatalf("unknown module = %+v", future)
	}
}

// Task 16: a CAPI numeric ID resolves a module only for the same slot and
// symbol; the caller verifies commander and ship identity first.
func TestAuditReconcileModulesWithFrontierIDs(t *testing.T) {
	mods, _ := parseModules(map[string]any{"Modules": []any{
		map[string]any{"Slot": "HugeHardpoint2", "Item": "hpt_beamlaser_fixed_huge", "Engineering": map[string]any{"BlueprintName": "Weapon_Overcharged", "Level": 5, "Modifiers": []any{}}},
		map[string]any{"Slot": "HugeHardpoint1", "Item": "hpt_beamlaser_fixed_huge", "Engineering": map[string]any{"BlueprintName": "Weapon_Overcharged", "Level": 5, "Modifiers": []any{}}},
	}})
	out := ReconcileModules(mods, []ModuleRef{
		{Slot: "hugehardpoint2", Item: "Hpt_BeamLaser_Fixed_Huge", ID: "129046189"},
		{Slot: "HugeHardpoint1", Item: "Hpt_PulseLaser_Fixed_Huge", ID: "128049383"}, // other module in that slot
	})
	if m := moduleBySlot(out, "HugeHardpoint2"); m.Identity != "exact_id" || m.Name != "Overloaded Beam Laser" || m.ModuleID != "129046189" || m.Reference == nil || m.Reference.MassT != nil || m.Reference.PowerMW != nil {
		t.Fatalf("matched slot = %+v", m)
	}
	if m := moduleBySlot(out, "HugeHardpoint1"); m.Identity != "ambiguous_variant" {
		t.Fatalf("mismatched symbol resolved: %+v", m)
	}
	if moduleBySlot(mods, "HugeHardpoint2").Identity != "ambiguous_variant" {
		t.Fatal("ReconcileModules mutated its input")
	}
}

// Task 16: identification never re-applies engineering. A pre-built variant
// resolved by ID carries no stock power, and the reported final PowerDraw is
// used exactly once.
func TestAuditResolvedVariantKeepsFinalEngineeredValues(t *testing.T) {
	mods, _ := parseModules(map[string]any{"Modules": []any{
		map[string]any{"Slot": "HugeHardpoint1", "Item": "hpt_beamlaser_fixed_huge", "On": true, "Priority": json.Number("0"), "Engineering": map[string]any{"BlueprintName": "Weapon_Overcharged", "Level": 5, "Modifiers": []any{map[string]any{"Label": "PowerDraw", "Value": json.Number("4.5"), "OriginalValue": json.Number("3")}}}},
	}})
	mods = ReconcileModules(mods, []ModuleRef{{Slot: "HugeHardpoint1", Item: "hpt_beamlaser_fixed_huge", ID: "129046189"}})
	v := Statistics(mods)
	if len(v.Modules) != 1 || v.KnownPriorityMW[0] != 4.5 {
		t.Fatalf("final engineered draw not used exactly once: %+v", v)
	}
}

// G3I review F2: a verified Frontier ID the bundled catalogue does not know
// (a future variant), or one that names another module for this symbol,
// leaves the module unresolved: no stock statistics from the journal-only
// guess, never a module ID contradicting Frontier.
func TestAuditUnknownFrontierIDStaysUnresolved(t *testing.T) {
	mods, _ := parseModules(map[string]any{"Modules": []any{
		map[string]any{"Slot": "PowerDistributor", "Item": "int_powerdistributor_size4_class5"},
		map[string]any{"Slot": "SmallHardpoint1", "Item": "hpt_pulselaser_fixed_small"},
		map[string]any{"Slot": "SmallHardpoint2", "Item": "hpt_pulselaser_fixed_small"},
	}})
	if m := moduleBySlot(mods, "PowerDistributor"); m.Identity != "standard_unengineered" || m.Reference == nil {
		t.Fatalf("journal-only baseline: %+v", m)
	}
	out := ReconcileModules(mods, []ModuleRef{
		{Slot: "PowerDistributor", Item: "Int_PowerDistributor_Size4_Class5", ID: "129099998"},
		{Slot: "SmallHardpoint1", Item: "Hpt_PulseLaser_Fixed_Small", ID: "129099999"},
		{Slot: "SmallHardpoint2", Item: "Hpt_PulseLaser_Fixed_Small", ID: "128049383"}, // ID of another module
	})
	for _, slot := range []string{"PowerDistributor", "SmallHardpoint1", "SmallHardpoint2"} {
		m := moduleBySlot(out, slot)
		if m.Identity != "unknown" || m.Reference != nil || m.ModuleID == "128064197" {
			t.Errorf("%s: %+v", slot, m)
		}
	}
}
