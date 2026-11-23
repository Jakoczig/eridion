package catalog

import "testing"

func TestAuditPublicInventoryClassification(t *testing.T) {
	for _, tc := range []struct {
		kind, id, symbol, canonical string
		permitted, known            bool
	}{
		{"outfitting", "128049513", "hpt_chafflauncher_tiny", "Hpt_ChaffLauncher_Tiny", true, true},
		{"outfitting", "128064272", "int_shieldgenerator_size3_class5", "Int_ShieldGenerator_Size3_Class5", true, true},
		{"outfitting", "128666676", "int_fuelscoop_size1_class5", "Int_FuelScoop_Size1_Class5", true, true},
		{"outfitting", "128049381", "hpt_pulselaser_fixed_small", "Hpt_PulseLaser_Fixed_Small", true, true},
		{"outfitting", "128049292", "dolphin_armour_grade1", "Dolphin_Armour_Grade1", true, true},
		{"outfitting", "", "Krait_MkII_Armour_Grade1", "Krait_MkII_Armour_Grade1", true, true},
		{"outfitting", "128671325", "int_shieldgenerator_size3_class5_strong", "Int_ShieldGenerator_Size3_Class5_Strong", false, true}, // PowerPlay
		{"outfitting", "128833975", "int_guardianfsdbooster_size1", "Int_GuardianFSDBooster_Size1", false, true},                       // Guardian unlock
		{"outfitting", "128672317", "int_planetapproachsuite", "Int_PlanetApproachSuite", false, true},
		{"outfitting", "", "paintjob_cobramkiii_default_52", "", false, true},       // not a module
		{"outfitting", "999999999", "hpt_future_weapon", "", false, false},          // unknown
		{"outfitting", "128049513", "hpt_pulselaser_fixed_small", "", false, false}, // ID/symbol mismatch
		// Tech-broker unlocks are commander-dependent (G3H final review).
		{"outfitting", "128793117", "int_metaalloyhullreinforcement_size1_class1", "Int_MetaAlloyHullReinforcement_Size1_Class1", false, true},
		{"outfitting", "128834002", "hpt_plasmashockcannon_fixed_medium", "Hpt_PlasmaShockCannon_Fixed_Medium", false, true},
		{"shipyard", "128049291", "Dolphin", "Dolphin", true, true},
		{"shipyard", "128816567", "krait_mkii", "Krait_MkII", true, true},
		{"shipyard", "", "CobraMkIV", "CobraMkIV", false, true},
		{"shipyard", "", "Future_Hull", "", false, false},
	} {
		canonical, permitted, known := PublicInventorySymbol(tc.kind, tc.id, tc.symbol)
		if canonical != tc.canonical || permitted != tc.permitted || known != tc.known {
			t.Errorf("%s %s %s = %q %v %v, want %q %v %v", tc.kind, tc.id, tc.symbol, canonical, permitted, known, tc.canonical, tc.permitted, tc.known)
		}
	}
}
