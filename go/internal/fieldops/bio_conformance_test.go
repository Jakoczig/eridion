package fieldops

import (
	"strings"
	"testing"
)

// Conformance fixtures for the native BioScan evaluator against the pinned
// EDMC-BioScan fc6b85c semantics (load.py value_estimate). Expected results
// are derived by hand from the pinned rule literals and upstream predicate
// code; see docs/testing/bioscan-conformance.md.

func zero() *float64 { v := 0.0; return &v }

// fixtureEnv is a single-star system: primary "Fixture Sys" (type F, at the
// arrival point) and the body "Fixture Sys 1" orbiting it.
func fixtureEnv(body map[string]any) Environment {
	if _, ok := body["BodyName"]; !ok {
		body["BodyName"] = "Fixture Sys 1"
	}
	return Environment{Body: body, SystemName: "Fixture Sys", Stars: []StarEvidence{{BodyName: "Fixture Sys", Type: "F", Luminosity: "V", DistanceLS: zero()}}}
}

func inference(t *testing.T, env Environment, genus, name string) Candidate {
	t.Helper()
	for _, c := range InferEnvironment(env, []string{genus}) {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("%s not sourced", name)
	return Candidate{}
}

func TestBioSourceCoverage(t *testing.T) {
	if len(pack.Species) != 119 {
		t.Fatalf("sourced species = %d, want 116 ruleset literals + 3 species.py literals = 119", len(pack.Species))
	}
	seen := map[string]bool{}
	for _, s := range pack.Species {
		k := s.Genus + "\x00" + s.Symbol
		if seen[k] {
			t.Fatalf("genus+species identity collapsed: %s", k)
		}
		seen[k] = true
	}
	for _, name := range []string{"Bark Mound", "Amphora Plant", "Radicoida Unicus"} {
		found := false
		for _, s := range pack.Species {
			found = found || s.Name == name
		}
		if !found {
			t.Errorf("%s (species.py) missing", name)
		}
	}
}

// Gravity uses upstream's 9.797759 divisor and inclusive bounds.
func TestBioGravityBoundary(t *testing.T) {
	const genus, name = "$Codex_Ent_Tussocks_Genus_Name;", "Tussock Stigmasis"
	body := func(g float64) Environment {
		return fixtureEnv(map[string]any{"AtmosphereType": "SulphurDioxide", "PlanetClass": "Rocky body", "SurfaceGravity": g, "SurfaceTemperature": 150.0, "SurfacePressure": 500.0, "Volcanism": ""})
	}
	if c := inference(t, body(0.276*GravityDivisor), genus, name); c.Inference == Excluded {
		t.Fatalf("max gravity is inclusive: %+v", c)
	}
	// 2.7043 m/s² is 0.27601 upstream units (excluded) but 0.27576 with
	// standard gravity 9.80665 (the pre-G3J native result: possible).
	if c := inference(t, body(2.7043), genus, name); c.Inference != Excluded {
		t.Fatalf("2.7043 m/s² should exceed max_gravity 0.276: %+v", c)
	}
	if c := inference(t, body(0.04*GravityDivisor-0.0001), genus, name); c.Inference != Excluded {
		t.Fatalf("below min gravity: %+v", c)
	}
}

// Pressure uses upstream's 101231.65625 divisor; max_pressure is exclusive
// (>= excludes). The audit probe: 1012.8 Pa reverses Tussock Stigmasis.
func TestBioPressureBoundary(t *testing.T) {
	const genus, name = "$Codex_Ent_Tussocks_Genus_Name;", "Tussock Stigmasis"
	body := func(p float64) Environment {
		return fixtureEnv(map[string]any{"AtmosphereType": "SulphurDioxide", "PlanetClass": "Rocky body", "SurfaceGravity": 1.0, "SurfaceTemperature": 150.0, "SurfacePressure": p, "Volcanism": ""})
	}
	if c := inference(t, body(1012.8), genus, name); c.Inference != Excluded {
		t.Fatalf("1012.8 Pa = 0.010005 upstream units >= max 0.01: %+v", c)
	}
	if c := inference(t, body(0.01*PressureDivisor), genus, name); c.Inference != Excluded {
		t.Fatalf("max_pressure equality excludes: %+v", c)
	}
	if c := inference(t, body(1012.0), genus, name); c.Inference == Excluded {
		t.Fatalf("1012.0 Pa is below max: %+v", c)
	}
	// Upstream skips a zero pressure ("not pressure"); min_pressure then passes.
	arcus := func(p float64) Candidate {
		return inference(t, fixtureEnv(map[string]any{"AtmosphereType": "CarbonDioxide", "PlanetClass": "Rocky body", "SurfaceGravity": 1.0, "SurfaceTemperature": 177.0, "SurfacePressure": p, "Volcanism": ""}), "$Codex_Ent_Aleoids_Genus_Name;", "Aleoida Arcus")
	}
	if c := arcus(0); c.Inference == Excluded {
		t.Fatalf("zero pressure is skipped upstream: %+v", c)
	}
	if c := arcus(0.0161*PressureDivisor - 1); c.Inference != Excluded {
		t.Fatalf("below min_pressure: %+v", c)
	}
}

func TestBioTemperatureBoundaryAndMissingEvidence(t *testing.T) {
	const genus, name = "$Codex_Ent_Aleoids_Genus_Name;", "Aleoida Arcus"
	body := map[string]any{"AtmosphereType": "CarbonDioxide", "PlanetClass": "Rocky body", "SurfaceGravity": 1.0, "SurfacePressure": 2000.0, "Volcanism": ""}
	for temp, want := range map[float64]string{175: Possible, 180: Possible, 174.99: Excluded, 180.01: Excluded} {
		body["SurfaceTemperature"] = temp
		if c := inference(t, fixtureEnv(body), genus, name); c.Inference != want {
			t.Errorf("temperature %v: %s, want %s", temp, c.Inference, want)
		}
	}
	delete(body, "SurfaceTemperature")
	c := inference(t, fixtureEnv(body), genus, name)
	if c.Inference != Unknown || !strings.Contains(strings.Join(c.Unknown, ","), "temperature not scanned") {
		t.Fatalf("missing temperature must stay unknown, never possible: %+v", c)
	}
}

func TestBioVolcanismForms(t *testing.T) {
	ev := newEvaluator(Environment{Body: map[string]any{"Volcanism": "minor rocky magma volcanism"}})
	for _, tc := range []struct {
		want any
		out  outcome
	}{
		{"Any", pass}, {"None", fail}, {"!rocky", fail}, {"!water", pass},
		{[]any{"rocky"}, pass}, {[]any{"=rocky magma volcanism"}, fail}, {[]any{"=minor rocky magma volcanism"}, pass}, {[]any{"water", "metallic"}, fail},
	} {
		if o, _ := ev.predicate("volcanism", tc.want); o != tc.out {
			t.Errorf("volcanism %v on minor rocky: %v, want %v", tc.want, o, tc.out)
		}
	}
	for _, none := range []string{"", "No volcanism"} {
		ev := newEvaluator(Environment{Body: map[string]any{"Volcanism": none}})
		if o, _ := ev.predicate("volcanism", "None"); o != pass {
			t.Errorf("%q should be no volcanism", none)
		}
		if o, _ := ev.predicate("volcanism", "!water"); o != fail {
			t.Errorf("'!' values require some volcanism (%q)", none)
		}
	}
	if o, _ := newEvaluator(Environment{Body: map[string]any{}}).predicate("volcanism", "None"); o != unknown {
		t.Error("missing volcanism must be unknown")
	}
}

func TestBioAtmosphereAndComposition(t *testing.T) {
	ev := newEvaluator(Environment{Body: map[string]any{"AtmosphereType": "SulphurDioxide", "AtmosphereComposition": []any{map[string]any{"Name": "SulphurDioxide", "Percent": 1.04}}}})
	if o, _ := ev.predicate("atmosphere_component", map[string]any{"SulphurDioxide": 1.05}); o != fail {
		t.Error("1.04% < 1.05% required")
	}
	if o, _ := ev.predicate("atmosphere_component", map[string]any{"SulphurDioxide": 1.04}); o != pass {
		t.Error("1.04% satisfies 1.04%")
	}
	none := newEvaluator(Environment{Body: map[string]any{"AtmosphereType": "None"}})
	if o, _ := none.predicate("atmosphere", "Any"); o != fail {
		t.Error("'Any' atmosphere excludes none")
	}
	if o, _ := none.predicate("atmosphere_component", map[string]any{"SulphurDioxide": 1.05}); o != fail {
		t.Error("no atmosphere has no gas")
	}
	thin := newEvaluator(Environment{Body: map[string]any{"AtmosphereType": "SulphurDioxide"}})
	if o, _ := thin.predicate("atmosphere_component", map[string]any{"SulphurDioxide": 1.05}); o != unknown {
		t.Error("composition not scanned must be unknown")
	}
}

func TestBioStellarAndSystemPredicates(t *testing.T) {
	be := func(lum string) StarEvidence {
		return StarEvidence{BodyName: "Fixture Sys B", Type: "B", Luminosity: lum}
	}
	env := Environment{Body: map[string]any{"BodyName": "Fixture Sys B 1"}, SystemName: "Fixture Sys", Stars: []StarEvidence{{BodyName: "Fixture Sys A", Type: "K", Luminosity: "V", DistanceLS: zero()}, be("IVab")}}
	ev := newEvaluator(env)
	tuples := []any{map[string]any{"tuple": []any{"B", "IV"}}, map[string]any{"tuple": []any{"B", "V"}}}
	if o, _ := ev.predicate("star", tuples); o != pass {
		t.Error("B IVab matches (B, IV) with the ab suffix")
	}
	if o, _ := ev.predicate("star", []any{[]any{"B", "IV"}}); o != unknown {
		t.Error("a plain list entry never matches upstream; incomplete system stays unknown")
	}
	env.AllBodiesFound = true
	if o, _ := newEvaluator(env).predicate("star", []any{"O"}); o != fail {
		t.Error("no O star in a fully scanned system")
	}
	if o, _ := ev.predicate("parent_star", []any{"B"}); o != pass {
		t.Error("parent star B of body 'B 1'")
	}
	if o, _ := ev.predicate("parent_star", []any{"A", "N"}); o != fail {
		t.Error("parent B and primary K are known and neither is A/N")
	}
	orphan := newEvaluator(Environment{Body: map[string]any{"BodyName": "Fixture Sys C 1"}, SystemName: "Fixture Sys", Stars: env.Stars})
	if o, _ := orphan.predicate("parent_star", []any{"A"}); o != unknown {
		t.Error("unscanned parent star C must stay unknown")
	}
	if o, _ := ev.predicate("system", "HIP 87621"); o != fail {
		t.Error("Radicoida Unicus only in HIP 87621")
	}
	if o, _ := ev.predicate("regions", []any{"orion-cygnus"}); o != unknown {
		t.Error("regions are not bundled: unknown")
	}
	if o, _ := ev.predicate("region", []any{"brain-tree"}); o != pass {
		t.Error("singular 'region' is not evaluated upstream")
	}
	if o, _ := ev.predicate("max_orbital_period", 86400); o != unknown {
		t.Error("orbital period not scanned")
	}
}

func TestBioBodiesZonesAndNebula(t *testing.T) {
	env := Environment{Body: map[string]any{}, SystemName: "Fixture Sys", Planets: []PlanetEvidence{{BodyName: "Fixture Sys 2", Class: "Icy body"}}}
	want := []any{"Earthlike body", "Water giant"}
	if o, _ := newEvaluator(env).predicate("bodies", want); o != unknown {
		t.Error("incomplete system: unknown")
	}
	env.AllBodiesFound = true
	if o, _ := newEvaluator(env).predicate("bodies", want); o != fail {
		t.Error("complete system without the body type: excluded")
	}
	env.Planets = append(env.Planets, PlanetEvidence{BodyName: "Fixture Sys 3", Class: "Water giant"})
	if o, _ := newEvaluator(env).predicate("bodies", want); o != pass {
		t.Error("water giant present")
	}
	// Guardian zone Gamma Velorum (750 ly around 1099.2, -146.7, -133.6).
	near, far := [3]float64{1099, -146, -133}, [3]float64{0, 0, 0}
	if o, _ := newEvaluator(Environment{Coords: &near}).predicate("guardian", true); o != pass {
		t.Error("inside Gamma Velorum zone")
	}
	if o, _ := newEvaluator(Environment{Coords: &far}).predicate("guardian", true); o != fail {
		t.Error("Sol is outside every guardian zone")
	}
	if o, _ := newEvaluator(Environment{}).predicate("guardian", true); o != unknown {
		t.Error("guardian without coordinates: unknown")
	}
	// Tuber zone Inner S-C Arm B 1: 150-300 ly around (-6645, 0, 12590).
	ring := [3]float64{-6645 + 200, 0, 12590}
	if o, _ := newEvaluator(Environment{Coords: &ring}).predicate("tuber", []any{"Inner S-C Arm B 1"}); o != pass {
		t.Error("200 ly from the tuber zone centre is inside 150-300")
	}
	centre := [3]float64{-6645, 0, 12590}
	if o, _ := newEvaluator(Environment{Coords: &centre}).predicate("tuber", []any{"Inner S-C Arm B 1"}); o != fail {
		t.Error("the zone centre is inside the 150 ly inner radius")
	}
	if o, _ := newEvaluator(Environment{SystemName: "Crab Sector DL-Y d9"}).predicate("nebula", "large"); o != pass {
		t.Error("nebula sector name")
	}
	if o, _ := newEvaluator(Environment{SystemName: "Fixture Sys"}).predicate("nebula", "large"); o != unknown {
		t.Error("nebula reference stars are not bundled: unknown")
	}
}

// Colour (variant) dependencies: star colours from the parent star, element
// colours from surface materials. Absent evidence stays unknown.
func TestBioVariantDependencies(t *testing.T) {
	nebulus := map[string]any{"AtmosphereType": "Helium", "PlanetClass": "Icy body", "SurfaceGravity": 0.45 * GravityDivisor, "SurfaceTemperature": 20.5, "SurfacePressure": 0.1 * PressureDivisor, "Volcanism": ""}
	c := inference(t, fixtureEnv(nebulus), "$Codex_Ent_Bacterial_Genus_Name;", "Bacterium Nebulus")
	if c.Inference != Unknown || !strings.Contains(strings.Join(c.Unknown, ","), "surface materials") {
		t.Fatalf("element colours need materials: %+v", c)
	}
	nebulus["Materials"] = []any{map[string]any{"Name": "iron", "Percent": 20.0}}
	if c := inference(t, fixtureEnv(nebulus), "$Codex_Ent_Bacterial_Genus_Name;", "Bacterium Nebulus"); c.Inference != Excluded {
		t.Fatalf("no colour element on the body: %+v", c)
	}
	nebulus["Materials"] = []any{map[string]any{"Name": "iron", "Percent": 20.0}, map[string]any{"Name": "polonium", "Percent": 0.5}}
	if c := inference(t, fixtureEnv(nebulus), "$Codex_Ent_Bacterial_Genus_Name;", "Bacterium Nebulus"); c.Inference != Possible || len(c.Variants) != 1 || c.Variants[0] != "Gold" {
		t.Fatalf("polonium gives the Gold variant: %+v", c)
	}
	// Aleoida colours come from the parent star; F gives Teal.
	arcus := map[string]any{"AtmosphereType": "CarbonDioxide", "PlanetClass": "Rocky body", "SurfaceGravity": 1.0, "SurfaceTemperature": 177.0, "SurfacePressure": 2000.0, "Volcanism": ""}
	if c := inference(t, fixtureEnv(arcus), "$Codex_Ent_Aleoids_Genus_Name;", "Aleoida Arcus"); c.Inference != Possible || len(c.Variants) != 1 || c.Variants[0] != "Teal" {
		t.Fatalf("Aleoida Arcus under an F star: %+v", c)
	}
	if c := inference(t, Environment{Body: arcus}, "$Codex_Ent_Aleoids_Genus_Name;", "Aleoida Arcus"); c.Inference != Unknown {
		t.Fatalf("without the parent star the colour, hence the species, is unknown: %+v", c)
	}
}

func TestBioAlternativesAndUnsourced(t *testing.T) {
	// Bacterium Nebulus has two alternatives; the Rocky ice body branch
	// allows gravity up to 0.7.
	body := map[string]any{"AtmosphereType": "Helium", "PlanetClass": "Rocky ice body", "SurfaceGravity": 0.65 * GravityDivisor, "SurfaceTemperature": 20.5, "SurfacePressure": 0.1 * PressureDivisor, "Volcanism": "", "Materials": []any{map[string]any{"Name": "yttrium", "Percent": 1.0}}}
	if c := inference(t, fixtureEnv(body), "$Codex_Ent_Bacterial_Genus_Name;", "Bacterium Nebulus"); c.Inference != Possible {
		t.Fatalf("second alternative should match: %+v", c)
	}
	// An alternative with an unknown predicate keeps the species unknown,
	// never dropped to manufacture a match or an exclusion.
	if c := inference(t, fixtureEnv(map[string]any{"Volcanism": "minor rocky magma volcanism"}), "$Codex_Ent_Cone_Name;", "Bark Mound"); c.Inference != Unknown {
		t.Fatalf("Bark Mound needs nebula/region evidence: %+v", c)
	}
	if c := inference(t, fixtureEnv(map[string]any{}), "$Codex_Ent_Stratum_04_Name;", "Stratum Aranaemus"); c.Inference != Unknown || c.Unknown[0] != "no sourced criteria" {
		t.Fatalf("a species without sourced rulesets is unknown, not possible: %+v", c)
	}
	env := fixtureEnv(map[string]any{})
	env.SystemName = "HIP 87621"
	env.Body["BodyName"] = "HIP 87621 1"
	env.Stars[0].BodyName = "HIP 87621"
	if c := inference(t, env, "$Codex_Ent_Ingensradices_Genus_Name;", "Radicoida Unicus"); c.Inference != Possible {
		t.Fatalf("Radicoida Unicus in its system: %+v", c)
	}
}
