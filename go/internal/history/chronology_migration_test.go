package history

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/privatefs"
)

// A database derived before chronological selection keeps every original
// event, gains observation keys, reclassifies Progress evidence and repairs
// the current construction row from retained history.
func TestAuditChronologyMigrationRepairsDerivedRows(t *testing.T) {
	dir := t.TempDir()
	if e := privatefs.Prepare(dir); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "daedalus.sqlite")
	f, e := privatefs.PrivateFile(path, true)
	if e != nil {
		t.Fatal(e)
	}
	f.Close()
	db, e := sql.Open("sqlite3", dsn(path))
	if e != nil {
		t.Fatal(e)
	}
	for i, m := range migrations[:6] {
		if _, e = db.Exec(m); e != nil {
			t.Fatal(e)
		}
		if _, e = db.Exec("INSERT INTO schema_migrations VALUES(?,?,?)", i+1, digest([]byte(m)), utc()); e != nil {
			t.Fatal(e)
		}
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := db.Exec(q, args...); e != nil {
			t.Fatal(q, e)
		}
	}
	exec("INSERT INTO commander VALUES('SYNTHETIC-A','Fixture',?)", utc())
	exec("INSERT INTO journal_sources(source_id,canonical_path,file_identity,observed_size,first_seen_at,last_seen_at) VALUES('nov','nov.log','nov',1,?,?),('oct','oct.log','oct',1,?,?)", utc(), utc(), utc(), utc())
	// Newer November evidence was ingested first; older October evidence later.
	events := []struct {
		id               int64
		source, at, kind string
		raw              string
	}{
		{1, "nov", "2026-11-22T10:00:01.000000000Z", "Rank", `{"event":"Rank","Combat":8}`},
		{2, "nov", "2026-11-22T10:00:05.000000000Z", "ColonisationConstructionDepot", `{"event":"ColonisationConstructionDepot","MarketID":3960000001,"ConstructionProgress":1.0,"ConstructionComplete":true,"ConstructionFailed":false}`},
		{3, "oct", "2026-10-22T10:00:01.000000000Z", "Progress", `{"event":"Progress","Combat":55}`},
		{4, "oct", "2026-10-22T10:00:05.000000000Z", "ColonisationConstructionDepot", `{"event":"ColonisationConstructionDepot","MarketID":3960000001,"ConstructionProgress":0.25,"ConstructionComplete":false,"ConstructionFailed":false}`},
	}
	for _, ev := range events {
		exec("INSERT INTO journal_events(event_id,source_id,byte_offset,byte_length,record_sequence,line_hash,event_utc,type,commander_fid,raw_relevant_json,ingested_at) VALUES(?,?,?,?,?,?,?,?,'SYNTHETIC-A',?,?)", ev.id, ev.source, ev.id*100, 100, ev.id, digest([]byte(ev.raw)), ev.at, ev.kind, ev.raw, utc())
	}
	exec("INSERT INTO field_provenance(entity,field,commander_fid,source,source_ref,observed_at,fetched_at,confidence,stale,value_json) VALUES('commander','ranks','SYNTHETIC-A','journal','1','2026-11-22T10:00:01.000000000Z',?,'witnessed local event',0,'{\"Combat\":\"8\"}')", utc())
	exec("INSERT INTO field_provenance(entity,field,commander_fid,source,source_ref,observed_at,fetched_at,confidence,stale,value_json) VALUES('commander','ranks','SYNTHETIC-A','journal','3','2026-10-22T10:00:01.000000000Z',?,'witnessed local event',0,'{\"Combat\":\"55\"}')", utc())
	// The wrong derived row written by the older unconditional UPSERT.
	exec("INSERT INTO colonisation_projects VALUES('SYNTHETIC-A','3960000001',NULL,0.25,0,0,'2026-10-22T10:00:05.000000000Z',4)")
	db.Close()

	s, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if s.Stats().Schema != schemaVersion {
		t.Fatal(s.Stats())
	}
	var progress float64
	var complete bool
	var source int64
	if e = s.reader.QueryRow("SELECT construction_progress,complete,source_event_id FROM colonisation_projects WHERE commander_fid='SYNTHETIC-A' AND market_id='3960000001'").Scan(&progress, &complete, &source); e != nil || progress != 1 || !complete || source != 2 {
		t.Fatalf("construction not repaired: %v %v %v %v", progress, complete, source, e)
	}
	var count int
	if e = s.reader.QueryRow("SELECT count(*) FROM journal_events").Scan(&count); e != nil || count != 4 {
		t.Fatalf("original events changed: %d %v", count, e)
	}
	facts, e := s.EffectiveFacts(context.Background(), "SYNTHETIC-A", "commander", time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC))
	if e != nil {
		t.Fatal(e)
	}
	got := map[string]string{}
	for _, f := range facts {
		got[f.Field] = string(f.Value)
	}
	if got["ranks"] != `{"Combat":"8"}` || got["rankProgress"] != `{"Combat":"55"}` {
		t.Fatalf("rank evidence not reclassified: %v", got)
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "backups", "pre-migration-v6-*.sqlite"))
	if len(paths) != 1 {
		t.Fatal("missing migration backup", paths)
	}
}

// A failing backfill rolls back the migration and keeps the prior database.
func TestAuditChronologyMigrationFailurePreservesDatabase(t *testing.T) {
	dir := t.TempDir()
	if e := privatefs.Prepare(dir); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "daedalus.sqlite")
	f, e := privatefs.PrivateFile(path, true)
	if e != nil {
		t.Fatal(e)
	}
	f.Close()
	db, e := sql.Open("sqlite3", dsn(path))
	if e != nil {
		t.Fatal(e)
	}
	for i, m := range migrations[:6] {
		if _, e = db.Exec(m); e != nil {
			t.Fatal(e)
		}
		if _, e = db.Exec("INSERT INTO schema_migrations VALUES(?,?,?)", i+1, digest([]byte(m)), utc()); e != nil {
			t.Fatal(e)
		}
	}
	// A conflicting index name makes the chronology migration fail mid-way.
	if _, e = db.Exec("CREATE INDEX provenance_observed ON field_provenance(entity)"); e != nil {
		t.Fatal(e)
	}
	db.Close()
	if s, e := Open(dir); e == nil {
		s.Close()
		t.Fatal("migration unexpectedly succeeded")
	}
	db, e = sql.Open("sqlite3", dsn(path))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var version int
	if e = db.QueryRow("SELECT max(version) FROM schema_migrations").Scan(&version); e != nil || version != 6 {
		t.Fatalf("version = %d (%v)", version, e)
	}
	var cols int
	if e = db.QueryRow("SELECT count(*) FROM pragma_table_info('field_provenance') WHERE name='observed_key'").Scan(&cols); e != nil || cols != 0 {
		t.Fatalf("partial migration left observed_key: %d %v", cols, e)
	}
}
