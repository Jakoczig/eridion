package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/journal"
)

func commitChronologySource(t *testing.T, s *Store, id string, lines []string) {
	t.Helper()
	src := journal.Source{ID: id, Identity: "synthetic-" + id}
	cp, err := s.Resume(src)
	if err != nil {
		t.Fatal(err)
	}
	records := make([]journal.Record, 0, len(lines))
	offset := int64(0)
	for i, line := range lines {
		e, err := journal.ParseLine([]byte(line), journal.SourcePosition{FileID: src.ID, Offset: offset, Sequence: uint64(i + 1), Length: int64(len(line) + 1)})
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, journal.Record{Event: e, Phase: journal.Replay})
		offset += int64(len(line) + 1)
	}
	cp.Offset, cp.Sequence = offset, uint64(len(lines))
	if err = s.Commit(context.Background(), cp, records, nil); err != nil {
		t.Fatal(err)
	}
}

func chronologyJournal(fid, day string, credits, fuel int, progress string, complete bool, combat, combatProgress int) []string {
	return []string{
		fmt.Sprintf(`{"timestamp":"%sT10:00:00Z","event":"Fileheader","part":1,"gameversion":"4.2.0.0"}`, day),
		fmt.Sprintf(`{"timestamp":"%sT10:00:01Z","event":"LoadGame","FID":"%s","Commander":"Fixture","Credits":%d}`, day, fid, credits),
		fmt.Sprintf(`{"timestamp":"%sT10:00:02Z","event":"Rank","Combat":%d,"Trade":1}`, day, combat),
		fmt.Sprintf(`{"timestamp":"%sT10:00:03Z","event":"Progress","Combat":%d,"Trade":10}`, day, combatProgress),
		fmt.Sprintf(`{"timestamp":"%sT10:00:04Z","event":"FuelScoop","Scooped":1.5,"Total":%d}`, day, fuel),
		fmt.Sprintf(`{"timestamp":"%sT10:00:05Z","event":"ColonisationConstructionDepot","MarketID":3960000001,"ConstructionProgress":%s,"ConstructionComplete":%t,"ConstructionFailed":false,"ResourcesRequired":[]}`, day, progress, complete),
	}
}

type chronologyView struct {
	credits, ranks, rankProgress string
	fuel                         float64
	progress                     float64
	complete                     bool
	rows                         int
}

func readChronologyView(t *testing.T, s *Store, fid string, now time.Time) chronologyView {
	t.Helper()
	facts, err := s.EffectiveFacts(context.Background(), fid, "commander", now)
	if err != nil {
		t.Fatal(err)
	}
	var out chronologyView
	for _, f := range facts {
		switch f.Field {
		case "credits":
			out.credits = string(f.Value)
		case "ranks":
			out.ranks = string(f.Value)
		case "rankProgress":
			out.rankProgress = string(f.Value)
		}
	}
	nav, err := s.fieldNavigationEvidenceAt(fid, now)
	if err != nil {
		t.Fatal(err)
	}
	if nav.FuelTotal != nil {
		out.fuel = *nav.FuelTotal
	}
	if err = s.reader.QueryRow("SELECT coalesce(construction_progress,0),coalesce(complete,0) FROM colonisation_projects WHERE commander_fid=? AND market_id='3960000001'", fid).Scan(&out.progress, &out.complete); err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	if err = s.reader.QueryRow("SELECT count(*) FROM journal_events WHERE commander_fid=?", fid).Scan(&out.rows); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAuditChronologyImportOrder(t *testing.T) {
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	newer := chronologyJournal("SYNTHETIC-A", "2026-11-22", 500, 32, "1.0", true, 8, 0)
	older := chronologyJournal("SYNTHETIC-A", "2026-10-22", 100, 4, "0.25", false, 5, 55)
	var views [2]chronologyView
	for i, order := range [][2][]string{{newer, older}, {older, newer}} {
		dir := t.TempDir()
		s, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		commitChronologySource(t, s, fmt.Sprintf("source-%d-a", i), order[0])
		commitChronologySource(t, s, fmt.Sprintf("source-%d-b", i), order[1])
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		if s, err = Open(dir); err != nil {
			t.Fatal(err)
		}
		views[i] = readChronologyView(t, s, "SYNTHETIC-A", now)
		var requests int
		if err = s.reader.QueryRow("SELECT count(*) FROM publication_candidates").Scan(&requests); err != nil {
			t.Fatal(err)
		}
		if requests != 0 {
			t.Fatalf("import synthesized %d publication candidates", requests)
		}
		s.Close()
	}
	for i, v := range views {
		if v.credits != `"500"` || v.fuel != 32 || v.progress != 1 || !v.complete || v.rows != 10 {
			t.Fatalf("order %d current view regressed: %+v", i, v)
		}
		var ranks map[string]any
		if err := json.Unmarshal([]byte(v.ranks), &ranks); err != nil || ranks["Combat"] != "8" {
			t.Fatalf("order %d rank level = %s (%v)", i, v.ranks, err)
		}
		if v.rankProgress == "" {
			t.Fatalf("order %d missing separate rank progress evidence", i)
		}
	}
	if views[0] != views[1] {
		t.Fatalf("import order changed current facts: %+v vs %+v", views[0], views[1])
	}
}

// A later Progress percentage must never overwrite the witnessed Rank level.
func TestAuditChronologyRankProgressSeparate(t *testing.T) {
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	commitChronologySource(t, s, "rank-source", []string{
		`{"timestamp":"2026-11-22T10:00:00Z","event":"Fileheader","part":1,"gameversion":"4.2.0.0"}`,
		`{"timestamp":"2026-11-22T10:00:01Z","event":"LoadGame","FID":"SYNTHETIC-A","Commander":"Fixture","Credits":1}`,
		`{"timestamp":"2026-11-22T10:00:02Z","event":"Rank","Combat":5}`,
		`{"timestamp":"2026-11-22T10:00:03Z","event":"Progress","Combat":55}`,
	})
	v := readChronologyView(t, s, "SYNTHETIC-A", now)
	if v.ranks != `{"Combat":"5","event":"Rank","timestamp":"2026-11-22T10:00:02Z"}` {
		t.Fatalf("rank level replaced by progress: %s", v.ranks)
	}
	if v.rankProgress != `{"Combat":"55","event":"Progress","timestamp":"2026-11-22T10:00:03Z"}` {
		t.Fatalf("rank progress = %s", v.rankProgress)
	}
}

func TestAuditChronologyTimestampEdges(t *testing.T) {
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// Fractional ordering, an implausible future record and a second
	// commander. The future record remains stored but cannot become current.
	commitChronologySource(t, s, "edge-b", []string{
		`{"timestamp":"2026-11-22T11:00:00.5Z","event":"LoadGame","FID":"SYNTHETIC-A","Commander":"Fixture","Credits":300}`,
	})
	commitChronologySource(t, s, "edge-a", []string{
		`{"timestamp":"2026-11-22T11:00:00Z","event":"LoadGame","FID":"SYNTHETIC-A","Commander":"Fixture","Credits":200}`,
		`{"timestamp":"2026-11-22T12:00:06Z","event":"LoadGame","FID":"SYNTHETIC-A","Commander":"Fixture","Credits":900}`,
		`{"timestamp":"2026-11-22T11:30:00Z","event":"LoadGame","FID":"SYNTHETIC-B","Commander":"Other","Credits":777}`,
	})
	facts, err := s.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range facts {
		if f.Field == "credits" && string(f.Value) != `"300"` {
			t.Fatalf("credits = %s, want fractional newer plausible 300", f.Value)
		}
	}
	// Once the clock passes the record, it becomes the plausible latest.
	facts, err = s.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range facts {
		if f.Field == "credits" && string(f.Value) != `"900"` {
			t.Fatalf("credits after clock advance = %s", f.Value)
		}
	}
	var stored int
	if err = s.reader.QueryRow("SELECT count(*) FROM field_provenance WHERE field='credits'").Scan(&stored); err != nil || stored != 4 {
		t.Fatalf("stored credit evidence = %d (%v)", stored, err)
	}
}

func TestAuditChronologyEqualTimeTieIsOrderIndependent(t *testing.T) {
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	line := func(c int) []string {
		return []string{fmt.Sprintf(`{"timestamp":"2026-11-22T11:00:00Z","event":"LoadGame","FID":"SYNTHETIC-A","Commander":"Fixture","Credits":%d}`, c)}
	}
	var got [2]string
	for i, order := range [][2]string{{"tie-a", "tie-b"}, {"tie-b", "tie-a"}} {
		s, err := Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		values := map[string]int{"tie-a": 10, "tie-b": 20}
		commitChronologySource(t, s, order[0], line(values[order[0]]))
		commitChronologySource(t, s, order[1], line(values[order[1]]))
		facts, err := s.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", now)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range facts {
			if f.Field == "credits" {
				got[i] = string(f.Value)
			}
		}
		s.Close()
	}
	if got[0] == "" || got[0] != got[1] {
		t.Fatalf("equal-time selection depends on ingestion order: %v", got)
	}
}

func TestValidObservedAt(t *testing.T) {
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	for raw, want := range map[string]bool{
		"":                               false,
		"garbage":                        false,
		"2026-11-22T12:00:05Z":           true,
		"2026-11-22T12:00:06Z":           false,
		"2026-11-22T11:59:59.123456789Z": true,
		"2026-11-22T13:00:00+02:00":      true,
	} {
		if _, ok := validObservedAt(raw, now); ok != want {
			t.Errorf("validObservedAt(%q) = %v, want %v", raw, ok, want)
		}
	}
	if observationKey("2026-11-22T11:00:00.5Z") <= observationKey("2026-11-22T11:00:00Z") {
		t.Fatal("observation keys must order fractional instants correctly")
	}
}
