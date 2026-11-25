package history

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Jakoczig/eridion/go/internal/privatefs"
)

// restoreFixture opens a store whose live cache reads "original" and returns a
// verified private backup whose cache reads "backup".
func restoreFixture(t *testing.T) (dir, backup string) {
	t.Helper()
	dir = t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err = s.SaveCache(ctx, []byte("backup")); err != nil {
		t.Fatal(err)
	}
	backupDir := t.TempDir()
	privatefs.Prepare(backupDir)
	backup = filepath.Join(backupDir, "selected.sqlite")
	if _, err = s.Backup(ctx, backup); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveCache(ctx, []byte("original")); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	return dir, backup
}

// cacheAfterOpen opens the store (consuming any staged restore) and returns its cache.
func cacheAfterOpen(t *testing.T, dir string) string {
	t.Helper()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	raw, err := s.Cache()
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func assertNoStageResidue(t *testing.T, dir string) {
	t.Helper()
	for _, pattern := range []string{"restore.pending.*", "restore.staging-*"} {
		if m, _ := filepath.Glob(filepath.Join(dir, pattern)); len(m) != 0 {
			t.Fatalf("failed staging left %v", m)
		}
	}
}

// TestAuditRestoreCancelAfterStagingRetry (Task 9): cancellation after the
// staging file was created left restore.pending.sqlite behind and every retry
// failed (exclusive create). The original stays intact and a retry succeeds.
func TestAuditRestoreCancelAfterStagingRetry(t *testing.T) {
	dir, backup := restoreFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := StageRecovery(ctx, dir, backup); err == nil {
		t.Fatal("cancelled staging reported success")
	}
	assertNoStageResidue(t, dir)
	if err := StageRecovery(context.Background(), dir, backup); err != nil {
		t.Fatalf("retry after cancellation refused: %v", err)
	}
	if got := cacheAfterOpen(t, dir); got != "backup" {
		t.Fatalf("restore after retry not applied: %q", got)
	}
}

// Filesystem/verification failures at each step after staging begins leave the
// original database untouched, remove only this attempt's own files, and allow
// a successful retry.
func TestAuditRestoreFaultAfterStagingRetry(t *testing.T) {
	for _, step := range []string{"copied", "consent", "verified", "promote"} {
		t.Run(step, func(t *testing.T) {
			dir, backup := restoreFixture(t)
			restoreFault = func(s string) error {
				if s == step {
					return errors.New("injected filesystem error")
				}
				return nil
			}
			err := StageRecovery(context.Background(), dir, backup)
			restoreFault = nil
			if err == nil {
				t.Fatal("injected failure reported success")
			}
			assertNoStageResidue(t, dir)
			if got := cacheAfterOpen(t, dir); got != "original" {
				t.Fatalf("original database changed by a failed staging: %q", got)
			}
			if err = StageRecovery(context.Background(), dir, backup); err != nil {
				t.Fatalf("retry refused: %v", err)
			}
			if got := cacheAfterOpen(t, dir); got != "backup" {
				t.Fatalf("retry not applied: %q", got)
			}
		})
	}
}

// An unexplained staging file (no matching intent, e.g. left by an older build)
// is never overwritten or deleted: it is moved aside byte-for-byte and the new
// stage proceeds. An explained, complete stage is refused rather than replaced.
func TestAuditRestoreUnexplainedStageNeverOverwritten(t *testing.T) {
	dir, backup := restoreFixture(t)
	stale := filepath.Join(dir, "restore.pending.sqlite")
	f, err := privatefs.PrivateFile(stale, true)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("unexplained private bytes")
	f.Close()
	if err = StageRecovery(context.Background(), dir, backup); err != nil {
		t.Fatalf("stage beside an unexplained file: %v", err)
	}
	kept, _ := filepath.Glob(filepath.Join(dir, "backups", "unexplained-stage-*", "restore.pending.sqlite"))
	if len(kept) != 1 {
		t.Fatalf("unexplained stage not preserved: %v", kept)
	}
	if raw, _ := os.ReadFile(kept[0]); string(raw) != "unexplained private bytes" {
		t.Fatalf("unexplained stage altered: %q", raw)
	}
	before, _ := os.ReadFile(stale)
	if err = StageRecovery(context.Background(), dir, backup); err == nil {
		t.Fatal("a complete pending restore was replaced")
	}
	if after, _ := os.ReadFile(stale); string(after) != string(before) {
		t.Fatal("complete pending restore modified by a refused retry")
	}
	if got := cacheAfterOpen(t, dir); got != "backup" {
		t.Fatalf("pending restore not applied: %q", got)
	}
}

// A process killed during staging leaves only this code's own temporary copy;
// the next staging removes it and succeeds, and Open ignores it.
func TestAuditRestoreKilledDuringStaging(t *testing.T) {
	dir, backup := restoreFixture(t)
	temp := filepath.Join(dir, "restore.staging-killed.sqlite")
	f, err := privatefs.PrivateFile(temp, true)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("partial copy")
	f.Close()
	if got := cacheAfterOpen(t, dir); got != "original" {
		t.Fatalf("partial staging affected startup: %q", got)
	}
	if err = StageRecovery(context.Background(), dir, backup); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(temp); !os.IsNotExist(err) {
		t.Fatal("stale partial copy kept")
	}
	if got := cacheAfterOpen(t, dir); got != "backup" {
		t.Fatalf("%q", got)
	}
}

// Restart interrupted inside consumeRestore after the staged file replaced the
// database but before the intent was removed: the next start verifies the
// database against the intent and completes; nothing is applied twice.
func TestAuditRestoreInterruptedRestartRecovers(t *testing.T) {
	dir, backup := restoreFixture(t)
	if err := StageRecovery(context.Background(), dir, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "restore.pending.sqlite"), filepath.Join(dir, "daedalus.sqlite")); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		os.Remove(filepath.Join(dir, "daedalus.sqlite"+suffix))
	}
	if got := cacheAfterOpen(t, dir); got != "backup" {
		t.Fatalf("%q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "restore.pending.json")); !os.IsNotExist(err) {
		t.Fatal("intent not completed")
	}
	if got := cacheAfterOpen(t, dir); got != "backup" {
		t.Fatalf("second start: %q", got)
	}
}

// Review #4: SQLite sidecars left without a staged database are moved aside,
// never replayed into the next stage.
func TestAuditRestoreOrphanSidecarsQuarantined(t *testing.T) {
	dir, backup := restoreFixture(t)
	wal := filepath.Join(dir, "restore.pending.sqlite-wal")
	f, err := privatefs.PrivateFile(wal, true)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("stale wal")
	f.Close()
	if err = StageRecovery(context.Background(), dir, backup); err != nil {
		t.Fatal(err)
	}
	kept, _ := filepath.Glob(filepath.Join(dir, "backups", "unexplained-stage-*", "restore.pending.sqlite-wal"))
	if len(kept) != 1 {
		t.Fatalf("orphan WAL not preserved aside: %v", kept)
	}
	if got := cacheAfterOpen(t, dir); got != "backup" {
		t.Fatalf("%q", got)
	}
}

// Review #3: concurrent staging requests are serialised: exactly one stage is
// installed, the other is refused as "already staged", and nothing is lost.
func TestAuditRestoreConcurrentStaging(t *testing.T) {
	dir, backup := restoreFixture(t)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { errs <- StageRecovery(context.Background(), dir, backup) }()
	}
	ok := 0
	for i := 0; i < 2; i++ {
		if <-errs == nil {
			ok++
		}
	}
	if ok != 1 {
		t.Fatalf("%d concurrent stagings succeeded, want exactly 1", ok)
	}
	if m, _ := filepath.Glob(filepath.Join(dir, "backups", "unexplained-stage-*")); len(m) != 0 {
		t.Fatalf("a concurrent attempt quarantined the other's stage: %v", m)
	}
	if got := cacheAfterOpen(t, dir); got != "backup" {
		t.Fatalf("%q", got)
	}
}
