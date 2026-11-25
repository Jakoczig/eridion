package history

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/Jakoczig/eridion/go/internal/privatefs"
	sqlite "github.com/mattn/go-sqlite3"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type restoreIntent struct {
	Hash string `json:"sha256"`
}

func (s *Store) StageRestore(ctx context.Context, source string) error {
	return StageRecovery(ctx, s.dir, source)
}

// StageRecovery is also usable when the active DB is corrupt. It never opens it
// for mutation: only a verified selected backup is staged, pending an app restart.
func StageRecovery(ctx context.Context, dir, source string) error {
	if !filepath.IsAbs(source) {
		return errors.New("Select an absolute private backup path")
	}
	if err := privatefs.CheckPrivate(source); err != nil {
		return err
	}
	if err := privatefs.Prepare(dir); err != nil {
		return err
	}
	if _, err := verifyBackup(source); err != nil {
		return err
	}
	u := url.URL{Scheme: "file", Path: databaseURIPath(source)}
	db, err := sql.Open("sqlite3", u.String()+"?mode=ro&immutable=1&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	var v int
	var sum string
	if err = db.QueryRow("SELECT version,checksum FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&v, &sum); err != nil || v < 1 || v > len(migrations) || sum != digest([]byte(migrations[v-1])) {
		return errors.New("Backup schema is not supported")
	}
	pendingPath := filepath.Join(dir, "restore.pending.sqlite")
	if err = clearStaleStage(dir, pendingPath); err != nil {
		return err
	}
	// Each attempt stages into its own new file; on any failure (cancellation,
	// filesystem or verification error) only that file is removed, so the
	// original database is never touched and a retry is always possible (G3L).
	var nonce [8]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return err
	}
	destination := filepath.Join(dir, "restore.staging-"+hex.EncodeToString(nonce[:])+".sqlite")
	tmp, err := privatefs.PrivateFile(destination, true)
	if err != nil {
		return err
	}
	tmp.Close()
	removeStage := func() {
		for _, suffix := range []string{"-wal", "-shm", "-journal", ""} {
			os.Remove(destination + suffix)
		}
	}
	if err = stageInto(ctx, db, destination); err != nil {
		removeStage()
		return err
	}
	hash, err := verifyBackup(destination)
	if err == nil {
		err = fault("verified")
	}
	if err == nil {
		err = fault("promote")
	}
	if err == nil {
		err = os.Rename(destination, pendingPath)
	}
	if err != nil {
		removeStage()
		return err
	}
	raw, _ := json.Marshal(restoreIntent{Hash: hash})
	if err = privatefs.Write(dir, "restore.pending.json", raw); err != nil {
		// No intent: the stage is inert at startup; remove it so a retry is clean.
		os.Remove(pendingPath)
		return err
	}
	return nil
}

func fault(step string) error {
	if restoreFault != nil {
		return restoreFault(step)
	}
	return nil
}

// stageInto copies the verified backup into destination and pauses restored
// provider consent there.
func stageInto(ctx context.Context, db *sql.DB, destination string) error {
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", destination); err != nil {
		return err
	}
	if err := fault("copied"); err != nil {
		return err
	}
	pending, err := sql.Open("sqlite3", dsn(destination))
	if err != nil {
		return err
	}
	pending.SetMaxOpenConns(1)
	tx, err := pending.BeginTx(ctx, nil)
	if err == nil {
		if err = pauseRestoredConsent(tx); err == nil {
			err = fault("consent")
		}
		if err == nil {
			err = tx.Commit()
		} else {
			tx.Rollback()
		}
	}
	if err != nil {
		pending.Close()
		return err
	}
	return pending.Close()
}

// clearStaleStage prepares the staging slot. A complete stage that matches its
// intent is refused (restart to apply it). Any other staged file has no
// explanation and is moved aside unchanged, never overwritten or deleted.
// Temporary copies from a killed attempt are this code's own partial files and
// are removed.
func clearStaleStage(dir, pendingPath string) error {
	stale, _ := filepath.Glob(filepath.Join(dir, "restore.staging-*.sqlite*"))
	for _, p := range stale {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if _, err := os.Lstat(pendingPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if raw, err := privatefs.Read(dir, "restore.pending.json"); err == nil {
		var intent restoreIntent
		if json.Unmarshal(raw, &intent) == nil && len(intent.Hash) == 64 {
			if hash, e := verifyBackup(pendingPath); e == nil && hash == intent.Hash {
				return errors.New("A restore is already staged; restart ERIDION to apply it")
			}
		}
	}
	evidence := filepath.Join(dir, "backups", "unexplained-stage-"+time.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := privatefs.Prepare(filepath.Join(dir, "backups")); err != nil {
		return err
	}
	if err := privatefs.Prepare(evidence); err != nil {
		return err
	}
	for _, name := range []string{"restore.pending.sqlite", "restore.pending.sqlite-wal", "restore.pending.sqlite-shm", "restore.pending.json"} {
		src := filepath.Join(dir, name)
		if _, err := os.Lstat(src); os.IsNotExist(err) {
			continue
		}
		if err := os.Rename(src, filepath.Join(evidence, name)); err != nil {
			return err
		}
	}
	return nil
}
func consumeRestore(dir string) error {
	raw, err := privatefs.Read(dir, "restore.pending.json")
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var intent restoreIntent
	if json.Unmarshal(raw, &intent) != nil || len(intent.Hash) != 64 {
		return errors.New("Restore intent damaged; originals retained")
	}
	path := filepath.Join(dir, "daedalus.sqlite")
	staged := filepath.Join(dir, "restore.pending.sqlite")
	if _, err = os.Stat(staged); os.IsNotExist(err) {
		hash, e := verifyBackup(path)
		if e != nil || hash != intent.Hash {
			return errors.New("Interrupted restore requires owner recovery")
		}
		return os.Remove(filepath.Join(dir, "restore.pending.json"))
	}
	hash, err := verifyBackup(staged)
	if err != nil || hash != intent.Hash {
		return errors.New("Staged restore checksum/integrity failure; originals retained")
	}
	backupDir := filepath.Join(dir, "backups")
	if err = privatefs.Prepare(backupDir); err != nil {
		return err
	}
	if _, err = os.Stat(path); err == nil {
		db, e := sql.Open("sqlite3", dsn(path))
		if e != nil {
			return e
		}
		db.SetMaxOpenConns(1)
		e = db.Ping()
		var integrity string
		if e == nil {
			e = db.QueryRow("PRAGMA integrity_check").Scan(&integrity)
			if e == nil && integrity != "ok" {
				e = errors.New("existing integrity check failed")
			}
		}
		if e != nil {
			db.Close()
			var se sqlite.Error
			corrupt := errors.As(e, &se) && (se.Code == sqlite.ErrNotADB || se.Code == sqlite.ErrCorrupt)
			if !corrupt && integrity == "" {
				return errors.New("Existing database could not be opened safely; originals retained")
			}
			evidence := filepath.Join(backupDir, "corrupt-before-restore-"+time.Now().UTC().Format("20060102T150405.000000000Z"))
			if e = privatefs.Prepare(evidence); e != nil {
				return e
			}
			for _, suffix := range []string{"-wal", "-shm", ""} {
				src := path + suffix
				if _, e = os.Stat(src); os.IsNotExist(e) {
					continue
				}
				if e = privatefs.CheckPrivate(src); e != nil {
					return e
				}
				if e = os.Rename(src, filepath.Join(evidence, "daedalus.sqlite"+suffix)); e != nil {
					return e
				}
			}
		} else {
			holder := &Store{writer: db, dir: dir, path: path}
			dest := filepath.Join(backupDir, "pre-restore-"+time.Now().UTC().Format("20060102T150405.000000000Z")+".sqlite")
			_, e = holder.Backup(context.Background(), dest)
			if e == nil {
				var busy, log, done int
				e = db.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &log, &done)
				if busy != 0 {
					e = errors.New("Restore checkpoint busy")
				}
			}
			db.Close()
			if e != nil {
				return e
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err = os.Rename(staged, path); err != nil {
		return err
	}
	return os.Remove(filepath.Join(dir, "restore.pending.json"))
}

// restoreFault injects failures at staging steps in tests.
var restoreFault func(step string) error
