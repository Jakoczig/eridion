package atlas

import (
	"container/list"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Jakoczig/eridion/go/internal/perfstats"
)

func installedSpatial(t *testing.T) *SpatialManager {
	t.Helper()
	files, key := spatialFixture(t)
	client := &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		b, ok := files[strings.TrimPrefix(r.URL.Path, "/atlas/spatial/")]
		code := 200
		if !ok {
			code = 404
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(string(b))), ContentLength: int64(len(b)), Header: http.Header{}}, nil
	})}
	m := newSpatial(context.Background(), t.TempDir(), client, key, time.Now)
	t.Cleanup(m.Close)
	if e := m.Configure(SpatialPreferences{Downloads: true}); e != nil {
		t.Fatal(e)
	}
	if e := m.Update(context.Background(), true); e != nil {
		t.Fatal(e)
	}
	return m
}

// TestPerfSpatialPlanningAndCache (G3K PERF-07/08).
func TestPerfSpatialPlanningAndCache(t *testing.T) {
	m := installedSpatial(t)
	q := SpatialQuery{Low: [3]float64{-2, -2, -2}, High: [3]float64{2, 2, 2}}

	// Planning validates assets but never decodes blocks into points.
	perfstats.Reset()
	m.ioMu.Lock()
	needs, pins := m.planMissing(q)
	m.ioMu.Unlock()
	if len(needs) != 0 || len(pins) == 0 {
		t.Fatalf("installed viewport planned as missing: %v", needs)
	}
	if c := perfstats.Take().Counters; c["spatial.decode"] != 0 {
		t.Fatalf("prefetch planning decoded %d blocks", c["spatial.decode"])
	}

	// The visible view decodes once; overlapping/repeated views reuse it.
	v, e := m.View(q, false)
	if e != nil || !v.Complete || len(v.Points) != 4 || !v.ViewportReady || v.ViewToken == "" {
		t.Fatalf("%+v %v", v, e)
	}
	decoded := perfstats.Take().Counters["spatial.decode"]
	if decoded == 0 {
		t.Fatal("visible view did not decode")
	}
	wider := SpatialQuery{Low: [3]float64{-3, -2, -2}, High: [3]float64{2, 2, 2}}
	w, e := m.View(wider, false)
	if e != nil || !w.Complete || len(w.Points) < 4 {
		t.Fatalf("%+v %v", w, e)
	}
	if again := perfstats.Take().Counters["spatial.decode"]; again != decoded {
		t.Fatalf("overlapping viewport decoded again: %d -> %d", decoded, again)
	}

	// Unchanged: the same complete view is confirmed without coordinates.
	q.Since = v.ViewToken
	u, e := m.View(q, false)
	if e != nil || !u.Unchanged || u.Points != nil || !u.Complete || u.ViewToken != v.ViewToken {
		t.Fatalf("unchanged view: %+v %v", u, e)
	}
	// A stale token (other query or release) gets the full view.
	q.Since = w.ViewToken
	if f, _ := m.View(q, false); f.Unchanged || len(f.Points) != 4 {
		t.Fatalf("stale token treated as unchanged: %+v", f)
	}
	if _, e := m.View(SpatialQuery{Low: q.Low, High: q.High, Since: strings.Repeat("x", 65)}, false); e == nil {
		t.Fatal("unbounded since accepted")
	}

	// Activation/rollback/removal clear derived caches; nothing is reused
	// across releases (entries are content-addressed anyway).
	m.ioMu.Lock()
	used := m.cache.used
	m.ioMu.Unlock()
	if used == 0 || used > decodedCacheBudget {
		t.Fatalf("decoded cache accounting: %d", used)
	}
	if e := m.RemoveBubble(true); e != nil {
		t.Fatal(e)
	}
	m.ioMu.Lock()
	cleared := m.cache.used == 0 && len(m.cache.items) == 0
	m.ioMu.Unlock()
	if !cleared {
		t.Fatal("Bubble removal kept derived caches")
	}
}

func TestPerfSpatialCacheBound(t *testing.T) {
	c := newSpatialCache()
	c.budget = 100
	c.put("a", 1, 60)
	c.put("b", 2, 60) // evicts a
	if _, ok := c.get("a"); ok || c.used != 60 {
		t.Fatalf("LRU bound: used=%d", c.used)
	}
	c.put("huge", 3, 101)
	if _, ok := c.get("huge"); ok {
		t.Fatal("an entry above the budget was cached")
	}
}

// Review R3: a block recorded as verified that later disappears from disk
// (decoded entry already dropped from the LRU) is planned for download again
// once a view finds it missing.
func TestPerfSpatialStaleVerifiedRecord(t *testing.T) {
	m := installedSpatial(t)
	q := SpatialQuery{Low: [3]float64{-2, -2, -2}, High: [3]float64{2, 2, 2}}
	if e := m.RemoveBubble(true); e != nil { // serve assets from disk only
		t.Fatal(e)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		v, _ := m.View(q, true) // owner view downloads the viewport
		if v.Complete {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("viewport download did not complete: %+v", v)
		}
		time.Sleep(20 * time.Millisecond)
	}
	// Drop decoded entries (as LRU eviction would) but keep verification,
	// then remove one block file behind the manager's back.
	m.ioMu.Lock()
	m.cache.order.Init()
	m.cache.items = map[string]*list.Element{}
	m.cache.used = 0
	var victim string
	for h := range m.cache.verified {
		if _, err := os.Stat(filepath.Join(m.root, "assets", "blocks", h+".z")); err == nil {
			victim = h
			break
		}
	}
	m.cachedKey = ""
	m.ioMu.Unlock()
	if victim == "" {
		t.Fatal("no downloaded block found")
	}
	if err := os.Remove(filepath.Join(m.root, "assets", "blocks", victim+".z")); err != nil {
		t.Fatal(err)
	}
	if v, _ := m.View(q, false); v.Complete {
		t.Fatalf("a missing block was reported complete: %+v", v)
	}
	m.ioMu.Lock()
	needs, _ := m.planMissing(q)
	m.ioMu.Unlock()
	if len(needs) == 0 {
		t.Fatal("planning trusted a stale verified record and planned no download")
	}
}
