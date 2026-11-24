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

// G3I review F4: a blank (or Horizons) entitlement on an ID added by the G3I
// refresh is not evidence of public stock; such IDs need a per-ID review.
func TestAuditNewIDsNeedReview(t *testing.T) {
	for _, c := range []struct{ kind, id, symbol, want string }{
		{"outfitting", "128049431", "Hpt_BeamLaser_Fixed_Huge", InventoryPublic},                   // pre-refresh, blank entitlement
		{"outfitting", "129030592", "Int_Hyperdrive_Overcharge_Size5_Class5", InventoryPublic},     // SCO 2-7, reviewed
		{"outfitting", "129036580", "Int_Hyperdrive_Overcharge_Size8_Class5", InventoryUnverified}, // SCO size 8, no evidence
		{"outfitting", "129041074", "Int_MkIIAgileBoost_Engine_Size5_Class5", InventoryUnverified}, // Mk II reward
		{"outfitting", "129038968", "Int_Hyperdrive_Overcharge_Size8_Class5_OverchargeBooster_MkII", InventoryUnverified},
		{"outfitting", "129046189", "Hpt_BeamLaser_Fixed_Huge", InventoryUnverified},             // pre-engineered
		{"outfitting", "129022079", "Hpt_ATDumbfireMissile_Fixed_Large_V2", InventoryUnverified}, // new, Horizons entitlement
		{"outfitting", "129030049", "Hpt_ATVentDisruptorPylon_Fixed_Medium", InventoryExcluded},  // Guardian broker
		{"outfitting", "129019260", "Int_ExpModuleStabiliser_Size3_Class3", InventoryExcluded},   // tech broker
		{"outfitting", "129030466", "Python_NX_Armour_Grade1", InventoryPublic},                  // specific reviewed entitlement
		{"shipyard", "129030464", "Python_NX", InventoryPublic},
	} {
		if _, got, _ := ClassifyInventory(c.kind, c.id, c.symbol); got != c.want {
			t.Errorf("%s %s %s = %s, want %s", c.kind, c.id, c.symbol, got, c.want)
		}
	}
	// Without an ID, a shared symbol whose variants now differ is ambiguous.
	if _, got, _ := ClassifyInventory("outfitting", "", "Hpt_BeamLaser_Fixed_Huge"); got != InventoryAmbiguous {
		t.Errorf("shared symbol without ID = %s", got)
	}
}

func TestInventoryIDReviewIsWellFormed(t *testing.T) {
	inventoryOnce.Do(loadInventory)
	if idWatermark == 0 {
		t.Fatal("inventory-id-review.csv watermark missing")
	}
	for id, class := range idReview {
		if _, ok := inventoryByID["outfitting"][id]; !ok {
			if _, ok := inventoryByID["shipyard"][id]; !ok {
				t.Errorf("reviewed ID %s is not in the catalogue", id)
			}
		}
		if class != "ordinary" && class != "excluded" {
			t.Errorf("reviewed ID %s has classification %q", id, class)
		}
	}
}
