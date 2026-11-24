package catalog

import (
	"strconv"
	"strings"
)

// Module identity outcomes (Task 16 / F20).
const (
	StatusExactID              = "exact_id"              // numeric ID and symbol agree
	StatusUniqueSymbol         = "unique_symbol"         // only one module uses the symbol
	StatusStandardUnengineered = "standard_unengineered" // shared symbol, no engineering: the standard module
	StatusAmbiguous            = "ambiguous_variant"     // shared symbol, engineered: variant not reported
	StatusUnknown              = "unknown"               // not in the bundled catalogue (or ID/symbol mismatch)
)

// Resolution is the outcome of identifying one observed module.
type Resolution struct {
	Status     string
	ID         string
	Module     Module
	Candidates []string // candidate names when ambiguous
}

// ResolveModule identifies a module from its symbol, an optional numeric ID
// (CAPI, or one carried over after a verified slot match) and whether the
// observation reports engineering. It never guesses: an ID that names another
// module or an unknown identifier is unknown; a shared symbol with
// engineering stays ambiguous. Reference statistics attach only to an exact
// or standard identification; pre-built variants get name/class/rating only,
// because the bundled stock statistics describe standard modules.
func ResolveModule(symbol, id string, engineered bool) Resolution {
	once.Do(initData)
	key := moduleKey(symbol)
	ids := candidates[key]
	if id != "" {
		e, ok := entries[id]
		if !ok || moduleKey(e.Symbol) != key {
			return Resolution{Status: StatusUnknown}
		}
		r := Resolution{Status: StatusExactID, ID: id, Module: Module{Source: "bundled_edcd_fdevids", Name: e.Name, Rating: e.Rating, Class: copyPtr(e.Class)}}
		if ids[0] == id {
			r.Module = referenceCopy(modules[key])
		}
		return r
	}
	switch {
	case len(ids) == 0:
		if m, ok := ModuleBySymbol(symbol); ok { // display-only special cases
			return Resolution{Status: StatusUniqueSymbol, Module: m}
		}
		return Resolution{Status: StatusUnknown}
	case len(ids) == 1:
		return Resolution{Status: StatusUniqueSymbol, ID: ids[0], Module: referenceCopy(modules[key])}
	case !engineered:
		// Every reviewed collision is a standard module plus pre-built
		// variants, which are always engineered; an unengineered
		// observation is therefore the standard module.
		return Resolution{Status: StatusStandardUnengineered, ID: ids[0], Module: referenceCopy(modules[key])}
	default:
		return Resolution{Status: StatusAmbiguous, Candidates: candidateNames(symbol)}
	}
}

// ModuleByID returns the outfitting entry for an exact numeric ID.
func ModuleByID(id string) (Entry, bool) {
	once.Do(initData)
	e, ok := entries[id]
	e.Class = copyPtr(e.Class)
	return e, ok
}

func referenceCopy(m Module) Module {
	m.Class = copyPtr(m.Class)
	m.MassT = copyPtr(m.MassT)
	m.PowerMW = copyPtr(m.PowerMW)
	return m
}

func candidateNames(symbol string) []string {
	once.Do(initData)
	var names []string
	seen := map[string]bool{}
	for _, id := range candidates[moduleKey(symbol)] {
		if n := entries[id].Name; !seen[n] {
			names = append(names, n)
			seen[n] = true
		}
	}
	return names
}

func numericLess(a, b string) bool {
	x, e1 := strconv.ParseUint(a, 10, 64)
	y, e2 := strconv.ParseUint(b, 10, 64)
	if e1 != nil || e2 != nil {
		return strings.Compare(a, b) < 0
	}
	return x < y
}
