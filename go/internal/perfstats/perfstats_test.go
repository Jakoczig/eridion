package perfstats

import (
	"sync"
	"testing"
	"time"
)

func TestCountersTimingsAndMutex(t *testing.T) {
	Reset()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); Add("x", 2) }()
	}
	wg.Wait()
	for _, d := range []time.Duration{time.Millisecond, 2 * time.Millisecond, 100 * time.Millisecond} {
		Observe("t", d)
	}
	var m Mutex
	m.Named("lock")
	m.Lock()
	time.Sleep(time.Millisecond)
	m.Unlock()
	s := Take()
	if s.Counters["x"] != 16 {
		t.Fatal(s.Counters)
	}
	tm := s.Timings["t"]
	if tm.Count != 3 || tm.MaxMs < 99 || tm.P50Ms < 1 || tm.P50Ms > 5 || tm.P95Ms < 64 {
		t.Fatalf("%+v", tm)
	}
	if s.Timings["lock.hold"].Count != 1 || s.Timings["lock.wait"].Count != 1 {
		t.Fatal(s.Timings)
	}
	Reset()
	if len(Take().Counters) != 0 {
		t.Fatal("reset")
	}
}
