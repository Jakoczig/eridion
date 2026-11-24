package explorestate

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Task 6 / F07: the stable genus symbol is the identity; the localized label
// is presentation only. The same discovery yields the same genusIds in every
// game language, with missing labels and duplicates.
func TestAuditBioLocaleIdentity(t *testing.T) {
	genuses := map[string]string{
		"english":  `[{"Genus":"$Codex_Ent_Bacterial_Genus_Name;","Genus_Localised":"Bacterium"},{"Genus":"$Codex_Ent_Stratum_Genus_Name;","Genus_Localised":"Stratum"}]`,
		"german":   `[{"Genus":"$Codex_Ent_Bacterial_Genus_Name;","Genus_Localised":"Bakterium"},{"Genus":"$Codex_Ent_Stratum_Genus_Name;","Genus_Localised":"Stratum"}]`,
		"missing":  `[{"Genus":"$Codex_Ent_Bacterial_Genus_Name;"},{"Genus":"$Codex_Ent_Stratum_Genus_Name;","Genus_Localised":""}]`,
		"repeated": `[{"Genus":"$Codex_Ent_Bacterial_Genus_Name;","Genus_Localised":"Bactérie"},{"Genus":"$Codex_Ent_Bacterial_Genus_Name;","Genus_Localised":"Bakterium"},{"Genus":"$Codex_Ent_Stratum_Genus_Name;","Genus_Localised":"Stratum"}]`,
	}
	want := []string{"$Codex_Ent_Bacterial_Genus_Name;", "$Codex_Ent_Stratum_Genus_Name;"}
	labels := map[string][]string{}
	for name, g := range genuses {
		m := start(t)
		apply(t, m, "A", scan)
		apply(t, m, "A", `{"event":"SAASignalsFound","SystemAddress":9007199254740993,"BodyID":9007199254740997,"Signals":[{"Type":"$SAA_SignalType_Biological;","Count":2}],"Genuses":`+g+`}`)
		b := m.View("A", "9007199254740993").Bodies[0]
		if !reflect.DeepEqual(b.GenusIDs, want) {
			t.Errorf("%s: genusIds = %v, want %v", name, b.GenusIDs, want)
		}
		if len(b.Genuses) != len(b.GenusIDs) {
			t.Errorf("%s: labels %v do not pair with ids %v", name, b.Genuses, b.GenusIDs)
		}
		labels[name] = b.Genuses
		raw, _ := json.Marshal(b)
		if !strings.Contains(string(raw), `"genusIds":[`) {
			t.Errorf("%s: genusIds not serialized: %s", name, raw)
		}
	}
	if labels["german"][0] != "Bakterium" || labels["english"][0] != "Bacterium" {
		t.Errorf("localized labels not preserved: %v", labels)
	}
	if labels["missing"][0] == "" || labels["missing"][1] == "" {
		t.Errorf("missing label left blank: %v", labels["missing"])
	}
	// The view is a copy: mutating it never changes the reducer.
	m := start(t)
	apply(t, m, "A", scan)
	apply(t, m, "A", `{"event":"SAASignalsFound","SystemAddress":9007199254740993,"BodyID":9007199254740997,"Genuses":`+genuses["english"]+`}`)
	v := m.View("A", "9007199254740993")
	v.Bodies[0].GenusIDs[0] = "MUTATED"
	if m.View("A", "9007199254740993").Bodies[0].GenusIDs[0] == "MUTATED" {
		t.Fatal("genusIds shared with the reducer")
	}
}

// Task 6: a sampling cycle keeps one identity when the game language changes
// mid-cycle; another species interrupts it without merging identities.
func TestAuditBioInterruptedCycleIdentity(t *testing.T) {
	m := start(t)
	apply(t, m, "A", scan)
	organic := func(stage, ts, genusLabel, species, speciesLabel string) {
		apply(t, m, "A", `{"event":"ScanOrganic","timestamp":"`+ts+`","ScanType":"`+stage+`","Genus":"$Codex_Ent_Bacterial_Genus_Name;","Genus_Localised":"`+genusLabel+`","Species":"`+species+`","Species_Localised":"`+speciesLabel+`","Variant":"`+species[:len(species)-6]+`_Tur_Name;","SystemAddress":9007199254740993,"Body":9007199254740997}`)
	}
	organic("Log", "2026-11-24T10:00:00Z", "Bacterium", "$Codex_Ent_Bacterial_06_Name;", "Bacterium Acies")
	organic("Sample", "2026-11-24T10:01:00Z", "Bakterium", "$Codex_Ent_Bacterial_06_Name;", "Bakterium Acies")
	b := m.View("A", "9007199254740993").Bodies[0]
	if len(b.Organisms) != 1 || b.Organisms[0].Stage != "sampled" || b.Organisms[0].SpeciesID != "$Codex_Ent_Bacterial_06_Name;" || b.Organisms[0].GenusID != "$Codex_Ent_Bacterial_Genus_Name;" || b.Organisms[0].Species != "Bakterium Acies" {
		t.Fatalf("language change split or relabelled the cycle identity: %+v", b.Organisms)
	}
	organic("Log", "2026-11-24T10:02:00Z", "Bacterium", "$Codex_Ent_Bacterial_01_Name;", "Bacterium Aurasus")
	b = m.View("A", "9007199254740993").Bodies[0]
	if len(b.Organisms) != 2 || b.Organisms[0].Stage != "interrupted" || b.Organisms[1].SpeciesID != "$Codex_Ent_Bacterial_01_Name;" || b.Organisms[1].Stage != "logged" {
		t.Fatalf("interruption: %+v", b.Organisms)
	}
	if b.Organisms[0].VariantID == "" {
		t.Fatalf("variant id lost: %+v", b.Organisms[0])
	}
}
