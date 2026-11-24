package gamestate

import (
	"bytes"
	"context"
	"encoding/gob"
	"os"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/fieldops"
	"github.com/Jakoczig/eridion/go/internal/history"
)

const bioSystem = `"StarSystem":"Fixture Bio","SystemAddress":9007199254740993`

// bioJournal writes a synthetic arrival, a landable body scan and an SAA
// signal event whose genus labels are in the given language.
func bioJournal(t *testing.T, dir, labels string) {
	t.Helper()
	appendEvent(t, dir, `{"event":"LoadGame","timestamp":"2026-11-24T10:00:00Z","FID":"SYNTHETIC-A","Commander":"Fixture"}`)
	appendEvent(t, dir, `{"event":"Location","timestamp":"2026-11-24T10:00:01Z",`+bioSystem+`,"StarPos":[0,0,0]}`)
	appendEvent(t, dir, `{"event":"Scan","timestamp":"2026-11-24T10:00:02Z","ScanType":"Detailed",`+bioSystem+`,"BodyID":5,"BodyName":"Fixture Bio 1","DistanceFromArrivalLS":120.5,"PlanetClass":"Rocky body","Landable":true,"Atmosphere":"thin ammonia atmosphere","AtmosphereType":"Ammonia","AtmosphereComposition":[{"Name":"Ammonia","Percent":100}],"Volcanism":"","SurfaceGravity":2.5,"SurfaceTemperature":170.0,"SurfacePressure":1000.0,"OrbitalPeriod":100000.0,"Materials":[{"Name":"iron","Percent":20}],"Parents":[{"Star":0}]}`)
	appendEvent(t, dir, `{"event":"SAASignalsFound","timestamp":"2026-11-24T10:00:03Z","SystemAddress":9007199254740993,"BodyName":"Fixture Bio 1","BodyID":5,"Signals":[{"Type":"$SAA_SignalType_Biological;","Count":1}],"Genuses":[{"Genus":"$Codex_Ent_Bacterial_Genus_Name;","Genus_Localised":"`+labels+`"}]}`)
}

func bioCandidates(t *testing.T, s *Service) []fieldops.Candidate {
	t.Helper()
	v, err := s.FieldBio("5", true)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := v["candidates"].([]fieldops.Candidate)
	return c
}

// Task 6 / F07 end to end: the candidate list for the same discovery is the
// same whether the game reported the genus label in English or German.
func TestAuditBioLocaleCandidatesEndToEnd(t *testing.T) {
	got := map[string][]string{}
	for _, label := range []string{"Bacterium", "Bakterium"} {
		dir, profile := t.TempDir(), t.TempDir()
		bioJournal(t, dir, label)
		writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
		s, _, stop := recordingService(t, profile, dir)
		waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 4 })
		for _, c := range bioCandidates(t, s) {
			got[label] = append(got[label], c.Symbol+"="+c.State)
		}
		stop()
	}
	if len(got["Bacterium"]) == 0 {
		t.Fatal("no candidates for the English label")
	}
	if len(got["Bakterium"]) != len(got["Bacterium"]) {
		t.Fatalf("localized label changed the candidates:\n english %v\n german  %v", got["Bacterium"], got["Bakterium"])
	}
	for i := range got["Bacterium"] {
		if got["Bacterium"][i] != got["Bakterium"][i] {
			t.Fatalf("candidate %d differs: %s vs %s", i, got["Bacterium"][i], got["Bakterium"][i])
		}
	}
}

// Task 6: a cache written by an older build (different reducer format, no
// genusIds) is rebuilt from retained journal history on restart. This works
// even when the original journal files are gone, and no sample history is lost.
func TestAuditBioCacheUpgradeRebuildsGenusIDs(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	bioJournal(t, dir, "Bakterium")
	appendEvent(t, dir, `{"event":"ScanOrganic","timestamp":"2026-11-24T10:00:04Z","ScanType":"Log","Genus":"$Codex_Ent_Bacterial_Genus_Name;","Genus_Localised":"Bakterium","Species":"$Codex_Ent_Bacterial_06_Name;","Species_Localised":"Bacterium Acies","SystemAddress":9007199254740993,"Body":5}`)
	writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
	s, _, stop := recordingService(t, profile, dir)
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 5 })
	before := bioCandidates(t, s)
	samplesBefore := len(mustBio(t, s)["samples"].([]history.BioSample))
	store, _ := s.History()
	raw, err := store.Cache()
	if err != nil || len(raw) == 0 {
		t.Fatal("no cache", err)
	}
	var c reducerCache
	if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&c); err != nil {
		t.Fatal(err)
	}
	// Simulate the older build: a different format and bodies without ids.
	c.Format = "older-build-format"
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(c); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveCache(context.Background(), b.Bytes()); err != nil {
		t.Fatal(err)
	}
	stop()
	// The original journal files are gone; only retained history remains.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(dir, 0o700)

	s, _, stop = recordingService(t, profile, dir)
	defer stop()
	deadline := time.Now().Add(5 * time.Second)
	for {
		v := s.Snapshot()
		if v.Stage == "DB_ERROR" {
			t.Fatalf("an older cache format put the service into recovery: %+v", v)
		}
		if !v.ImportInProgress && len(bioCandidates(t, s)) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("candidates not rebuilt from history: %+v", v)
		}
		time.Sleep(20 * time.Millisecond)
	}
	after := bioCandidates(t, s)
	if len(after) != len(before) {
		t.Fatalf("rebuilt candidates differ: %d vs %d", len(after), len(before))
	}
	if n := len(mustBio(t, s)["samples"].([]history.BioSample)); n != samplesBefore || n == 0 {
		t.Fatalf("sample history changed across the upgrade: %d vs %d", n, samplesBefore)
	}
}

func mustBio(t *testing.T, s *Service) map[string]any {
	t.Helper()
	v, err := s.FieldBio("5", true)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
