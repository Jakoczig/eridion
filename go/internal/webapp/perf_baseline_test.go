package webapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/perfstats"
)

// TestPerfNativeServiceBaseline measures the real embedded service (journal
// reader, SQLite writer, reducers, providers wiring) in-process, without a
// window. It is skipped unless ERIDION_PERF_JOURNAL names a synthetic journal
// directory (see scripts/testing/perf-journal-fixture.cjs). One run = one
// repetition: cold import, incremental restart and live append latency.
// Results are printed as one JSON line for docs/benchmarks/native-performance.
func TestPerfNativeServiceBaseline(t *testing.T) {
	source := os.Getenv("ERIDION_PERF_JOURNAL")
	if source == "" {
		t.Skip("set ERIDION_PERF_JOURNAL to a synthetic journal directory")
	}
	journalDir, profile := t.TempDir(), t.TempDir()
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	expected := 0
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(source, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range raw {
			if b == '\n' {
				expected++
			}
		}
		if err = os.WriteFile(filepath.Join(journalDir, e.Name()), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	result := map[string]any{"events": expected}
	start := func() *Server {
		s, err := StartLocalWithOptions(context.Background(), fixtureFS(), Options{StorageDir: profile, JournalDir: journalDir, EnableJournal: true, EnableNavigation: true, EnableExploration: true, EnableDashboard: true, EnableParity: true})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	wait := func(s *Server, want int, limit time.Duration) time.Duration {
		begin := time.Now()
		for time.Since(begin) < limit {
			v := s.game.Snapshot()
			if !v.ImportInProgress && int(v.EventsRead) >= want {
				return time.Since(begin)
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("service did not reach %d events: %+v", want, s.game.Snapshot())
		return 0
	}
	stop := func(s *Server) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.Close(ctx)
	}

	// Cold import of the whole fixture into an empty profile.
	perfstats.Reset()
	cpu0, _, _ := processSample()
	t0 := time.Now()
	s := start()
	wait(s, expected, 10*time.Minute)
	cold := time.Since(t0)
	cpu1, ws1, peak1 := processSample()
	result["coldImportSeconds"], result["coldImportCPUSeconds"], result["coldWorkingSetMB"], result["coldPeakWorkingSetMB"] = cold.Seconds(), cpu1-cpu0, ws1>>20, peak1>>20
	result["coldPerfstats"] = perfstats.Take()
	perfstats.Reset()
	stop(s)
	size := int64(0)
	_ = filepath.Walk(profile, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	result["profileBytesAfterImport"] = size

	// Incremental restart: 1k new events in a new journal file.
	extra := filepath.Join(journalDir, "Journal.2026-11-30T000000.01.log")
	f, err := os.Create(extra)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	line := func(i int, kind string) string {
		raw, _ := json.Marshal(map[string]any{"timestamp": stamp.Add(time.Duration(i) * time.Second).Format(time.RFC3339), "event": kind, "StarSystem": fmt.Sprintf("Perf Restart %d", i), "SystemAddress": 5000000 + i, "StarPos": []float64{float64(i), 0, 0}})
		return string(raw) + "\n"
	}
	_, _ = io.WriteString(f, `{"timestamp":"2026-11-30T00:00:00Z","event":"Fileheader","part":1,"gameversion":"4.1.0.100","build":"r300000/r0 "}`+"\n")
	_, _ = io.WriteString(f, `{"timestamp":"2026-11-30T00:00:00Z","event":"LoadGame","FID":"F9999001","Commander":"Perf Fixture","Ship":"anaconda","ShipID":7}`+"\n")
	for i := 0; i < 998; i++ {
		_, _ = io.WriteString(f, line(i+1, "FSDJump"))
	}
	f.Close()
	cpu0, _, _ = processSample()
	t0 = time.Now()
	s = start()
	begin := time.Now()
	for v := s.game.Snapshot(); v.ImportInProgress || v.CurrentSystem != "Perf Restart 998"; v = s.game.Snapshot() {
		if time.Since(begin) > 5*time.Minute {
			t.Fatalf("restart did not ingest the new events: %+v", v)
		}
		time.Sleep(5 * time.Millisecond)
	}
	result["incrementalRestartSeconds"], result["incrementalRestartCPUSeconds"] = time.Since(t0).Seconds(), func() float64 { c, _, _ := processSample(); return c - cpu0 }()

	// Live append latency: one FSDJump at a time, journal write to service
	// summary (the first host-side point every display reads).
	live, err := os.OpenFile(extra, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()
	stamp = time.Now().UTC().Add(-time.Second)
	latencies := []float64{}
	for i := 0; i < 40; i++ {
		before := s.game.Snapshot().EventsRead
		raw, _ := json.Marshal(map[string]any{"timestamp": time.Now().UTC().Format(time.RFC3339), "event": "FSDJump", "StarSystem": fmt.Sprintf("Perf Live %d", i), "SystemAddress": 7000000 + i, "StarPos": []float64{float64(i), 1, 0}})
		w := time.Now()
		if _, err = live.Write(append(raw, '\n')); err != nil {
			t.Fatal(err)
		}
		for s.game.Snapshot().EventsRead == before {
			if time.Since(w) > 10*time.Second {
				t.Fatal("live append not observed")
			}
			time.Sleep(time.Millisecond)
		}
		latencies = append(latencies, float64(time.Since(w).Microseconds())/1000)
		time.Sleep(150 * time.Millisecond)
	}
	sort.Float64s(latencies)
	result["liveAppendP50ms"], result["liveAppendP95ms"], result["liveAppendMaxms"] = latencies[len(latencies)/2], latencies[len(latencies)*95/100], latencies[len(latencies)-1]
	_, _, peak := processSample()
	result["runPeakWorkingSetMB"] = peak >> 20
	result["perfstats"] = perfstats.Take() // whole run: builds, checkpoints, lock timings
	stop(s)
	raw, _ := json.Marshal(result)
	fmt.Println("PERF_RESULT " + string(raw))
}
