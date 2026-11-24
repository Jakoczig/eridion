// Package catalog contains reference data selected from the repository's bundled
// EDCD FDevIDs CSVs and Coriolis modules. These are static reference facts, never
// measurements of a player's ship. No user files, runtime IO or network is used.
package catalog

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

//go:embed *.csv modules.json collisions-reviewed.txt entitlement-review.csv
var data embed.FS

type Module struct {
	Source, Name, Rating, Description string
	Class                             *int
	MassT, PowerMW                    *float64
}

var once sync.Once

// modules holds reference data for symbols that identify exactly one module
// (and, for a shared symbol, its standard module; see ResolveModule).
var modules map[string]Module
var labels map[string]string

// Entry is one outfitting row, keyed first by its exact numeric ID (Task 16).
type Entry struct {
	ID, Symbol, Category, Name, Mount, Guidance, Ship, Rating, Entitlement string
	Class                                                                  *int
}

var entries map[string]Entry       // numeric ID -> entry
var candidates map[string][]string // moduleKey(symbol) -> IDs, ascending

func initData() {
	modules = map[string]Module{}
	labels = map[string]string{}
	entries = map[string]Entry{}
	candidates = map[string][]string{}
	for _, filename := range []string{"shipyard.csv", "outfitting.csv", "material.csv", "microresources.csv", "commodity.csv"} {
		b, _ := data.ReadFile(filename)
		rows, err := csv.NewReader(strings.NewReader(string(b))).ReadAll()
		if err != nil || len(rows) == 0 {
			continue
		}
		cols := map[string]int{}
		for i, h := range rows[0] {
			cols[h] = i
		}
		nameCol, ok := cols["name"]
		if !ok {
			nameCol = cols["English name"]
		}
		symbolCol := cols["symbol"]
		for _, row := range rows[1:] {
			if len(row) <= nameCol || len(row) <= symbolCol {
				continue
			}
			key := moduleKey(row[symbolCol])
			if filename != "outfitting.csv" {
				labels[strings.ToLower(row[symbolCol])] = row[nameCol]
				continue
			}
			e := Entry{ID: row[cols["id"]], Symbol: row[symbolCol], Category: row[cols["category"]], Name: row[nameCol], Mount: row[cols["mount"]], Guidance: row[cols["guidance"]], Ship: row[cols["ship"]], Rating: row[cols["rating"]], Entitlement: row[cols["entitlement"]]}
			if n, err := strconv.Atoi(row[cols["class"]]); err == nil && n >= 0 && n <= 8 {
				e.Class = &n
			}
			entries[e.ID] = e
			candidates[key] = append(candidates[key], e.ID)
		}
	}
	// Reference data per symbol. A shared symbol's reference describes its
	// standard module: the lowest ID (every reviewed collision is a standard
	// module plus newer pre-built variants; collisions-reviewed.txt).
	for key, ids := range candidates {
		sort.Slice(ids, func(i, j int) bool { return numericLess(ids[i], ids[j]) })
		e := entries[ids[0]]
		modules[key] = Module{Source: "bundled_edcd_coriolis_reference", Name: e.Name, Rating: e.Rating, Class: e.Class}
		if len(ids) == 1 {
			labels[strings.ToLower(e.Symbol)] = e.Name
		}
	}
	b, _ := data.ReadFile("modules.json")
	var rows []struct {
		Symbol, Name, Rating, Description string
		Class                             *int
		Mass, Power                       *float64
	}
	if json.Unmarshal(b, &rows) != nil {
		return
	}
	for _, r := range rows {
		key := moduleKey(r.Symbol)
		v := modules[key]
		v.Source = "bundled_edcd_coriolis_reference"
		if v.Name == "" && r.Name != "" {
			v.Name = r.Name
		}
		if r.Class != nil {
			v.Class = r.Class
		}
		v.Rating = r.Rating
		v.MassT = r.Mass
		v.PowerMW = r.Power
		v.Description = r.Description
		modules[key] = v
	}
}
func copyPtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// ModuleBySymbol returns reference data for a symbol that identifies exactly
// one module. A symbol shared by several variants is not resolved here; use
// ResolveModule with the observation's numeric ID or engineering evidence.
func ModuleBySymbol(symbol string) (Module, bool) {
	once.Do(initData)
	if len(candidates[moduleKey(symbol)]) > 1 {
		return Module{}, false
	}
	m, ok := modules[moduleKey(symbol)]
	if !ok && moduleKey(symbol) == "intfighterbaymk2s" {
		m = Module{Name: "Fighter Hangar Mk II", Source: "symbol_display_name_only", Description: "Display name only; class, rating and performance not verified."}
		ok = true
	}
	m.Class = copyPtr(m.Class)
	m.MassT = copyPtr(m.MassT)
	m.PowerMW = copyPtr(m.PowerMW)
	return m, ok
}
func Label(symbol string) string {
	once.Do(initData)
	if label := labels[strings.ToLower(symbol)]; label != "" {
		return label
	}
	return symbol
}

// Formatting aliases never map unknown variants to another module's statistics.
func moduleKey(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

var wordBreak = regexp.MustCompile(`([a-z])([A-Z0-9])`)

func ModuleDisplayName(symbol string) string {
	if m, ok := ModuleBySymbol(symbol); ok && m.Name != "" {
		return m.Name
	}
	if names := candidateNames(symbol); len(names) > 1 {
		return strings.Join(names, " or ")
	}
	s := strings.Trim(strings.TrimSpace(symbol), "$;")
	s = wordBreak.ReplaceAllString(s, "$1 $2")
	s = strings.ReplaceAll(s, "_", " ")
	parts := strings.Fields(strings.ToLower(s))
	if len(parts) > 0 && (parts[0] == "int" || parts[0] == "hpt") {
		parts = parts[1:]
	}
	for i, w := range parts {
		r := []rune(w)
		if len(r) > 0 {
			r[0] = unicode.ToUpper(r[0])
			parts[i] = string(r)
		}
	}
	if len(parts) == 0 {
		return "Unidentified module"
	}
	return strings.Join(parts, " ")
}

// ShipDisplayName is a label only; it never substitutes another hull's stats.
func ShipDisplayName(symbol string) string {
	if strings.TrimSpace(symbol) == "" {
		return ""
	}
	once.Do(initData)
	if v := labels[strings.ToLower(symbol)]; v != "" {
		return v
	}
	return ModuleDisplayName(symbol)
}
