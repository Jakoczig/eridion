package fieldops

import (
	"math"
	"sort"
	"strings"
)

// Native evaluation of the pinned EDMC-BioScan criteria (Task 6 / Task 23,
// G3J). Every predicate is three-valued: it passes, fails, or stays unknown
// when the journal evidence it needs is missing or the capability is not
// bundled. A ruleset fails when any predicate fails, passes when all pass,
// and is unknown otherwise; a species is possible when any alternative
// passes, excluded when all fail, and unknown otherwise. Insufficient
// evidence never becomes a positive prediction.

// Inference outcomes.
const (
	Possible = "possible"
	Excluded = "excluded"
	Unknown  = "unknown"
)

// Upstream conversion constants (load.py, BioScan fc6b85c). The thresholds
// were derived in these units, so the native comparison uses them too; see
// docs/testing/bioscan-conformance.md.
const (
	GravityDivisor  = 9.797759      // m/s² per threshold unit
	PressureDivisor = 101231.656250 // Pa per threshold unit
)

// StarEvidence is one scanned star of the current system.
type StarEvidence struct {
	BodyName   string
	Type       string
	Luminosity string
	DistanceLS *float64
}

// PlanetEvidence is one scanned non-star body of the current system.
type PlanetEvidence struct {
	BodyName string
	Class    string
}

// Environment is the journal evidence available for one body. Body holds the
// body's latest Scan fields as decoded from the journal.
type Environment struct {
	Body       map[string]any
	SystemName string
	// Coords are the system's galactic coordinates (StarPos), if witnessed.
	Coords  *[3]float64
	Stars   []StarEvidence
	Planets []PlanetEvidence
	// AllBodiesFound: FSSAllBodiesFound was witnessed, so absence of a star
	// or body type among the scans is evidence, not a gap.
	AllBodiesFound bool
}

type outcome int

const (
	pass outcome = iota
	unknown
	fail
)

func combine(a, b outcome) outcome {
	if a > b {
		return a
	}
	return b
}

// evaluator caches per-environment derived facts.
type evaluator struct {
	env         Environment
	body        map[string]any
	parentNames []string
	stars       map[string]StarEvidence
	main        *StarEvidence
}

func newEvaluator(env Environment) *evaluator {
	e := &evaluator{env: env, body: env.Body, stars: map[string]StarEvidence{}}
	if e.body == nil {
		e.body = map[string]any{}
	}
	for i := range env.Stars {
		s := env.Stars[i]
		e.stars[e.starKey(s.BodyName)] = s
		if s.DistanceLS != nil && *s.DistanceLS == 0 {
			m := s
			e.main = &m
		}
	}
	e.parentNames = e.parents()
	return e
}

// starKey mirrors ExploData: a star's key is its name without the system
// prefix; the system's sole primary is keyed by the system name.
func (e *evaluator) starKey(name string) string {
	sys := e.env.SystemName
	if sys != "" && strings.HasPrefix(name, sys+" ") {
		return strings.TrimPrefix(name, sys+" ")
	}
	return name
}

// parents mirrors ExploData's parent-star derivation from the body's short
// name: "AB 1 a" → A and B; "1 a" → the system's primary.
func (e *evaluator) parents() []string {
	name := text(e.body["BodyName"])
	sys := e.env.SystemName
	if name == "" || sys == "" || !strings.HasPrefix(name, sys+" ") {
		return nil
	}
	short := strings.TrimPrefix(name, sys+" ")
	head, _, found := strings.Cut(short, " ")
	if found && head != "" && strings.ToUpper(head) == head && strings.Trim(head, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") == "" {
		out := []string{}
		for _, r := range head {
			out = append(out, string(r))
		}
		return out
	}
	return []string{sys}
}

func text(v any) string { s, _ := v.(string); return s }

func (e *evaluator) float(field string) (float64, bool) {
	v, ok := e.body[field]
	if !ok {
		return 0, false
	}
	return floatValue(v)
}

// atmosphere returns the AtmosphereType, treating an explicitly empty
// Atmosphere description as no atmosphere.
func (e *evaluator) atmosphere() (string, bool) {
	if v, ok := e.body["AtmosphereType"].(string); ok {
		return v, true
	}
	if v, ok := e.body["Atmosphere"].(string); ok && v == "" {
		return "None", true
	}
	return "", false
}

func (e *evaluator) volcanism() (string, bool) {
	v, ok := e.body["Volcanism"].(string)
	if !ok {
		return "", false
	}
	if strings.EqualFold(v, "No volcanism") {
		v = "" // older journals spell "no volcanism" out
	}
	return v, true
}

// starCheck mirrors upstream body_data.util.star_check.
func starCheck(query, kind string) bool {
	switch query {
	case "A":
		return kind == "A" || kind == "A_BlueWhiteSuperGiant"
	case "B":
		return kind == "B" || kind == "B_BlueWhiteSuperGiant"
	case "F":
		return kind == "F" || kind == "F_WhiteSuperGiant"
	case "G":
		return kind == "G" || kind == "G_WhiteSuperGiant"
	case "K":
		return kind == "K" || kind == "K_OrangeGiant"
	case "M":
		return kind == "M" || kind == "M_RedGiant" || kind == "M_RedSuperGiant"
	case "D", "C", "W":
		return strings.HasPrefix(kind, query)
	}
	return kind == query
}

// starSpecMatches evaluates one entry of a star list: a type string, or a
// {"tuple": [type, luminosity]} pair (luminosity may carry an a/b/ab/z
// suffix). Upstream never matches a plain list entry; neither does this.
func starSpecMatches(spec any, s StarEvidence) bool {
	switch x := spec.(type) {
	case string:
		return starCheck(x, s.Type)
	case map[string]any:
		t, _ := x["tuple"].([]any)
		if len(t) != 2 {
			return false
		}
		kind, lum := text(t[0]), text(t[1])
		if !starCheck(kind, s.Type) {
			return false
		}
		for _, flag := range []string{"", "a", "b", "ab", "z"} {
			if lum+flag == s.Luminosity {
				return true
			}
		}
	}
	return false
}

func distance(a, b [3]float64) float64 {
	return math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
}

// predicate evaluates one rule key. reason names the missing evidence or
// unsupported capability when the result is unknown.
func (e *evaluator) predicate(key string, want any) (outcome, string) {
	switch key {
	case "atmosphere":
		atm, ok := e.atmosphere()
		if !ok {
			return unknown, "atmosphere not scanned"
		}
		if want == "Any" {
			if atm == "" || atm == "None" {
				return fail, ""
			}
			return pass, ""
		}
		if containsString(want, atm) {
			return pass, ""
		}
		return fail, ""
	case "atmosphere_component":
		gases, ok := want.(map[string]any)
		if !ok {
			return unknown, "unsupported atmosphere component rule"
		}
		comp, hasComp := e.body["AtmosphereComposition"].([]any)
		atm, atmOK := e.atmosphere()
		if !hasComp && !(atmOK && (atm == "" || atm == "None")) {
			return unknown, "atmosphere composition not scanned"
		}
		for gas, min := range gases {
			need, _ := floatValue(min)
			got := 0.0
			for _, c := range comp {
				m, _ := c.(map[string]any)
				if text(m["Name"]) == gas {
					got, _ = floatValue(m["Percent"])
				}
			}
			if got < need {
				return fail, ""
			}
		}
		return pass, ""
	case "min_gravity", "max_gravity":
		g, ok := e.float("SurfaceGravity")
		if !ok {
			return unknown, "gravity not scanned"
		}
		return bound(key, g/GravityDivisor, want, false), ""
	case "min_temperature", "max_temperature":
		t, ok := e.float("SurfaceTemperature")
		if !ok {
			return unknown, "temperature not scanned"
		}
		if t == 0 {
			return pass, "" // upstream skips a zero temperature
		}
		return bound(key, t, want, false), ""
	case "min_pressure", "max_pressure":
		p, ok := e.float("SurfacePressure")
		if !ok {
			return unknown, "pressure not scanned"
		}
		if p == 0 {
			return pass, "" // upstream skips a zero pressure
		}
		return bound(key, p/PressureDivisor, want, true), ""
	case "max_orbital_period":
		p, ok := e.float("OrbitalPeriod")
		if !ok {
			return unknown, "orbital period not scanned"
		}
		n, _ := floatValue(want)
		if p >= n {
			return fail, ""
		}
		return pass, ""
	case "volcanism":
		v, ok := e.volcanism()
		if !ok {
			return unknown, "volcanism not scanned"
		}
		switch w := want.(type) {
		case []any:
			for _, x := range w {
				s := text(x)
				if strings.HasPrefix(s, "=") {
					if v == s[1:] {
						return pass, ""
					}
				} else if strings.Contains(v, s) {
					return pass, ""
				}
			}
			return fail, ""
		case string:
			switch {
			case w == "Any":
				if v == "" {
					return fail, ""
				}
			case w == "None":
				if v != "" {
					return fail, ""
				}
			case strings.HasPrefix(w, "!"):
				if v == "" || strings.Contains(v, w[1:]) {
					return fail, ""
				}
			}
			return pass, ""
		}
		return unknown, "unsupported volcanism rule"
	case "body_type":
		c, ok := e.body["PlanetClass"].(string)
		if !ok {
			return unknown, "body type not scanned"
		}
		if containsString(want, c) {
			return pass, ""
		}
		return fail, ""
	case "distance":
		d, ok := e.float("DistanceFromArrivalLS")
		if !ok {
			return unknown, "arrival distance not scanned"
		}
		n, _ := floatValue(want)
		if d < n {
			return fail, ""
		}
		return pass, ""
	case "system":
		if e.env.SystemName == "" {
			return unknown, "system name unknown"
		}
		if e.env.SystemName != text(want) {
			return fail, ""
		}
		return pass, ""
	case "region":
		// Upstream has no handler for the singular key (Brain Tree), so
		// it never restricts; mirrored, see the conformance document.
		return pass, ""
	case "regions":
		return unknown, "galactic region map not bundled"
	case "guardian":
		if b, ok := want.(bool); ok && !b {
			return pass, ""
		}
		if e.env.Coords == nil {
			return unknown, "system coordinates unknown"
		}
		for _, z := range pack.GuardianZones {
			if distance(*e.env.Coords, z.Center) < z.MaxLY {
				return pass, ""
			}
		}
		return fail, ""
	case "tuber":
		if e.env.Coords == nil {
			return unknown, "system coordinates unknown"
		}
		for _, z := range pack.TuberZones {
			if want == "Any" || containsString(want, z.Name) {
				d := distance(*e.env.Coords, z.Center)
				if z.MinLY <= d && d <= z.MaxLY {
					return pass, ""
				}
			}
		}
		return fail, ""
	case "nebula":
		for _, sector := range pack.NebulaSectors {
			if e.env.SystemName != "" && strings.HasPrefix(e.env.SystemName, sector) {
				return pass, ""
			}
		}
		return unknown, "nebula reference-star distances not bundled"
	case "bodies":
		for _, p := range e.env.Planets {
			if containsString(want, p.Class) {
				return pass, ""
			}
		}
		if e.env.AllBodiesFound {
			return fail, ""
		}
		return unknown, "system bodies not fully scanned"
	case "star":
		specs, list := want.([]any)
		if !list {
			specs = []any{want}
		}
		for _, s := range e.env.Stars {
			for _, spec := range specs {
				if starSpecMatches(spec, s) {
					return pass, ""
				}
			}
		}
		if e.env.AllBodiesFound {
			return fail, ""
		}
		return unknown, "system stars not fully scanned"
	case "main_star":
		if e.main == nil {
			return unknown, "primary star not scanned"
		}
		specs, list := want.([]any)
		if !list {
			specs = []any{want}
		}
		for _, spec := range specs {
			if starSpecMatches(spec, *e.main) {
				return pass, ""
			}
		}
		return fail, ""
	case "parent_star":
		specs, _ := want.([]any)
		matches := func(s StarEvidence) bool {
			for _, spec := range specs {
				if t, ok := spec.(string); ok && starCheck(t, s.Type) {
					return true
				}
			}
			return false
		}
		if e.main != nil && matches(*e.main) {
			return pass, ""
		}
		complete := e.main != nil && len(e.parentNames) > 0
		for _, name := range e.parentNames {
			s, ok := e.stars[name]
			if !ok {
				complete = false
				continue
			}
			if matches(s) {
				return pass, ""
			}
		}
		if complete {
			return fail, ""
		}
		return unknown, "parent star not scanned"
	}
	return unknown, "unsupported predicate " + key
}

func bound(key string, got float64, want any, exclusiveMax bool) outcome {
	n, ok := floatValue(want)
	if !ok {
		return unknown
	}
	if strings.HasPrefix(key, "min_") && got < n {
		return fail
	}
	if strings.HasPrefix(key, "max_") && (got > n || exclusiveMax && got >= n) {
		return fail
	}
	return pass
}

func containsString(list any, v string) bool {
	a, _ := list.([]any)
	for _, x := range a {
		if text(x) == v {
			return true
		}
	}
	return false
}

// colours mirrors upstream's colour derivation: the first scanned parent
// star (in parent order) whose type has a colour supplies it, then the
// primary when it is not a parent. complete is false when an unscanned
// parent precedes any match, or the primary is unknown or a black hole
// (upstream then also considers stars orbiting the black hole, which is
// not modelled).
func (e *evaluator) colours(table map[string]any) (found []string, complete bool) {
	set := map[string]bool{}
	complete = len(e.parentNames) > 0
	matched := false
	for _, name := range e.parentNames {
		s, ok := e.stars[name]
		if !ok {
			complete = false
			continue
		}
		if c, ok := colourFor(table, s.Type); ok {
			set[c] = true
			matched = true
			break
		}
	}
	if matched {
		complete = true
	}
	if e.main == nil || e.main.Type == "H" {
		complete = complete && matched
	}
	if e.main != nil && !containsName(e.parentNames, e.starKey(e.main.BodyName)) {
		if c, ok := colourFor(table, e.main.Type); ok {
			set[c] = true
		}
	}
	for c := range set {
		found = append(found, c)
	}
	sort.Strings(found)
	return found, complete
}

func colourFor(table map[string]any, kind string) (string, bool) {
	for _, k := range sortedKeys(table) {
		if starCheck(k, kind) {
			return text(table[k]), true
		}
	}
	return "", false
}

func containsName(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// sortedKeys keeps colour lookup deterministic. Upstream iterates the dict
// in source order and takes the first match; each star type matches at
// most one key in the pinned tables, so the order does not change results.
func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// elements derives element-dependent colours from the body's materials.
func (e *evaluator) elements(table map[string]any) (found []string, known bool) {
	mats, ok := e.body["Materials"].([]any)
	if !ok {
		return nil, false
	}
	for _, m := range mats {
		o, _ := m.(map[string]any)
		name := strings.ToLower(text(o["Name"]))
		if c, ok := table[name]; ok {
			found = append(found, text(c))
		}
	}
	sort.Strings(found)
	return found, true
}
