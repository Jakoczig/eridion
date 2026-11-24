package gamestate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/Jakoczig/eridion/go/internal/fieldops"
)

// bioInference memoizes candidate inference (Task 18 / PERF). The key binds
// commander, system, body, rule revision and a digest of the exact evidence,
// so a cached result can never answer for different evidence; concurrent
// identical requests share one computation (single flight). Bounded.
type bioInference struct {
	mu       sync.Mutex
	entries  map[string]*bioEntry
	order    []string
	computed int // inference runs, for tests
}

type bioEntry struct {
	done       chan struct{}
	candidates []fieldops.Candidate
}

const bioInferenceEntries = 64

func bioKey(c bioContext, env fieldops.Environment, genera []string) string {
	raw, _ := json.Marshal(struct {
		Env    fieldops.Environment
		Genera []string
	}{env, genera})
	sum := sha256.Sum256(raw)
	return c.fid + "\x00" + c.system + "\x00" + c.body + "\x00" + fieldops.BioRuleVersion() + "\x00" + hex.EncodeToString(sum[:])
}

func (s *Service) bioCandidates(c bioContext, env fieldops.Environment, genera []string) []fieldops.Candidate {
	key := bioKey(c, env, genera)
	b := &s.bioCache
	b.mu.Lock()
	if b.entries == nil {
		b.entries = map[string]*bioEntry{}
	}
	if e, ok := b.entries[key]; ok {
		b.mu.Unlock()
		<-e.done
		return cloneCandidates(e.candidates)
	}
	e := &bioEntry{done: make(chan struct{})}
	b.entries[key] = e
	b.order = append(b.order, key)
	if len(b.order) > bioInferenceEntries {
		old := b.order[0]
		b.order = b.order[1:]
		delete(b.entries, old)
	}
	b.computed++
	b.mu.Unlock()
	e.candidates = fieldops.InferEnvironment(env, genera)
	close(e.done)
	return cloneCandidates(e.candidates)
}

func cloneCandidates(in []fieldops.Candidate) []fieldops.Candidate {
	out := make([]fieldops.Candidate, len(in))
	for i, c := range in {
		c.Evidence = append([]string(nil), c.Evidence...)
		c.Unknown = append([]string(nil), c.Unknown...)
		c.Variants = append([]string(nil), c.Variants...)
		out[i] = c
	}
	return out
}
