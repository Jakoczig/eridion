package bioconvert

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Pinned upstream revisions of the vendored files (see
// upstream-sources.lock.json, ids silarn-bioscan and silarn-explodata).
const (
	BioScanRevision   = "fc6b85c3de48c8fb79fa9e4f79ad780e48f56629"
	ExploDataRevision = "fd4e188defd6405e46398fb0a54ac8669c9e2189"
	// FormatVersion changes when the generated structure changes.
	FormatVersion = 2
)

// Source records one vendored upstream file.
type Source struct {
	Repository string `json:"repository"`
	Revision   string `json:"revision"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
}

// Species is one sourced species with its alternative rulesets. Each ruleset
// maps a predicate key to its literal value; tuples are encoded as
// {"tuple": [...]} because upstream treats them differently from lists.
type Species struct {
	Genus    string           `json:"genus"`
	Symbol   string           `json:"symbol"`
	Name     string           `json:"name"`
	Value    int64            `json:"value"`
	File     string           `json:"file"`
	Rulesets []map[string]any `json:"rulesets"`
}

// Genus holds the ExploData genus record: label, clonal distance and the
// colour (variant) dependencies.
type Genus struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Distance float64        `json:"distance"`
	Colors   map[string]any `json:"colors,omitempty"`
}

// Zone is a guardian or tuber zone from regions.py.
type Zone struct {
	Name   string     `json:"name"`
	MinLY  float64    `json:"minLy"`
	MaxLY  float64    `json:"maxLy"`
	Center [3]float64 `json:"center"`
}

// Pack is the generated native rule file.
type Pack struct {
	Format        int              `json:"format"`
	Version       string           `json:"version"`
	Notice        string           `json:"notice"`
	Sources       []Source         `json:"sources"`
	Genera        []Genus          `json:"genera"`
	Species       []Species        `json:"species"`
	GuardianZones []Zone           `json:"guardianZones"`
	TuberZones    []Zone           `json:"tuberZones"`
	NebulaSectors []string         `json:"nebulaSectors"`
	RegionMap     map[string][]int `json:"regionMap"`
}

var importLine = regexp.MustCompile(`(?m)^from bio_scan\.bio_data\.rulesets\.(\w+) import catalog as (\w+)$`)

// Convert reads the vendored upstream directory (containing bioscan/ and
// explodata/) and builds the pack.
func Convert(dir string) (*Pack, error) {
	p := &Pack{Format: FormatVersion, RegionMap: map[string][]int{}}
	read := func(repo, rev, upstreamPath, local string) (string, error) {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(local)))
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(b)
		p.Sources = append(p.Sources, Source{repo, rev, upstreamPath, hex.EncodeToString(sum[:])})
		return string(b), nil
	}
	const bio, explo = "https://github.com/Silarn/EDMC-BioScan", "https://github.com/Silarn/EDMC-ExploData"
	speciesSrc, err := read(bio, BioScanRevision, "src/bio_scan/bio_data/species.py", "bioscan/species.py")
	if err != nil {
		return nil, err
	}
	// Merge order follows species.py: the in-file catalogue first, then the
	// ruleset modules in the order of the `rules` union expression.
	alias := map[string]string{}
	for _, m := range importLine.FindAllStringSubmatch(speciesSrc, -1) {
		alias[m[2]] = m[1]
	}
	unionAt := strings.Index(speciesSrc, "\nrules:")
	if unionAt < 0 {
		return nil, fmt.Errorf("species.py: rules union not found")
	}
	union := speciesSrc[unionAt:]
	union = union[strings.Index(union, "(")+1 : strings.Index(union, ")")]
	type part struct{ file, src string }
	var parts []part
	for _, name := range strings.Split(union, "|") {
		name = strings.TrimSpace(name)
		switch {
		case name == "_mound_amphora":
			parts = append(parts, part{"species.py", ""})
		case alias[name] != "":
			src, err := read(bio, BioScanRevision, "src/bio_scan/bio_data/rulesets/"+alias[name]+".py", "bioscan/rulesets/"+alias[name]+".py")
			if err != nil {
				return nil, err
			}
			parts = append(parts, part{alias[name] + ".py", src})
		default:
			return nil, fmt.Errorf("species.py: unknown union member %q", name)
		}
	}
	seen := map[string]bool{}
	for _, pt := range parts {
		var v any
		if pt.file == "species.py" {
			v, err = Assignment(speciesSrc, "_mound_amphora")
		} else {
			v, err = Assignment(pt.src, "catalog")
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", pt.file, err)
		}
		genera, ok := v.(*Dict)
		if !ok {
			return nil, fmt.Errorf("%s: catalogue is not a dict", pt.file)
		}
		for _, g := range genera.Keys {
			sp, ok := genera.Values[g].(*Dict)
			if !ok {
				return nil, fmt.Errorf("%s: genus %s is not a dict", pt.file, g)
			}
			for _, s := range sp.Keys {
				rec, ok := sp.Values[s].(*Dict)
				if !ok {
					return nil, fmt.Errorf("%s: species %s is not a dict", pt.file, s)
				}
				if seen[g+"\x00"+s] {
					return nil, fmt.Errorf("%s: duplicate species %s", pt.file, s)
				}
				seen[g+"\x00"+s] = true
				name, _ := rec.Values["name"].(string)
				value, _ := rec.Values["value"].(int64)
				rs, _ := rec.Values["rulesets"].([]any)
				out := Species{Genus: g, Symbol: s, Name: name, Value: value, File: pt.file, Rulesets: []map[string]any{}}
				for _, r := range rs {
					d, ok := r.(*Dict)
					if !ok {
						return nil, fmt.Errorf("%s: %s ruleset is not a dict", pt.file, s)
					}
					out.Rulesets = append(out.Rulesets, plain(d).(map[string]any))
				}
				if name == "" {
					return nil, fmt.Errorf("%s: %s lacks a name", pt.file, s)
				}
				p.Species = append(p.Species, out)
			}
		}
	}

	genusSrc, err := read(explo, ExploDataRevision, "src/ExploData/explo_data/bio_data/genus.py", "explodata/genus.py")
	if err != nil {
		return nil, err
	}
	gv, err := Assignment(genusSrc, "data")
	if err != nil {
		return nil, err
	}
	gd, _ := gv.(*Dict)
	if gd == nil {
		return nil, fmt.Errorf("genus.py: data is not a dict")
	}
	for _, id := range gd.Keys {
		rec, _ := gd.Values[id].(*Dict)
		if rec == nil {
			return nil, fmt.Errorf("genus.py: %s is not a dict", id)
		}
		g := Genus{ID: id}
		g.Name, _ = rec.Values["name"].(string)
		g.Distance = number(rec.Values["distance"])
		if c, ok := rec.Values["colors"].(*Dict); ok {
			g.Colors = plain(c).(map[string]any)
		}
		p.Genera = append(p.Genera, g)
	}

	regionSrc, err := read(bio, BioScanRevision, "src/bio_scan/bio_data/regions.py", "bioscan/regions.py")
	if err != nil {
		return nil, err
	}
	rm, err := Assignment(regionSrc, "region_map")
	if err != nil {
		return nil, err
	}
	for _, k := range rm.(*Dict).Keys {
		for _, n := range rm.(*Dict).Values[k].([]any) {
			p.RegionMap[k] = append(p.RegionMap[k], int(n.(int64)))
		}
	}
	gz, err := Assignment(regionSrc, "guardian_nebulae")
	if err != nil {
		return nil, err
	}
	for _, k := range gz.(*Dict).Keys {
		t := gz.(*Dict).Values[k].(Tuple)
		p.GuardianZones = append(p.GuardianZones, Zone{Name: k, MaxLY: number(t[0]), Center: coords(t[1])})
	}
	tz, err := Assignment(regionSrc, "tuber_zones")
	if err != nil {
		return nil, err
	}
	for _, k := range tz.(*Dict).Keys {
		t := tz.(*Dict).Values[k].(Tuple)
		r := t[0].(Tuple)
		p.TuberZones = append(p.TuberZones, Zone{Name: k, MinLY: number(r[0]), MaxLY: number(r[1]), Center: coords(t[1])})
	}
	sectorSrc, err := read(bio, BioScanRevision, "src/bio_scan/nebula_data/sectors.py", "bioscan/nebula_sectors.py")
	if err != nil {
		return nil, err
	}
	sv, err := Assignment(sectorSrc, "data")
	if err != nil {
		return nil, err
	}
	for _, s := range sv.([]any) {
		p.NebulaSectors = append(p.NebulaSectors, s.(string))
	}
	sort.SliceStable(p.Sources, func(i, j int) bool { return p.Sources[i].Path < p.Sources[j].Path })
	p.Version = fmt.Sprintf("bioscan-%s-explodata-%s-native-%d", BioScanRevision[:7], ExploDataRevision[:7], FormatVersion)
	p.Notice = "Pinned community criteria (EDMC-BioScan / ExploData, GPL-2.0-or-later, GPL-3.0 selected). Possible is never confirmed; evidence that is missing stays unknown. Values are base community references, never sale or discovery guarantees."
	return p, nil
}

// Marshal renders the pack deterministically.
func Marshal(p *Pack) ([]byte, error) {
	b, err := json.MarshalIndent(p, "", " ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func plain(v any) any {
	switch x := v.(type) {
	case *Dict:
		m := map[string]any{}
		for _, k := range x.Keys {
			m[k] = plain(x.Values[k])
		}
		return m
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = plain(x[i])
		}
		return out
	case Tuple:
		out := make([]any, len(x))
		for i := range x {
			out[i] = plain(x[i])
		}
		return map[string]any{"tuple": out}
	}
	return v
}

func number(v any) float64 {
	switch x := v.(type) {
	case int64:
		return float64(x)
	case float64:
		return x
	}
	return 0
}

func coords(v any) [3]float64 {
	t, _ := v.(Tuple)
	var out [3]float64
	for i := 0; i < 3 && i < len(t); i++ {
		out[i] = number(t[i])
	}
	return out
}
