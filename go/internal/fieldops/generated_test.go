package fieldops

import (
	"bytes"
	"os"
	"testing"

	"github.com/Jakoczig/eridion/go/internal/fieldops/bioconvert"
)

// The bundled rules are exactly what the converter produces from the
// vendored, pinned upstream files, and the support manifest matches them.
func TestBioRulesGeneratedFromPinnedSources(t *testing.T) {
	p, err := bioconvert.Convert("upstream")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := bioconvert.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, bioRules) {
		t.Fatal("bio-rules.json differs from the pinned sources; run go run ./cmd/bioscan-rules and review")
	}
	m, err := MarshalManifest(SupportManifest())
	if err != nil {
		t.Fatal(err)
	}
	cur, err := os.ReadFile("predicate-support.json")
	if err != nil || !bytes.Equal(cur, m) {
		t.Fatal("predicate-support.json is stale; run go run ./cmd/bioscan-rules -manifest")
	}
	for k := range predicateKeysInRules() {
		if _, ok := predicateSupport[k]; !ok {
			t.Errorf("rule key %q has no declared support level", k)
		}
	}
}

func predicateKeysInRules() map[string]bool {
	out := map[string]bool{}
	for _, s := range pack.Species {
		for _, r := range s.Rulesets {
			for k := range r {
				out[k] = true
			}
		}
	}
	return out
}
