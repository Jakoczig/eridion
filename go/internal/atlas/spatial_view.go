package atlas

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jakoczig/eridion/go/internal/perfstats"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type SpatialQuery struct {
	Low    [3]float64   `json:"low"`
	High   [3]float64   `json:"high"`
	Route  [][3]float64 `json:"route,omitempty"`
	Target *[3]float64  `json:"target,omitempty"`
	// Since is the viewToken of the view the client already holds; an
	// unchanged complete view is then answered without coordinates.
	Since string `json:"since,omitempty"`
}
type SpatialView struct {
	Points   [][3]float64 `json:"points"`
	Complete bool         `json:"complete"`
	Pending  bool         `json:"pending"`
	Notice   string       `json:"notice"`
	Source   string       `json:"source"`
	Release  string       `json:"release"`
	Indexes  int          `json:"indexes"`
	Blocks   int          `json:"blocks"`
	Decoded  int64        `json:"decodedBytes"`
	Missing  int          `json:"missing"`
	// G3K PERF-07: ViewportReady means the requested viewport is complete;
	// PrefetchPending reports background adjacent/route downloads only.
	// Unchanged replies carry no points: they confirm the client's view
	// (identified by ViewToken), they are not an empty map.
	ViewToken       string `json:"viewToken"`
	Unchanged       bool   `json:"unchanged,omitempty"`
	ViewportReady   bool   `json:"viewportReady"`
	PrefetchPending bool   `json:"prefetchPending"`
}
type spatialNeed struct {
	path  string
	bytes int64
	hash  string
}

func validSpatialQuery(q SpatialQuery) bool {
	for i := range 3 {
		if math.IsNaN(q.Low[i]) || math.IsInf(q.Low[i], 0) || math.IsNaN(q.High[i]) || math.IsInf(q.High[i], 0) || q.Low[i] < -1e6 || q.High[i] > 1e6 || q.Low[i] >= q.High[i] {
			return false
		}
	}
	if len(q.Since) > 64 {
		return false
	}
	if len(q.Route) > 5 {
		return false
	}
	points := append([][3]float64(nil), q.Route...)
	if q.Target != nil {
		points = append(points, *q.Target)
	}
	for _, p := range points {
		for _, x := range p {
			if math.IsNaN(x) || math.IsInf(x, 0) || math.Abs(x) > 1e6 {
				return false
			}
		}
	}
	return true
}
func spatialQueryKey(q SpatialQuery, route bool) string {
	q.Since = ""
	if !route {
		q.Route = nil
		q.Target = nil
	}
	b, _ := json.Marshal(q)
	return string(b)
}
func sphereContains(q SpatialQuery, radius int64) bool {
	var sq float64
	for i := range 3 {
		x := math.Max(math.Abs(q.Low[i]), math.Abs(q.High[i]))
		sq += x * x
	}
	return sq <= float64(radius*radius)
}
func (m *SpatialManager) asset(need spatialNeed) ([]byte, error) {
	if m.bubble != nil {
		if member := m.bubble.members[need.path[len("assets/"):]]; member != nil {
			b, e := m.bubble.read(member.Name, need.bytes)
			if e == nil && int64(len(b)) == need.bytes && digest(b) == need.hash {
				return b, nil
			}
		}
	}
	b, e := readFile(m.root, need.path, need.bytes)
	if e != nil || int64(len(b)) != need.bytes || digest(b) != need.hash {
		return nil, errors.New("Spatial asset unavailable or corrupt")
	}
	p := filepath.Join(m.root, filepath.FromSlash(need.path))
	_ = os.Chtimes(p, m.now(), m.now())
	return b, nil
}

// ioMu serializes lazy ZIP access, eviction, activation, and bounded query scratch.
// mu is never held while acquiring ioMu, making shutdown/configuration cancellable.
func (m *SpatialManager) query(q SpatialQuery) (SpatialView, []spatialNeed, map[string]bool) {
	return m.walk(q, true)
}

// planMissing identifies the assets a query still needs without decoding
// any point (G3K PERF-08): indexes are parsed (or reused), blocks are only
// validated (length and hash) when not already decoded or verified.
func (m *SpatialManager) planMissing(q SpatialQuery) ([]spatialNeed, map[string]bool) {
	_, needs, pins := m.walk(q, false)
	return needs, pins
}

// walk resolves q against the selected manifest. With decode it also builds
// the visible points; limits, corruption and missing-asset handling are the
// same in both modes. Caller holds ioMu.
func (m *SpatialManager) walk(q SpatialQuery, decode bool) (SpatialView, []spatialNeed, map[string]bool) {
	if m.cache == nil {
		m.cache = newSpatialCache()
	}
	v := SpatialView{Points: [][3]float64{}, Source: "EDSM reported XYZ · 1/32 LY"}
	pins := map[string]bool{}
	needs := []spatialNeed{}
	manifest := m.current
	if m.bubble != nil && sphereContains(q, m.bubble.manifest.RadiusLY) {
		manifest = m.bubble.manifest
	}
	if manifest == nil {
		v.Notice = "Spatial data not installed or selected volume outside offline Bubble; density fallback"
		return v, needs, pins
	}
	m.mu.Lock()
	v.Release = m.state.Installed
	if manifest == m.bubbleManifest() {
		v.Release = m.state.BubbleRelease
	}
	m.mu.Unlock()
	v.Source += " · " + manifest.Source.Label + " · source " + manifest.Source.SHA256[:min(len(manifest.Source.SHA256), 12)]
	lo, hi := [3]int32{}, [3]int32{}
	for i := range 3 {
		lo[i] = int32(math.Ceil(q.Low[i] * 32))
		hi[i] = int32(math.Ceil(q.High[i] * 32))
	}
	var indexBytes int64
	var descriptors []spatialBlock
	for _, r := range manifest.Roots {
		a, b := [3]int32{}, [3]int32{}
		for i := range 3 {
			a[i] = int32(r.Cell[i] * manifest.RootLY * 32)
			b[i] = int32((r.Cell[i] + 1) * manifest.RootLY * 32)
		}
		if !overlap(lo, hi, a, b) {
			continue
		}
		v.Indexes++
		indexBytes += r.Bytes
		if v.Indexes > 256 || indexBytes > 4<<20 {
			v.Notice = errSpatialLimit.Error()
			return v, nil, pins
		}
		need := spatialNeed{spatialPath("indexes", r.SHA256), r.Bytes, r.SHA256}
		pins[need.path] = true
		indexKey := fmt.Sprintf("i:%s:%v:%d", r.SHA256, r.Cell, manifest.RootLY)
		var blocks []spatialBlock
		if cached, ok := m.cache.get(indexKey); ok {
			blocks = cached.([]spatialBlock)
		} else {
			raw, e := m.asset(need)
			if e != nil {
				needs = append(needs, need)
				continue
			}
			blocks, e = parseSpatialIndex(raw, r, manifest.RootLY)
			if e != nil {
				v.Notice = e.Error()
				return v, nil, pins
			}
			m.cache.markVerified(r.SHA256)
			m.cache.put(indexKey, blocks, int64(len(blocks))*96+64)
		}
		for _, d := range blocks {
			if overlap(lo, hi, d.Lo, d.Hi) {
				descriptors = append(descriptors, d)
				v.Blocks++
				v.Decoded += int64(8 + 12*d.Count)
				if v.Blocks > 128 || v.Decoded > 16<<20 {
					v.Notice = errSpatialLimit.Error()
					return v, nil, pins
				}
			}
		}
	}
	for _, d := range descriptors {
		need := spatialNeed{spatialPath("blocks", d.SHA256), d.Bytes, d.SHA256}
		pins[need.path] = true
		blockKey := fmt.Sprintf("b:%s:%v:%v:%d", d.SHA256, d.Lo, d.Hi, d.Count)
		cached, decoded := m.cache.get(blockKey)
		if !decode {
			// Planning: presence only; never materialize coordinates.
			if decoded || m.cache.verified[d.SHA256] {
				continue
			}
			if _, e := m.asset(need); e != nil {
				needs = append(needs, need)
				continue
			}
			m.cache.markVerified(d.SHA256)
			perfstats.Add("spatial.plan.verified", 1)
			continue
		}
		var points [][3]int32
		if decoded {
			points = cached.([][3]int32)
		} else {
			raw, e := m.asset(need)
			if e != nil {
				needs = append(needs, need)
				continue
			}
			points, e = decodeESP(raw, d)
			if e != nil {
				v.Notice = e.Error()
				v.Points = [][3]float64{}
				return v, nil, pins
			}
			m.cache.markVerified(d.SHA256)
			m.cache.put(blockKey, points, int64(len(points))*12+64)
			perfstats.Add("spatial.decode", 1)
		}
		for _, p := range points {
			inside := true
			for i := range 3 {
				if p[i] < lo[i] || p[i] >= hi[i] {
					inside = false
					break
				}
			}
			if !inside {
				continue
			}
			if manifest.RadiusLY > 0 {
				var sq int64
				for _, x := range p {
					sq += int64(x) * int64(x)
				}
				if sq > (manifest.RadiusLY*32)*(manifest.RadiusLY*32) {
					continue
				}
			}
			v.Points = append(v.Points, [3]float64{float64(p[0]) / 32, float64(p[1]) / 32, float64(p[2]) / 32})
			if len(v.Points) > 80000 {
				v.Points = [][3]float64{}
				v.Notice = errSpatialLimit.Error()
				return v, nil, pins
			}
		}
	}
	v.Missing = len(needs)
	if len(needs) > 0 {
		v.Points = [][3]float64{}
		v.Notice = "Spatial assets missing or corrupt; density is a full line-of-sight fallback, not an empty volume"
		return v, needs, pins
	}
	v.Complete = true
	v.Notice = "Verified reported positions in selected XYZ volume; names and spectral classes not supplied"
	return v, nil, pins
}
func (m *SpatialManager) View(q SpatialQuery, owner bool) (SpatialView, error) {
	perfstats.Add("spatial.view", 1)
	if !validSpatialQuery(q) {
		return SpatialView{}, errors.New("Invalid spatial volume or prefetch context")
	}
	m.ioMu.Lock()
	key := spatialQueryKey(q, false)
	var v SpatialView
	var needs []spatialNeed
	var pins map[string]bool
	if key == m.cachedKey && m.cached.Complete {
		v = m.cached
	} else {
		v, needs, pins = m.query(q)
		v.ViewToken = spatialViewToken(v, key)
		if v.Complete {
			m.cachedKey = key
			m.cached = v
		}
	}
	if owner && pins != nil {
		m.pins = pins
	}
	m.ioMu.Unlock()
	if owner {
		m.schedule(q, needs)
	}
	m.mu.Lock()
	v.Pending = m.pending
	v.PrefetchPending = m.pending
	m.mu.Unlock()
	v.ViewportReady = v.Complete
	if v.Complete && q.Since != "" && q.Since == v.ViewToken {
		perfstats.Add("spatial.unchanged", 1)
		return SpatialView{Points: nil, Complete: true, Pending: v.Pending, Notice: v.Notice, Source: v.Source, Release: v.Release, Indexes: v.Indexes, Blocks: v.Blocks, Decoded: v.Decoded, ViewToken: v.ViewToken, Unchanged: true, ViewportReady: true, PrefetchPending: v.PrefetchPending}, nil
	}
	return v, nil
}

// spatialViewToken identifies a view's content: release, query, completeness
// and coordinates. A different release, query or result never matches.
func spatialViewToken(v SpatialView, key string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%t\x00%d\x00%d\x00", v.Release, v.Source, key, v.Complete, v.Missing, len(v.Points))
	var buf [24]byte
	for _, p := range v.Points {
		for i := range 3 {
			binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(p[i]))
		}
		h.Write(buf[:])
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

// Only owner views initiate or reprioritize work. Viewer requests never mutate
// consent, the active queue or pinned owner working set.
func (m *SpatialManager) schedule(q SpatialQuery, needs []spatialNeed) {
	key := spatialQueryKey(q, true)
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.allowed() || m.current == nil || m.busy {
		return
	}
	if key == m.queueKey {
		return
	}
	if m.queueCancel != nil {
		m.queueCancel()
	}
	ctx, cancel := context.WithCancel(m.network)
	m.queueCancel = cancel
	m.queueKey = key
	m.pending = true
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer func() {
			m.mu.Lock()
			if m.queueKey == key {
				m.pending = false
			}
			m.mu.Unlock()
		}()
		// Viewport first, one expanded adjacent window, next five known hops, target.
		windows := []SpatialQuery{q}
		adj := q
		adj.Route = nil
		adj.Target = nil
		for i := range 3 {
			s := (q.High[i] - q.Low[i]) * .20
			adj.Low[i] = math.Max(-1e6, q.Low[i]-s)
			adj.High[i] = math.Min(1e6, q.High[i]+s)
		}
		windows = append(windows, adj)
		centers := append([][3]float64(nil), q.Route...)
		if q.Target != nil {
			centers = append(centers, *q.Target)
		}
		for _, c := range centers {
			w := SpatialQuery{}
			for i := range 3 {
				half := math.Min(100, (q.High[i]-q.Low[i])/2)
				w.Low[i] = math.Max(-1e6, c[i]-half)
				w.High[i] = math.Min(1e6, c[i]+half)
			}
			windows = append(windows, w)
		}
		// Whole incremental plan is bounded, not a long-route/worldwide downloader.
		remaining := 256
		encoded := int64(32 << 20)
		for _, w := range windows {
			if ctx.Err() != nil {
				return
			}
			for pass := 0; pass < 2; pass++ {
				m.ioMu.Lock()
				list, _ := m.planMissing(w)
				m.ioMu.Unlock()
				if len(list) == 0 {
					break
				}
				if len(list) > remaining {
					return
				}
				remaining -= len(list)
				var batch []spatialNeed
				for _, n := range list {
					encoded -= n.bytes
					if encoded < 0 {
						return
					}
					m.mu.Lock()
					blocked := m.now().Before(m.backoff[n.path])
					m.mu.Unlock()
					if !blocked {
						batch = append(batch, n)
					}
				}
				if len(batch) == 0 {
					break
				}
				jobs := make(chan spatialNeed)
				var wg sync.WaitGroup
				for range 2 {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for n := range jobs {
							if ctx.Err() != nil {
								return
							}
							m.downloadAsset(ctx, n)
						}
					}()
				}
				for _, n := range batch {
					select {
					case jobs <- n:
					case <-ctx.Done():
						close(jobs)
						wg.Wait()
						return
					}
				}
				close(jobs)
				wg.Wait()
			}
		}
	}()
}
func (m *SpatialManager) downloadAsset(ctx context.Context, n spatialNeed) {
	m.ioMu.Lock()
	_, e := m.asset(n)
	m.ioMu.Unlock()
	if e == nil {
		return
	}
	b, _, _, e := m.fetch(ctx, n.path, n.bytes, "")
	if e == nil && (int64(len(b)) != n.bytes || digest(b) != n.hash) {
		e = errSpatial
	}
	if e == nil && ctx.Err() != nil {
		return
	}
	if e == nil {
		m.ioMu.Lock()
		e = writeFile(m.root, n.path, b)
		if e == nil {
			m.evict()
			m.revision.Add(1)
		}
		m.ioMu.Unlock()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if e != nil && ctx.Err() == nil {
		if len(m.backoff) >= 512 {
			var oldest string
			var at time.Time
			for p, t := range m.backoff {
				if oldest == "" || t.Before(at) {
					oldest = p
					at = t
				}
			}
			delete(m.backoff, oldest)
			delete(m.failures, oldest)
		}
		m.failures[n.path]++
		seconds := time.Duration(1 << min(m.failures[n.path], 8))
		m.backoff[n.path] = m.now().Add(seconds * time.Second)
		m.lastError = "Spatial download failed; cached data retained, retry after backoff"
	} else if e == nil {
		delete(m.backoff, n.path)
		delete(m.failures, n.path)
		if len(m.backoff) == 0 && m.lastError == "Spatial download failed; cached data retained, retry after backoff" {
			m.lastError = ""
		}
	}
}
func (m *SpatialManager) evict() {
	type entry struct {
		path string
		size int64
		time time.Time
	}
	var all []entry
	var total int64
	for _, g := range []string{"indexes", "blocks"} {
		es, _ := os.ReadDir(filepath.Join(m.root, "assets", g))
		for _, e := range es {
			st, err := e.Info()
			if err != nil || !st.Mode().IsRegular() {
				continue
			}
			p := "assets/" + g + "/" + e.Name()
			all = append(all, entry{p, st.Size(), st.ModTime()})
			total += st.Size()
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].time.Before(all[j].time) })
	entries := len(all)
	for _, e := range all {
		if total <= spatialCacheBudget && entries <= 4096 {
			break
		}
		if !m.pins[e.path] {
			if os.Remove(filepath.Join(m.root, filepath.FromSlash(e.path))) == nil {
				total -= e.size
				entries--
				if m.cache != nil {
					m.cache.forgetFiles() // a verified file is gone
				}
			}
		}
	}
}
func (m *SpatialManager) bubbleManifest() *spatialManifest {
	if m.bubble != nil {
		return m.bubble.manifest
	}
	return nil
}
