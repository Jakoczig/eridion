package fieldops

import (
	"encoding/json"
	"sort"
)

// Predicate support levels in the native evaluator.
const (
	// SupportNative: evaluated natively; unknown only while the named journal
	// evidence is missing.
	SupportNative = "native"
	// SupportPartial: only part of the upstream test is possible natively;
	// the rest stays unknown.
	SupportPartial = "partial"
	// SupportUnsupported: the capability's data is not bundled; always unknown.
	SupportUnsupported = "unsupported"
	// SupportIgnoredUpstream: upstream has no handler for the key, so it never
	// restricts; mirrored as a pass.
	SupportIgnoredUpstream = "ignored-upstream"
)

// PredicateSupport describes one rule key.
type PredicateSupport struct {
	Key      string `json:"key"`
	Support  string `json:"support"`
	Evidence string `json:"evidence"`
	Note     string `json:"note,omitempty"`
}

var predicateSupport = map[string]PredicateSupport{
	"atmosphere":           {Support: SupportNative, Evidence: "Scan.AtmosphereType"},
	"atmosphere_component": {Support: SupportNative, Evidence: "Scan.AtmosphereComposition"},
	"min_gravity":          {Support: SupportNative, Evidence: "Scan.SurfaceGravity", Note: "m/s² / 9.797759, inclusive"},
	"max_gravity":          {Support: SupportNative, Evidence: "Scan.SurfaceGravity", Note: "m/s² / 9.797759, inclusive"},
	"min_temperature":      {Support: SupportNative, Evidence: "Scan.SurfaceTemperature", Note: "K, inclusive; zero skipped as upstream"},
	"max_temperature":      {Support: SupportNative, Evidence: "Scan.SurfaceTemperature", Note: "K, inclusive; zero skipped as upstream"},
	"min_pressure":         {Support: SupportNative, Evidence: "Scan.SurfacePressure", Note: "Pa / 101231.65625, inclusive; zero skipped as upstream"},
	"max_pressure":         {Support: SupportNative, Evidence: "Scan.SurfacePressure", Note: "Pa / 101231.65625, exclusive (>= excludes); zero skipped as upstream"},
	"max_orbital_period":   {Support: SupportNative, Evidence: "Scan.OrbitalPeriod", Note: "seconds, exclusive"},
	"volcanism":            {Support: SupportNative, Evidence: "Scan.Volcanism", Note: "substring, '=' exact, 'Any', 'None', '!' forms"},
	"body_type":            {Support: SupportNative, Evidence: "Scan.PlanetClass"},
	"distance":             {Support: SupportNative, Evidence: "Scan.DistanceFromArrivalLS"},
	"system":               {Support: SupportNative, Evidence: "system name"},
	"star":                 {Support: SupportNative, Evidence: "star Scans; FSSAllBodiesFound for exclusion"},
	"main_star":            {Support: SupportNative, Evidence: "primary star Scan"},
	"parent_star":          {Support: SupportNative, Evidence: "primary and parent star Scans"},
	"bodies":               {Support: SupportNative, Evidence: "body Scans; FSSAllBodiesFound for exclusion"},
	"guardian":             {Support: SupportNative, Evidence: "system StarPos", Note: "bundled guardian zone table"},
	"tuber":                {Support: SupportNative, Evidence: "system StarPos", Note: "bundled tuber zone table"},
	"nebula":               {Support: SupportPartial, Evidence: "system name", Note: "nebula sector names only; reference-star distances not bundled"},
	"regions":              {Support: SupportUnsupported, Evidence: "galactic region", Note: "region map not bundled"},
	"region":               {Support: SupportIgnoredUpstream, Evidence: "-", Note: "upstream has no handler for the singular key"},
}

// RulesetSupport summarises one alternative.
type RulesetSupport struct {
	Predicates []string `json:"predicates"`
	// Support is "full" when every predicate is native (or ignored upstream),
	// otherwise the weakest level present.
	Support string `json:"support"`
}

// SpeciesSupport is one manifest row.
type SpeciesSupport struct {
	Genus    string           `json:"genus"`
	Symbol   string           `json:"symbol"`
	Name     string           `json:"name"`
	Source   string           `json:"source"`
	Rulesets []RulesetSupport `json:"rulesets"`
	// FullySupportedAlternative: at least one alternative can be decided
	// natively once its journal evidence exists.
	FullySupportedAlternative bool `json:"fullySupportedAlternative"`
	// Variant: the colour dependency ("parent-star", "materials" or "none").
	Variant string `json:"variant"`
}

// Manifest is the machine-readable predicate-support manifest.
type Manifest struct {
	Version     string             `json:"version"`
	Predicates  []PredicateSupport `json:"predicates"`
	Species     []SpeciesSupport   `json:"species"`
	Total       int                `json:"total"`
	WithoutFull int                `json:"withoutFullySupportedAlternative"`
}

// SupportManifest derives the manifest from the bundled rule data.
func SupportManifest() Manifest {
	m := Manifest{Version: pack.Version}
	for _, k := range sortedPredicateKeys() {
		p := predicateSupport[k]
		p.Key = k
		m.Predicates = append(m.Predicates, p)
	}
	rank := map[string]int{SupportNative: 0, SupportIgnoredUpstream: 0, SupportPartial: 1, SupportUnsupported: 2}
	for _, s := range pack.Species {
		row := SpeciesSupport{Genus: s.Genus, Symbol: s.Symbol, Name: s.Name, Source: s.File, Rulesets: []RulesetSupport{}, Variant: "none"}
		for _, r := range s.Rulesets {
			rs := RulesetSupport{Predicates: sortedKeys(r), Support: "full"}
			worst := 0
			for _, k := range rs.Predicates {
				p, ok := predicateSupport[k]
				level := SupportUnsupported
				if ok {
					level = p.Support
				}
				if rank[level] > worst {
					worst = rank[level]
					rs.Support = level
				}
			}
			if rs.Support == "full" {
				row.FullySupportedAlternative = true
			}
			row.Rulesets = append(row.Rulesets, rs)
		}
		g := pack.genus[s.Genus]
		if spec, ok := g.Colors["species"].(map[string]any); ok {
			entry, _ := spec[s.Symbol].(map[string]any)
			if _, ok := entry["star"]; ok {
				row.Variant = "parent-star"
			} else if _, ok := entry["element"]; ok {
				row.Variant = "materials"
			}
		} else if _, ok := g.Colors["star"]; ok {
			row.Variant = "parent-star"
		}
		if !row.FullySupportedAlternative {
			m.WithoutFull++
		}
		m.Species = append(m.Species, row)
	}
	m.Total = len(m.Species)
	return m
}

// MarshalManifest renders the manifest deterministically.
func MarshalManifest(m Manifest) ([]byte, error) {
	b, err := json.MarshalIndent(m, "", " ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func sortedPredicateKeys() []string {
	out := make([]string, 0, len(predicateSupport))
	for k := range predicateSupport {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
