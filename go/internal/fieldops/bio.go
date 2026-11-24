package fieldops

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"math"
	"sort"
	"time"
)

//go:embed bio-rules.json
var bioRules []byte

// Candidate is one sourced species for a body. Inference is the
// three-valued result; State is the legacy display string kept for older
// clients. Unknown lists the evidence or capability each unresolved
// alternative still needs; Variants are the colour variants the evidence
// allows (empty when not determined).
type Candidate struct {
	Genus     string           `json:"genus"`
	Symbol    string           `json:"symbol"`
	Name      string           `json:"name"`
	State     string           `json:"state"`
	Inference string           `json:"inference"`
	Evidence  []string         `json:"evidence"`
	Unknown   []string         `json:"unknown"`
	Variants  []string         `json:"variants"`
	MinimumM  *float64         `json:"minimumM"`
	BaseValue *float64         `json:"baseValue"`
	Criteria  []map[string]any `json:"criteria"`
}

type criteriaSpecies struct {
	Genus, Symbol, Name, File string
	Value                     float64
	Rulesets                  []map[string]any
}
type criteriaGenus struct {
	ID, Name string
	Distance float64
	Colors   map[string]any
}
type criteriaZone struct {
	Name         string
	MinLY, MaxLY float64
	Center       [3]float64
}
type criteriaPack struct {
	Format                    int
	Version, Notice           string
	Genera                    []criteriaGenus
	Species                   []criteriaSpecies
	GuardianZones, TuberZones []criteriaZone
	NebulaSectors             []string
	genus                     map[string]criteriaGenus
}

var pack = func() criteriaPack {
	var p criteriaPack
	if json.Unmarshal(bioRules, &p) != nil || p.Format != 2 || len(p.Species) == 0 {
		panic("Invalid embedded BioScan criteria")
	}
	p.genus = map[string]criteriaGenus{}
	for _, g := range p.Genera {
		p.genus[g.ID] = g
	}
	return p
}()

// BioRuleVersion identifies the pinned rule data and native format.
func BioRuleVersion() string { return pack.Version }

// MinimumDistance is the genus's clonal colony distance (ExploData).
func MinimumDistance(genus string) *float64 {
	if g, ok := pack.genus[genus]; ok && g.Distance > 0 {
		v := g.Distance
		return &v
	}
	return nil
}

// GenusName is the reference (English) genus name for a genus symbol, for
// presentation fallback only; identity is always the symbol.
func GenusName(genus string) string { return pack.genus[genus].Name }

var legacyState = map[string]string{Possible: "possible by known criteria", Excluded: "excluded by a witnessed condition", Unknown: "unknown criteria"}

// Infer evaluates the body's Scan fields alone (no system context), kept for
// callers without system evidence: star, parent, body, zone and region
// predicates then remain unknown.
func Infer(scan map[string]any, genera []string) []Candidate {
	return InferEnvironment(Environment{Body: scan}, genera)
}

// InferEnvironment evaluates every sourced species of the given genus
// symbols (all species when genera is empty) against the evidence.
func InferEnvironment(env Environment, genera []string) []Candidate {
	e := newEvaluator(env)
	out := []Candidate{}
	wanted := map[string]bool{}
	for _, g := range genera {
		wanted[g] = true
	}
	genusVariants := map[string][]string{}
	genusOutcome := map[string]outcome{}
	for _, s := range pack.Species {
		if len(wanted) > 0 && !wanted[s.Genus] {
			continue
		}
		c := Candidate{Genus: s.Genus, Symbol: s.Symbol, Name: s.Name, Evidence: []string{"Pinned community criteria " + pack.Version}, Unknown: []string{}, Variants: []string{}, MinimumM: MinimumDistance(s.Genus), Criteria: s.Rulesets}
		if s.Value > 0 {
			v := s.Value
			c.BaseValue = &v
		}
		result := fail
		reasons := map[string]bool{}
		if len(s.Rulesets) == 0 {
			result = unknown
			reasons["no sourced criteria"] = true
		}
		for _, rule := range s.Rulesets {
			r := pass
			missing := []string{}
			for _, k := range sortedKeys(rule) {
				o, why := e.predicate(k, rule[k])
				r = combine(r, o)
				if o == fail {
					break
				}
				if o == unknown {
					missing = append(missing, why)
				}
			}
			if r == unknown {
				for _, m := range missing {
					reasons[m] = true
				}
			}
			if r < result {
				result = r
			}
		}
		// Colour (variant) dependencies can still exclude a species.
		if result != fail {
			g := pack.genus[s.Genus]
			if spec, ok := g.Colors["species"].(map[string]any); ok {
				entry, _ := spec[s.Symbol].(map[string]any)
				if table, ok := entry["star"].(map[string]any); ok {
					found, complete := e.colours(table)
					c.Variants = found
					if len(found) == 0 {
						if complete {
							result = fail
						} else {
							result = combine(result, unknown)
							reasons["parent star needed for colour"] = true
						}
					}
				} else if table, ok := entry["element"].(map[string]any); ok {
					found, known := e.elements(table)
					c.Variants = found
					if len(found) == 0 {
						if known {
							result = fail
						} else {
							result = combine(result, unknown)
							reasons["surface materials needed for colour"] = true
						}
					}
				}
			} else if table, ok := g.Colors["star"].(map[string]any); ok {
				if _, done := genusOutcome[s.Genus]; !done {
					found, complete := e.colours(table)
					genusVariants[s.Genus] = found
					switch {
					case len(found) > 0:
						genusOutcome[s.Genus] = pass
					case complete:
						genusOutcome[s.Genus] = fail
					default:
						genusOutcome[s.Genus] = unknown
					}
				}
				c.Variants = genusVariants[s.Genus]
				switch genusOutcome[s.Genus] {
				case fail:
					result = fail
				case unknown:
					result = combine(result, unknown)
					reasons["parent star needed for colour"] = true
				}
			}
		}
		switch result {
		case pass:
			c.Inference = Possible
		case fail:
			c.Inference = Excluded
		default:
			c.Inference = Unknown
			for r := range reasons {
				c.Unknown = append(c.Unknown, r)
			}
			sort.Strings(c.Unknown)
		}
		if c.Variants == nil {
			c.Variants = []string{}
		}
		c.State = legacyState[c.Inference]
		c.Evidence = append(c.Evidence, "Possible is a community suggestion, never a confirmed observation")
		out = append(out, c)
		if len(out) >= 128 {
			break
		}
	}
	return out
}

func floatValue(x any) (float64, bool) {
	var v float64
	switch x := x.(type) {
	case float64:
		v = x
	case json.Number:
		var e error
		v, e = x.Float64()
		if e != nil {
			return 0, false
		}
	default:
		return 0, false
	}
	return v, !math.IsNaN(v) && !math.IsInf(v, 0)
}

type Position struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius    float64 `json:"radiusM"`
	At        string  `json:"at"`
}
type Spacing struct {
	At                string   `json:"at"`
	DistanceM         *float64 `json:"distanceM"`
	RemainingM        *float64 `json:"remainingM"`
	AllLocationsKnown bool     `json:"allLocationsKnown"`
	Notice            string   `json:"notice"`
}

func LiveSpacing(current Position, previous []Position, minimum *float64, complete bool) Spacing {
	out := Spacing{At: current.At, AllLocationsKnown: complete, Notice: "Derived from current Status and approximate sample snapshots; never game-confirmed eligibility"}
	for _, p := range previous {
		if d, ok := SurfaceDistance(current, p); ok {
			if out.DistanceM == nil || d < *out.DistanceM {
				out.DistanceM = &d
			}
		} else {
			out.AllLocationsKnown = false
		}
	}
	if out.DistanceM != nil && minimum != nil && out.AllLocationsKnown {
		n := math.Max(0, *minimum-*out.DistanceM)
		out.RemainingM = &n
	}
	return out
}

func validPosition(p Position) bool {
	return !math.IsNaN(p.Latitude) && !math.IsInf(p.Latitude, 0) && math.Abs(p.Latitude) <= 90 && !math.IsNaN(p.Longitude) && !math.IsInf(p.Longitude, 0) && math.Abs(p.Longitude) <= 180 && p.Radius > 0 && p.Radius < 1e9
}
func SurfaceDistance(a, b Position) (float64, bool) {
	if !validPosition(a) || !validPosition(b) || math.Abs(a.Radius-b.Radius) > 1 {
		return 0, false
	}
	lat1, lat2 := a.Latitude*math.Pi/180, b.Latitude*math.Pi/180
	dlat, dlon := lat2-lat1, (b.Longitude-a.Longitude)*math.Pi/180
	x := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dlon/2), 2)
	return 2 * a.Radius * math.Asin(math.Sqrt(math.Max(0, math.Min(1, x)))), true
}
func StatusPosition(raw []byte, body string, sample, now time.Time) (Position, bool) {
	var v map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&v) != nil || v["event"] != "Status" || body == "" || v["BodyName"] != body {
		return Position{}, false
	}
	flags, ok := v["Flags"].(json.Number)
	if !ok {
		return Position{}, false
	}
	n, e := flags.Int64()
	if e != nil || n&(1<<21) == 0 {
		return Position{}, false
	}
	at, e := time.Parse(time.RFC3339Nano, stringValue(v["timestamp"]))
	if e != nil || math.Abs(at.Sub(sample).Seconds()) > 2 || math.Abs(now.Sub(at).Seconds()) > 5 {
		return Position{}, false
	}
	lat, a := floatValue(v["Latitude"])
	lon, b := floatValue(v["Longitude"])
	radius, c := floatValue(v["PlanetRadius"])
	p := Position{lat, lon, radius, at.UTC().Format(time.RFC3339Nano)}
	return p, a && b && c && validPosition(p)
}
func stringValue(v any) string { s, _ := v.(string); return s }
