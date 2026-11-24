package gamestate

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"os"
	"strings"
	"sync"
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
	// Simulate the older build: a different reducer format. Startup must not
	// restore any of it; every derived view is rebuilt from history.
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
	// The genus identity itself is rebuilt from the retained SAASignalsFound
	// evidence (not only reached through the sample-genus fallback).
	s.mu.Lock()
	survey := s.views.exploration.View("SYNTHETIC-A", "9007199254740993")
	s.mu.Unlock()
	if len(survey.Bodies) == 0 || len(survey.Bodies[0].GenusIDs) != 1 || survey.Bodies[0].GenusIDs[0] != "$Codex_Ent_Bacterial_Genus_Name;" || survey.Bodies[0].Genuses[0] != "Bakterium" {
		t.Fatalf("genus ids not rebuilt from history: %+v", survey.Bodies)
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

// Task 18 (backend): identical concurrent BioScan reads share one inference;
// different evidence never reuses a cached result; cache keys never cross
// commanders.
func TestAuditBioInferenceSingleFlight(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	bioJournal(t, dir, "Bacterium")
	writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
	s, _, stop := recordingService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 4 })
	var wg sync.WaitGroup
	results := make([][]fieldops.Candidate, 8)
	for i := range results {
		wg.Add(1)
		go func(i int) { defer wg.Done(); results[i] = bioCandidates(t, s) }(i)
	}
	wg.Wait()
	s.bioCache.mu.Lock()
	computed := s.bioCache.computed
	s.bioCache.mu.Unlock()
	if computed != 1 {
		t.Fatalf("8 identical reads ran %d inferences", computed)
	}
	for i := range results {
		if len(results[i]) == 0 || len(results[i]) != len(results[0]) {
			t.Fatalf("read %d differs", i)
		}
	}
	results[0][0].Unknown = append(results[0][0].Unknown, "MUTATED")
	if again := bioCandidates(t, s); strings.Contains(strings.Join(again[0].Unknown, ","), "MUTATED") {
		t.Fatal("cached candidates shared with a caller")
	}
	// New evidence (FSS complete) changes the key and recomputes.
	appendEvent(t, dir, `{"event":"FSSAllBodiesFound","timestamp":"2026-11-24T10:00:05Z","SystemName":"Fixture Bio","SystemAddress":9007199254740993,"Count":1}`)
	waitState(t, s, func(v Summary) bool { return v.EventsRead == 5 })
	bioCandidates(t, s)
	s.bioCache.mu.Lock()
	computed = s.bioCache.computed
	s.bioCache.mu.Unlock()
	if computed != 2 {
		t.Fatalf("changed evidence reused a cached result (computed=%d)", computed)
	}
	env := fieldops.Environment{Body: map[string]any{"PlanetClass": "Rocky body"}}
	if bioKey(bioContext{fid: "SYNTHETIC-A", system: "1", body: "5"}, env, nil) == bioKey(bioContext{fid: "SYNTHETIC-B", system: "1", body: "5"}, env, nil) {
		t.Fatal("cache key shared across commanders")
	}
}

// Task 18: disabling the BioScan module stops presentation only; journal
// ingestion keeps recording samples, which reappear when it is re-enabled.
func TestAuditBioModuleToggleKeepsIngestion(t *testing.T) {
	dir, profile := t.TempDir(), t.TempDir()
	bioJournal(t, dir, "Bacterium")
	writeAccounts(t, profile, `{"version":2,"multiple":false,"selected":"","defaultFID":"SYNTHETIC-A"}`)
	s, _, stop := recordingService(t, profile, dir)
	defer stop()
	waitState(t, s, func(v Summary) bool { return !v.ImportInProgress && v.EventsRead == 4 })
	store, _ := s.History()
	if err := store.SetBuiltinModule("bioscan", false); err != nil {
		t.Fatal(err)
	}
	if v := mustBio(t, s); v["notice"] != "BioScan module disabled; stored observations retained" {
		t.Fatalf("disabled module: %v", v["notice"])
	}
	appendEvent(t, dir, `{"event":"ScanOrganic","timestamp":"2026-11-24T10:00:06Z","ScanType":"Log","Genus":"$Codex_Ent_Bacterial_Genus_Name;","Species":"$Codex_Ent_Bacterial_06_Name;","SystemAddress":9007199254740993,"Body":5}`)
	waitState(t, s, func(v Summary) bool { return v.EventsRead == 5 })
	if err := store.SetBuiltinModule("bioscan", true); err != nil {
		t.Fatal(err)
	}
	v := mustBio(t, s)
	if samples := v["samples"].([]history.BioSample); len(samples) != 1 || samples[0].Stage != "Log" {
		t.Fatalf("sample recorded while disabled was lost: %+v", v["samples"])
	}
	if len(v["candidates"].([]fieldops.Candidate)) == 0 {
		t.Fatal("candidates not restored after re-enable")
	}
}

// G3J review F3: a cache from a build whose reducer types changed (not just
// gained fields) still counts as an older format, not as corruption.
func TestAuditCacheFormatCheckedBeforeFullDecode(t *testing.T) {
	var b bytes.Buffer
	type olderCache struct {
		Format string
		Ship   int // a different type than today's field
	}
	if err := gob.NewEncoder(&b).Encode(olderCache{Format: "older-build-format", Ship: 7}); err != nil {
		t.Fatal(err)
	}
	s := &Service{}
	if err := s.restoreCacheLocked(b.Bytes()); !errors.Is(err, errCacheFormat) {
		t.Fatalf("older incompatible cache = %v, want errCacheFormat", err)
	}
	if err := s.restoreCacheLocked([]byte("not a gob stream")); err == nil || errors.Is(err, errCacheFormat) {
		t.Fatalf("corrupt cache must not be treated as an older format: %v", err)
	}
}

// G3J review F5/F9: a failed inference never strands later readers of the
// same key; FSS completeness needs scans covering every reported body.
func TestAuditBioInferenceFailureAndCompleteness(t *testing.T) {
	s := &Service{}
	calls := 0
	inferBio = func(fieldops.Environment, []string) []fieldops.Candidate {
		calls++
		if calls == 1 {
			panic("synthetic inference failure")
		}
		return []fieldops.Candidate{{Symbol: "OK"}}
	}
	defer func() { inferBio = fieldops.InferEnvironment }()
	c := bioContext{fid: "SYNTHETIC-A", system: "1", body: "5"}
	func() {
		defer func() { _ = recover() }()
		s.bioCandidates(c, fieldops.Environment{}, nil)
	}()
	done := make(chan []fieldops.Candidate, 1)
	go func() { done <- s.bioCandidates(c, fieldops.Environment{}, nil) }()
	select {
	case got := <-done:
		if len(got) != 1 || got[0].Symbol != "OK" {
			t.Fatalf("retry after failure: %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a reader waited forever on a failed inference")
	}
	for _, tc := range []struct {
		all              bool
		scanner, scanned int
		want             bool
	}{{true, 3, 3, true}, {true, 3, 2, false}, {false, 3, 3, false}, {true, 0, 0, false}} {
		if got := fssComplete(tc.all, tc.scanner, tc.scanned); got != tc.want {
			t.Errorf("fssComplete(%v,%d,%d) = %v", tc.all, tc.scanner, tc.scanned, got)
		}
	}
}
