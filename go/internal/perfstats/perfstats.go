// Package perfstats holds cheap in-process performance counters used by the
// G3K native performance measurements (docs/testing/g3k-native-performance.md).
// Counters are atomic and always on; they hold no gameplay data, only counts
// and durations, and are read only by test and measurement harnesses.
package perfstats

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type counter struct{ n atomic.Int64 }

// timing accumulates durations with a coarse logarithmic histogram (bucket i
// covers [2^i, 2^(i+1)) microseconds) for p50/p95 estimates.
type timing struct {
	count, sum, max atomic.Int64 // nanoseconds
	buckets         [32]atomic.Int64
}

var counters, timings sync.Map

// Add increments the named counter.
func Add(name string, n int64) {
	c, ok := counters.Load(name)
	if !ok {
		c, _ = counters.LoadOrStore(name, &counter{})
	}
	c.(*counter).n.Add(n)
}

// Observe records one duration under name.
func Observe(name string, d time.Duration) {
	v, ok := timings.Load(name)
	if !ok {
		v, _ = timings.LoadOrStore(name, &timing{})
	}
	t := v.(*timing)
	ns := d.Nanoseconds()
	t.count.Add(1)
	t.sum.Add(ns)
	for {
		m := t.max.Load()
		if ns <= m || t.max.CompareAndSwap(m, ns) {
			break
		}
	}
	us, b := ns/1000, 0
	for us > 1 && b < 31 {
		us >>= 1
		b++
	}
	t.buckets[b].Add(1)
}

// Timing is a snapshot of one duration series (milliseconds).
type Timing struct {
	Count  int64   `json:"count"`
	SumMs  float64 `json:"sumMs"`
	MaxMs  float64 `json:"maxMs"`
	P50Ms  float64 `json:"p50Ms"`
	P95Ms  float64 `json:"p95Ms"`
	MeanMs float64 `json:"meanMs"`
}

// Snapshot is every counter and timing at one instant.
type Snapshot struct {
	Counters map[string]int64  `json:"counters"`
	Timings  map[string]Timing `json:"timings"`
}

// Take returns the current values.
func Take() Snapshot {
	s := Snapshot{Counters: map[string]int64{}, Timings: map[string]Timing{}}
	counters.Range(func(k, v any) bool { s.Counters[k.(string)] = v.(*counter).n.Load(); return true })
	timings.Range(func(k, v any) bool {
		t := v.(*timing)
		out := Timing{Count: t.count.Load(), SumMs: float64(t.sum.Load()) / 1e6, MaxMs: float64(t.max.Load()) / 1e6}
		if out.Count > 0 {
			out.MeanMs = out.SumMs / float64(out.Count)
			out.P50Ms, out.P95Ms = quantile(t, out.Count, 0.5), quantile(t, out.Count, 0.95)
		}
		s.Timings[k.(string)] = out
		return true
	})
	return s
}

// quantile returns the upper bound of the bucket holding quantile q.
func quantile(t *timing, count int64, q float64) float64 {
	want := int64(float64(count)*q + 0.5)
	if want < 1 {
		want = 1
	}
	seen := int64(0)
	for i := range t.buckets {
		seen += t.buckets[i].Load()
		if seen >= want {
			return float64(int64(1)<<(i+1)) / 1000
		}
	}
	return float64(t.max.Load()) / 1e6
}

// Reset clears all counters and timings.
func Reset() {
	counters.Range(func(k, _ any) bool { counters.Delete(k); return true })
	timings.Range(func(k, _ any) bool { timings.Delete(k); return true })
}

// Names lists counter names in order (diagnostics).
func Names() []string {
	out := []string{}
	counters.Range(func(k, _ any) bool { out = append(out, k.(string)); return true })
	sort.Strings(out)
	return out
}

// Mutex is a sync.Mutex that records how long callers waited for it and how
// long it was held, under "<name>.wait" and "<name>.hold".
type Mutex struct {
	mu     sync.Mutex
	name   string
	held   time.Time
	inited bool
}

// Named sets the series name (call once before use).
func (m *Mutex) Named(name string) { m.name = name; m.inited = true }

// Lock acquires the mutex.
func (m *Mutex) Lock() {
	if !m.inited {
		m.mu.Lock()
		return
	}
	start := time.Now()
	m.mu.Lock()
	now := time.Now()
	Observe(m.name+".wait", now.Sub(start))
	m.held = now
}

// Unlock releases the mutex.
func (m *Mutex) Unlock() {
	if m.inited {
		Observe(m.name+".hold", time.Since(m.held))
	}
	m.mu.Unlock()
}
