package journal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func backlogName(i int) string { return fmt.Sprintf("Journal.2026%06d.01.log", i) }

func writeJournal(t testing.TB, dir string, i int, systems ...string) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(dir, backlogName(i)), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, s := range systems {
		fmt.Fprintf(f, `{"timestamp":"2026-01-01T00:00:00Z","event":"FSDJump","StarSystem":%q}`+"\n", s)
	}
}

// TestAuditDiscoveryBacklog (Task 9 / F11): a journal directory with more than
// 4,096 entries (journals plus other files) is ingested in bounded batches.
// The active journal, its appends and a rollover are read while the
// historical backlog is still incomplete; every event is read exactly once.
func TestAuditDiscoveryBacklog(t *testing.T) {
	dir := t.TempDir()
	const files = 4200
	for i := 1; i <= files; i++ {
		writeJournal(t, dir, i, fmt.Sprintf("Old %d", i))
	}
	for i := 0; i < 50; i++ { // a mixed directory: other game files
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("Screenshot_%04d.bmp", i)), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r, err := NewReader(dir, ReaderOptions{PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	emit := func(e Event, _ Phase) error {
		seen[e.SystemName]++
		return nil
	}
	ctx := context.Background()
	if err := r.poll(ctx, emit); err != nil {
		t.Fatal(err)
	}
	st := r.Stats()
	if st.SourceState == "source_error" {
		t.Fatalf("a large journal directory stopped ingestion: %+v", st)
	}
	if seen[fmt.Sprintf("Old %d", files)] != 1 {
		t.Fatal("the active (newest) journal was not read while the backlog remained")
	}
	if st.Backlog == 0 || st.Backlog >= files {
		t.Fatalf("backlog not reported or not bounded per poll: %+v", st)
	}
	// Appends to the active journal and a rollover during the backlog.
	writeJournal(t, dir, files, "Live append")
	writeJournal(t, dir, files+1, "Rolled over")
	if err := r.poll(ctx, emit); err != nil {
		t.Fatal(err)
	}
	if seen["Live append"] != 1 || seen["Rolled over"] != 1 {
		t.Fatalf("active appends/rollover waited for the backlog: %v %v (backlog %d)", seen["Live append"], seen["Rolled over"], r.Stats().Backlog)
	}
	var mem runtime.MemStats
	polls := 2
	start := time.Now()
	for r.Stats().Backlog > 0 {
		if err := r.poll(ctx, emit); err != nil {
			t.Fatal(err)
		}
		polls++
		if polls > files {
			t.Fatal("backlog never drained")
		}
	}
	runtime.ReadMemStats(&mem)
	t.Logf("drained %d journals in %d polls (%v); heap in use %d KiB", files+1, polls, time.Since(start).Round(time.Millisecond), mem.HeapInuse>>10)
	for i := 1; i <= files; i++ {
		if n := seen[fmt.Sprintf("Old %d", i)]; n != 1 {
			t.Fatalf("journal %d read %d times", i, n)
		}
	}
	// A later poll reads nothing again (no duplicates), and a change to an
	// old journal is still detected within a bounded number of polls.
	before := len(seen)
	writeJournal(t, dir, 7, "Late write to an old journal")
	for i := 0; i < files/defaultBacklogBatch+4 && seen["Late write to an old journal"] == 0; i++ {
		if err := r.poll(ctx, emit); err != nil {
			t.Fatal(err)
		}
	}
	if seen["Late write to an old journal"] != 1 || len(seen) != before+1 {
		t.Fatal("old-journal change not detected exactly once")
	}
}

// Beyond the hard bound the source is refused explicitly (bounded memory).
func TestAuditDiscoveryHardBound(t *testing.T) {
	old := maxDirectoryEntries
	maxDirectoryEntries = 100
	defer func() { maxDirectoryEntries = old }()
	dir := t.TempDir()
	for i := 1; i <= 101; i++ {
		writeJournal(t, dir, i, "x")
	}
	r, _ := NewReader(dir, ReaderOptions{PollInterval: time.Millisecond})
	if err := r.poll(context.Background(), func(Event, Phase) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if st := r.Stats(); st.SourceState != "source_error" || st.LastError == "" {
		t.Fatalf("%+v", st)
	}
}

// Review #2: a journal that cannot be opened (permissions, an exclusive lock)
// is retried, but it is not counted as backlog forever, so it cannot hold the
// import open and block default-commander validation.
func TestAuditDiscoveryUnreadableJournalNotBacklog(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("POSIX permission fixture")
	}
	dir := t.TempDir()
	for i := 1; i <= 3; i++ {
		writeJournal(t, dir, i, fmt.Sprintf("Old %d", i))
	}
	locked := filepath.Join(dir, backlogName(2))
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(locked, 0o600)
	r, _ := NewReader(dir, ReaderOptions{PollInterval: time.Millisecond})
	seen := map[string]int{}
	emit := func(e Event, _ Phase) error { seen[e.SystemName]++; return nil }
	if err := r.poll(context.Background(), emit); err != nil {
		t.Fatal(err)
	}
	if b := r.Stats().Backlog; b != 0 {
		t.Fatalf("an unreadable journal is counted as backlog forever: %d", b)
	}
	os.Chmod(locked, 0o600)
	if err := r.poll(context.Background(), emit); err != nil {
		t.Fatal(err)
	}
	if seen["Old 2"] != 1 || seen["Old 1"] != 1 || seen["Old 3"] != 1 {
		t.Fatalf("readable again, but not read exactly once: %v", seen)
	}
}
