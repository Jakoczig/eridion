package sidecar

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Jakoczig/eridion/go/internal/journal"
)

type Options struct {
	MaxBytes    int
	StationData bool
}
type Change struct {
	Kind      string
	Timestamp time.Time
	Valid     bool
	Raw       json.RawMessage
	State     string
}
type Reader struct {
	stationData bool
	dir         string
	limit       int
	mu          sync.Mutex
	good        map[string]Change
	hashes      map[string][32]byte
	states      map[string]string
	// verified caches the classification of the last stable read per kind,
	// keyed by its content hash, so identical bytes are not decoded again.
	verified map[string]verifiedRead
	decodes  int // envelope decodes performed; observed by tests
}

type verifiedRead struct {
	hash   [32]byte
	change Change
}

func NewReader(dir string, opts Options) (*Reader, error) {
	if dir != "" && !filepath.IsAbs(dir) {
		return nil, errors.New("sidecar directory must be absolute")
	}
	if opts.MaxBytes == 0 {
		opts.MaxBytes = 4 << 20
	}
	if opts.MaxBytes < 1 || opts.MaxBytes > 4<<20 {
		return nil, errors.New("invalid sidecar limit")
	}
	return &Reader{stationData: opts.StationData, dir: dir, limit: opts.MaxBytes, good: map[string]Change{}, hashes: map[string][32]byte{}, states: map[string]string{}, verified: map[string]verifiedRead{}}, nil
}
func clone(c Change) Change { c.Raw = append(json.RawMessage(nil), c.Raw...); return c }
func (r *Reader) LastGood(kind string) Change {
	r.mu.Lock()
	defer r.mu.Unlock()
	return clone(r.good[kind])
}
func (r *Reader) Poll(ctx context.Context) ([]Change, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var changes []Change
	kinds := []string{"Status", "Cargo", "Materials", "NavRoute"}
	if r.stationData {
		kinds = append(kinds, "Market", "Shipyard", "Outfitting")
	}
	for _, kind := range kinds {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		change, hash, reused := r.read(kind)
		if change.Valid && !change.Timestamp.IsZero() && change.Timestamp.Before(r.good[kind].Timestamp) {
			change.Valid = false
			change.State = "stale"
			change.Raw = nil
		}
		if r.states[kind] == change.State && r.hashes[kind] == hash {
			continue
		}
		r.states[kind] = change.State
		r.hashes[kind] = hash
		if reused {
			change = clone(change) // never share the cached bytes with callers
		}
		if change.Valid {
			r.good[kind] = clone(change)
		} else {
			change.Timestamp = r.good[kind].Timestamp
		}
		changes = append(changes, change)
	}
	return changes, nil
}

// read performs the bounded safe read, hashing and file-stability checks on
// every call. Only after those checks pass may an identical verified
// content hash reuse the previous classification instead of decoding and
// copying the bytes again; reused reports that case.
func (r *Reader) read(kind string) (Change, [32]byte, bool) {
	c := Change{Kind: kind, State: "missing"}
	var hash [32]byte
	if r.dir == "" {
		return c, hash, false
	}
	path := filepath.Join(r.dir, kind+".json")
	before, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			c.State = "permission_error"
		}
		return c, hash, false
	}
	if !before.Mode().IsRegular() {
		c.State = "unsafe"
		return c, hash, false
	}
	if before.Size() > int64(r.limit) {
		c.State = "too_large"
		return c, hash, false
	}
	f, err := journal.OpenReadOnly(path)
	if err != nil {
		c.State = "permission_error"
		return c, hash, false
	}
	raw, readErr := io.ReadAll(io.LimitReader(f, int64(r.limit)+1))
	after, statErr := f.Stat()
	f.Close()
	if readErr != nil || statErr != nil {
		c.State = "unreadable"
		return c, hash, false
	}
	hash = sha256.Sum256(raw)
	if len(raw) > r.limit {
		c.State = "too_large"
		return c, hash, false
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, after) || !os.SameFile(after, current) || before.Size() != after.Size() || before.ModTime() != after.ModTime() {
		c.State = "changing"
		return c, hash, false
	}
	if v, ok := r.verified[kind]; ok && v.hash == hash {
		return v.change, hash, true
	}
	defer func() { r.verified[kind] = verifiedRead{hash: hash, change: clone(c)} }()
	trimmed := bytes.TrimSpace(raw)
	c.State = "invalid"
	if len(trimmed) == 0 || trimmed[0] != '{' || !utf8.Valid(raw) {
		return c, hash, false
	}
	var envelope struct {
		Timestamp string
		Event     string
	}
	r.decodes++
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&envelope) != nil || decoder.Decode(new(any)) != io.EOF {
		return c, hash, false
	}
	if envelope.Event != "" && envelope.Event != kind {
		return c, hash, false
	}
	if envelope.Timestamp != "" {
		c.Timestamp, err = time.Parse(time.RFC3339Nano, envelope.Timestamp)
		if err != nil {
			return c, hash, false
		}
	}
	c.Valid = true
	c.State = "valid"
	c.Raw = append(json.RawMessage(nil), raw...)
	return c, hash, false
}
