package catalog

import "testing"

// Task 16 / F20: symbols shared by a standard module and its pre-built
// variants are never collapsed. A numeric ID resolves exactly; a symbol alone
// resolves to the standard module only when no engineering is reported
// (pre-built variants are always engineered); otherwise it stays unresolved.
func TestAuditModuleIdentityCollisions(t *testing.T) {
	for _, tc := range []struct {
		symbol, id string
		engineered bool
		status     string
		name       string
	}{
		{"Hpt_BeamLaser_Fixed_Huge", "128049431", false, StatusExactID, "Beam Laser"},
		{"Hpt_BeamLaser_Fixed_Huge", "129046189", true, StatusExactID, "Overloaded Beam Laser"},
		{"hpt_beamlaser_fixed_huge", "", false, StatusStandardUnengineered, "Beam Laser"},
		{"hpt_beamlaser_fixed_huge", "", true, StatusAmbiguous, ""},
		{"Int_PowerDistributor_Size5_Class5", "", true, StatusAmbiguous, ""},
		{"Int_PowerDistributor_Size5_Class5", "129044376", true, StatusExactID, "Balanced Power Distributor"},
		{"hpt_basicmissilerack_fixed_medium", "", false, StatusStandardUnengineered, "Seeker Missile Rack"},
		{"Int_Hyperdrive_Overcharge_Size5_Class5", "", false, StatusUniqueSymbol, ""}, // SCO FSD: own symbol
		{"Hpt_ChaffLauncher_Tiny", "", true, StatusUniqueSymbol, "Chaff Launcher"},
		{"Hpt_BeamLaser_Fixed_Huge", "128049513", false, StatusUnknown, ""}, // ID of another module
		{"hpt_future_weapon_fixed_small", "", false, StatusUnknown, ""},
		{"hpt_future_weapon_fixed_small", "999999999", false, StatusUnknown, ""},
	} {
		r := ResolveModule(tc.symbol, tc.id, tc.engineered)
		if r.Status != tc.status || (tc.name != "" && r.Module.Name != tc.name) {
			t.Errorf("%s id=%q eng=%v: status=%s name=%q, want %s %q", tc.symbol, tc.id, tc.engineered, r.Status, r.Module.Name, tc.status, tc.name)
		}
		if r.Status == StatusAmbiguous && (len(r.Candidates) < 2 || r.Module.MassT != nil || r.Module.Class != nil) {
			t.Errorf("%s ambiguous resolution leaked one variant's statistics: %+v", tc.symbol, r)
		}
	}
	// The symbol-only lookup used for labels never returns one arbitrary
	// variant of a shared symbol.
	if m, ok := ModuleBySymbol("hpt_beamlaser_fixed_huge"); ok && m.Name == "Overloaded Beam Laser" {
		t.Fatal("shared symbol collapsed to a pre-built variant")
	}
	if name := ModuleDisplayName("hpt_beamlaser_fixed_huge"); name != "Beam Laser or Overloaded Beam Laser" {
		t.Fatalf("ambiguous display name = %q", name)
	}
}

// Task 16: newer hulls resolve to their verified labels; unknown hulls stay
// visible with their raw symbol, never mapped to another hull.
func TestAuditNewHullLabels(t *testing.T) {
	for symbol, want := range map[string]string{
		"Python_NX": "Python MkII", "Type8": "Type-8 Transporter", "Mandalay": "Mandalay", "CobraMkV": "Cobra MkV",
		"Corsair": "Corsair", "PantherMkII": "Panther Clipper MkII", "LakonMiner": "Type-11 Prospector",
		"Explorer_NX": "Caspian Explorer", "SmallCombat01_NX": "Kestrel Mk II", "MediumTransport01": "Lynx Highliner",
		"AnacondaMkII": "Anaconda Mk II", "python_nx": "Python MkII",
	} {
		if got := ShipDisplayName(symbol); got != want {
			t.Errorf("%s = %q, want %q", symbol, got, want)
		}
	}
	if got := ShipDisplayName("Future_Hull_99"); got == "" || got == "Python MkII" {
		t.Fatalf("unknown hull label = %q", got)
	}
	if VehicleKind("Explorer_NX") != "ship" || VehicleKind("AnacondaMkII") != "ship" {
		t.Fatal("new hulls misclassified as vehicles")
	}
}
