package journal

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Phase string

const (
	Replay Phase = "replay"
	Live   Phase = "live"
)

type ReaderOptions struct {
	Durable        Durable
	PollInterval   time.Duration
	MaxRecordBytes int
	// HistoryReordered is called once when a backlog finished after older
	// journals committed new records behind the active one (Task 9): the
	// caller rebuilds derived state chronologically from durable history.
	HistoryReordered func()
}

// Bounded discovery (Task 9 / F11). Directories up to fullScanLimit journals
// are read completely on every poll, as before. Larger ones are read in
// bounded batches: the two newest journals (active and just rolled over)
// first, then up to defaultBacklogBatch unread journals in chronological
// order, then a round-robin slice of settled journals to detect changes.
var (
	maxDirectoryEntries = 65536
	fullScanLimit       = 512
	defaultBacklogBatch = 256
)

// SetDiscoveryLimitsForTest lowers the full-scan limit and batch size so a
// service-level test can exercise backlog mode with a few dozen journals
// instead of hundreds of durable commits. Tests only: call before starting a
// reader and restore with the returned function after it stopped.
func SetDiscoveryLimitsForTest(fullScan, batch int) (restore func()) {
	oldFull, oldBatch := fullScanLimit, defaultBacklogBatch
	fullScanLimit, defaultBacklogBatch = fullScan, batch
	return func() { fullScanLimit, defaultBacklogBatch = oldFull, oldBatch }
}

type ReaderStats struct {
	SourceState                                   string
	LastError                                     string
	Files                                         int
	EventsRead, MalformedLines, BytesRead, Resets uint64
	Pending, ImportInProgress                     bool
	// Backlog counts journals not yet read in this run (bounded discovery).
	Backlog int
}
type cursor struct {
	info                os.FileInfo
	offset, replayUntil int64
	id                  string
	sequence            uint64
	prefix, anchor      []byte
	discard, pending    bool
}
type Reader struct {
	currentFile string
	dir         string
	opts        ReaderOptions
	mu          sync.RWMutex
	stats       ReaderStats
	cursors     map[string]*cursor
	serial      uint64
	started     bool
	startedAt   time.Time
	roundRobin  int
	reordered   bool
	// unreadable holds journals whose open failed this run (permission,
	// exclusive lock): they are retried by the round-robin, not counted as
	// backlog, so one locked file cannot hold import open forever (review #2).
	unreadable map[string]bool
	running    atomic.Bool
}

func NewReader(dir string, opts ReaderOptions) (*Reader, error) {
	if dir != "" && !filepath.IsAbs(dir) {
		return nil, errors.New("journal directory must be absolute")
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 500 * time.Millisecond
	}
	if opts.MaxRecordBytes == 0 {
		opts.MaxRecordBytes = MaxRecordBytes
	}
	if opts.MaxRecordBytes < 1 || opts.MaxRecordBytes > MaxRecordBytes {
		return nil, errors.New("invalid journal record limit")
	}
	return &Reader{dir: dir, opts: opts, cursors: map[string]*cursor{}, startedAt: time.Now(), stats: ReaderStats{SourceState: "waiting", ImportInProgress: true}}, nil
}
func (r *Reader) Stats() ReaderStats          { r.mu.RLock(); defer r.mu.RUnlock(); return r.stats }
func (r *Reader) change(f func(*ReaderStats)) { r.mu.Lock(); defer r.mu.Unlock(); f(&r.stats) }
func (r *Reader) Run(ctx context.Context, emit func(Event, Phase) error) error {
	if !r.running.CompareAndSwap(false, true) {
		return errors.New("journal reader already running")
	}
	defer r.running.Store(false)
	ticker := time.NewTicker(r.opts.PollInterval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := r.poll(ctx, emit); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		r.started = true
		r.change(func(s *ReaderStats) { s.ImportInProgress = false })
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func journalLess(a, b string) bool {
	x, y := strings.Split(a, "."), strings.Split(b, ".")
	if x[1] != y[1] {
		return x[1] < y[1]
	}
	p, q := strings.TrimLeft(x[2], "0"), strings.TrimLeft(y[2], "0")
	if len(p) != len(q) {
		return len(p) < len(q)
	}
	if p != q {
		return p < q
	}
	return a < b
}
func (r *Reader) poll(ctx context.Context, emit func(Event, Phase) error) error {
	if r.dir == "" {
		return nil
	}
	f, err := os.Open(r.dir)
	if err != nil {
		r.sourceError(err)
		return nil
	}
	entries, err := f.ReadDir(maxDirectoryEntries + 1)
	f.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		r.sourceError(err)
		return nil
	}
	if len(entries) > maxDirectoryEntries {
		r.change(func(s *ReaderStats) {
			s.SourceState = "source_error"
			s.LastError = "journal directory exceeds " + strconv.Itoa(maxDirectoryEntries) + " entries"
		})
		return nil
	}
	var names []string
	for _, entry := range entries {
		if entry.Type().IsRegular() && journalName.MatchString(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	sort.Slice(names, func(i, j int) bool { return journalLess(names[i], names[j]) })
	r.change(func(s *ReaderStats) {
		s.Files = len(names)
		s.Pending = false
		s.LastError = ""
		if len(names) == 0 {
			s.SourceState = "waiting"
		} else {
			s.SourceState = "live"
		}
		if !r.started && len(names) > 0 {
			s.SourceState = "replaying"
		}
	})
	if len(names) > 0 {
		r.currentFile = names[len(names)-1]
	}
	current := map[string]bool{}
	for _, name := range names {
		current[name] = true
	}
	for name := range r.cursors {
		if !current[name] {
			delete(r.cursors, name)
		}
	}
	for name := range r.unreadable {
		if !current[name] {
			delete(r.unreadable, name)
		}
	}
	work, backlogMode := r.plan(names)
	for _, name := range work {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		older := backlogMode && journalLess(name, names[max(0, len(names)-2)]) // behind the hot journals
		before := r.Stats().EventsRead                                         // counts emitted and durably committed records
		if err := r.readFile(ctx, name, emit); err != nil {
			return err
		}
		if older && r.Stats().EventsRead > before {
			r.reordered = true // history committed behind the active journal
		}
		if r.cursors[name] == nil {
			if r.unreadable == nil {
				r.unreadable = map[string]bool{}
			}
			r.unreadable[name] = true
		} else {
			delete(r.unreadable, name)
		}
	}
	backlog := 0
	for _, name := range names {
		if r.cursors[name] == nil && !r.unreadable[name] {
			backlog++
		}
	}
	// The reorder callback runs before the empty backlog is published, so a
	// caller never observes "backlog drained" without the pending rebuild.
	if backlog == 0 && r.reordered {
		r.reordered = false
		if r.opts.HistoryReordered != nil {
			r.opts.HistoryReordered()
		}
	}
	r.change(func(s *ReaderStats) { s.Backlog = backlog })
	return nil
}

// plan chooses this poll's journals (chronological names). Small directories
// are read completely; large ones in bounded batches with the newest
// journals first so live ingestion and rollover never wait for history.
func (r *Reader) plan(names []string) ([]string, bool) {
	if len(names) <= fullScanLimit {
		return names, false
	}
	hot := names[max(0, len(names)-2):]
	work := append([]string(nil), hot...)
	inWork := map[string]bool{}
	for _, n := range hot {
		inWork[n] = true
	}
	unread := 0
	for _, n := range names {
		if unread >= defaultBacklogBatch {
			break
		}
		if r.cursors[n] == nil && !r.unreadable[n] && !inWork[n] {
			work = append(work, n)
			inWork[n] = true
			unread++
		}
	}
	// Round-robin over settled journals: changes to old journals are still
	// detected, within len(names)/defaultBacklogBatch polls.
	for k := 0; k < defaultBacklogBatch && k < len(names); k++ {
		n := names[(r.roundRobin+k)%len(names)]
		if !inWork[n] && (r.cursors[n] != nil || r.unreadable[n]) {
			work = append(work, n)
			inWork[n] = true
		}
	}
	r.roundRobin = (r.roundRobin + defaultBacklogBatch) % len(names)
	return work, unread > 0
}
func (r *Reader) sourceError(err error) {
	r.change(func(s *ReaderStats) {
		s.Pending = false
		s.Files = 0
		s.SourceState = "source_error"
		s.LastError = "journal source unavailable"
		if errors.Is(err, os.ErrNotExist) {
			s.SourceState = "waiting"
			s.LastError = ""
		}
		if errors.Is(err, os.ErrPermission) {
			s.SourceState = "permission_error"
			s.LastError = "journal source unreadable"
		}
	})
}
func sample(f *os.File, offset int64, size int) []byte {
	if offset < 0 {
		return nil
	}
	b := make([]byte, size)
	n, _ := f.ReadAt(b, offset)
	return b[:n]
}
func (r *Reader) readFile(ctx context.Context, name string, emit func(Event, Phase) error) error {
	path := filepath.Join(r.dir, name)
	info, err := os.Lstat(path)
	if err != nil {
		r.sourceError(err)
		return nil
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	c := r.cursors[name]
	if (r.opts.Durable == nil || name != r.currentFile) && c != nil && os.SameFile(c.info, info) && c.info.Size() == info.Size() && c.info.ModTime() == info.ModTime() {
		if c.pending {
			r.change(func(s *ReaderStats) { s.Pending = true })
		}
		return nil
	}
	f, err := OpenReadOnly(path)
	if err != nil {
		r.sourceError(err)
		return nil
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !actual.Mode().IsRegular() {
		return nil
	}
	info = actual
	prefix := sample(f, 0, int(min(info.Size(), 128)))
	reset := c == nil
	if c != nil {
		reset = !os.SameFile(c.info, info) || info.Size() < c.offset || len(prefix) < len(c.prefix) || !bytes.Equal(prefix[:min(len(prefix), len(c.prefix))], c.prefix)
		if !reset && c.offset > 0 {
			anchor := sample(f, c.offset-int64(len(c.anchor)), len(c.anchor))
			reset = !bytes.Equal(anchor, c.anchor)
		}
	}
	var source Source
	if r.opts.Durable != nil {
		identity, err := durableIdentity(f)
		if err != nil {
			return err
		}
		source = Source{ID: fmt.Sprintf("%x", sha256.Sum256([]byte(identity))), Identity: identity, Path: path, Size: info.Size()}
		if c == nil {
			saved, err := r.opts.Durable.Resume(source)
			if err != nil {
				return err
			}
			source.ID = saved.Source.ID
			if saved.Offset == 0 && saved.Sequence == 0 {
				if resolver, ok := r.opts.Durable.(interface {
					ResolveCopy(Source, []byte, []byte, *os.File) (Checkpoint, bool, error)
				}); ok {
					resolved, match, err := resolver.ResolveCopy(source, prefix, sample(f, max(0, info.Size()-128), int(min(info.Size(), 128))), f)
					if err != nil {
						return err
					}
					if match {
						saved = resolved
					}
				}
			}
			if saved.Offset > info.Size() || len(prefix) < len(saved.Head) || !bytes.Equal(prefix[:min(len(prefix), len(saved.Head))], saved.Head) || !bytes.Equal(sample(f, saved.Offset-int64(len(saved.Anchor)), len(saved.Anchor)), saved.Anchor) {
				if q, ok := r.opts.Durable.(interface{ RejectSource(Source) error }); ok {
					_ = q.RejectSource(source)
				}
				return errors.New("journal source changed; private history retained for recovery")
			}
			source.ID = saved.Source.ID
			c = &cursor{id: source.ID, offset: saved.Offset, sequence: saved.Sequence, discard: saved.Discard, replayUntil: info.Size()}
			if r.started && saved.Offset == 0 {
				c.replayUntil = 0
			}
			r.cursors[name] = c
			reset = false
		} else if reset {
			source.ID = c.id
			if q, ok := r.opts.Durable.(interface{ RejectSource(Source) error }); ok {
				_ = q.RejectSource(source)
			}
			return errors.New("journal source changed; private history retained for recovery")
		}
	}
	if r.opts.Durable != nil && c != nil {
		source.ID = c.id
	}
	if reset {
		r.serial++
		c = &cursor{id: strconv.FormatUint(r.serial, 10)}
		if !r.started {
			c.replayUntil = info.Size()
		}
		if r.cursors[name] != nil {
			r.change(func(s *ReaderStats) { s.Resets++ })
		}
		r.cursors[name] = c
	}
	c.prefix = prefix
	c.pending = false
	if _, err = f.Seek(c.offset, io.SeekStart); err != nil {
		r.sourceError(err)
		return nil
	}
	reader := bufio.NewReaderSize(io.LimitReader(f, max(0, info.Size()-c.offset)), 64<<10)
	position := c.offset
	start := c.offset
	var line []byte
	var batch []Record
	checkpointOffset := c.offset
	commit := func() error {
		if r.opts.Durable == nil {
			return nil
		}
		if c.offset == checkpointOffset && len(batch) == 0 {
			return nil
		}
		cp := Checkpoint{Source: source, Offset: c.offset, Sequence: c.sequence, Head: sample(f, 0, int(min(c.offset, 128))), Anchor: sample(f, max(0, c.offset-128), int(min(c.offset, 128))), Discard: c.discard}
		if err := r.opts.Durable.CommitRecords(batch, cp); err != nil {
			return err
		}
		batch = nil
		checkpointOffset = c.offset
		return nil
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		fragment, readErr := reader.ReadSlice('\n')
		position += int64(len(fragment))
		r.change(func(s *ReaderStats) { s.BytesRead += uint64(len(fragment)) })
		if !c.discard {
			if len(line)+len(fragment) > r.opts.MaxRecordBytes+1 {
				c.discard = true
				line = nil
				r.change(func(s *ReaderStats) { s.MalformedLines++ })
			} else {
				line = append(line, fragment...)
			}
		}
		complete := len(fragment) > 0 && fragment[len(fragment)-1] == '\n'
		if complete {
			c.sequence++
			if !c.discard {
				e, eventErr := ParseLine(bytes.TrimSuffix(line, []byte("\n")), SourcePosition{Path: path, FileID: c.id, Offset: start, Sequence: c.sequence, Length: position - start})
				if eventErr != nil {
					r.change(func(s *ReaderStats) { s.MalformedLines++ })
				} else {
					phase := Live
					if position <= c.replayUntil || (r.opts.Durable != nil && !e.Timestamp.IsZero() && e.Timestamp.Before(r.startedAt)) {
						phase = Replay
					}
					if r.opts.Durable != nil {
						batch = append(batch, Record{Event: e, Phase: phase})
					} else if err := emit(e, phase); err != nil {
						return err
					}
					r.change(func(s *ReaderStats) { s.EventsRead++ })
				}
			}
			c.offset = position
			start = position
			c.discard = false
			line = nil
			if len(batch) >= 256 {
				if err := commit(); err != nil {
					return err
				}
			}
		} else if c.discard {
			c.offset = position
		}
		if readErr != nil {
			if errors.Is(readErr, bufio.ErrBufferFull) {
				continue
			}
			if !errors.Is(readErr, io.EOF) {
				r.sourceError(readErr)
			}
			c.pending = position > start || c.discard
			if c.pending {
				r.change(func(s *ReaderStats) { s.Pending = true })
			}
			break
		}
	}
	if err := commit(); err != nil {
		return err
	}
	c.info = info
	c.anchor = sample(f, max(0, c.offset-128), int(min(c.offset, 128)))
	return nil
}
