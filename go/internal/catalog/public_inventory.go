package catalog

import (
	"encoding/csv"
	"regexp"
	"strings"
	"sync"
)

// Public station inventory classification for EDDN shipyard/outfitting
// publication (Task 8 / F09). Every item is classified against the bundled,
// reviewed catalogue by exact numeric ID (or symbol when the journal gives no
// ID). EDDN's outfitting/2 and shipyard/2 schemas ask senders to omit items
// that depend on the commander's purchases or rank; the catalogue's
// entitlement column identifies those. An item the catalogue does not know is
// "unknown": callers must not publish a silently truncated inventory.

type inventoryEntry struct{ symbol, entitlement string }

var inventoryOnce sync.Once
var inventoryByID map[string]map[string]inventoryEntry
var inventoryBySymbol map[string]map[string]inventoryEntry

// eddnModuleSymbol is the outfitting/2 module pattern; other symbols
// (paint jobs, decals, bobbleheads, ...) are not station modules.
var eddnModuleSymbol = regexp.MustCompile(`(?i)(^hpt_|^int_|_armour_)`)

func loadInventory() {
	inventoryByID = map[string]map[string]inventoryEntry{}
	inventoryBySymbol = map[string]map[string]inventoryEntry{}
	for kind, file := range map[string]string{"outfitting": "outfitting.csv", "shipyard": "shipyard.csv"} {
		inventoryByID[kind] = map[string]inventoryEntry{}
		inventoryBySymbol[kind] = map[string]inventoryEntry{}
		b, _ := data.ReadFile(file)
		rows, err := csv.NewReader(strings.NewReader(string(b))).ReadAll()
		if err != nil || len(rows) == 0 {
			continue
		}
		cols := map[string]int{}
		for i, h := range rows[0] {
			cols[h] = i
		}
		for _, row := range rows[1:] {
			e := inventoryEntry{symbol: row[cols["symbol"]], entitlement: row[cols["entitlement"]]}
			inventoryByID[kind][row[cols["id"]]] = e
			inventoryBySymbol[kind][strings.ToLower(e.symbol)] = e
		}
	}
}

func permittedEntitlement(kind, entitlement, symbol string) bool {
	switch {
	case entitlement == "removed":
		return false
	case strings.HasPrefix(entitlement, "ELITE_SPECIFIC_V_POWER"): // PowerPlay rank modules
		return false
	case strings.HasPrefix(entitlement, "ELITE_HORIZONS_V_GUARDIAN"): // per-commander unlocks
		return false
	case strings.Contains(entitlement, "COBRA_MK_IV"): // pre-order entitlement
		return false
	}
	if kind == "outfitting" && strings.EqualFold(symbol, "Int_PlanetApproachSuite") {
		return false // fitted to every Horizons ship; not station stock
	}
	return true
}

// PublicInventorySymbol classifies one station inventory item of kind
// "outfitting" or "shipyard". id is the journal's numeric item ID ("" when
// absent); symbol is its name. canonical is the catalogue's symbol; known
// reports whether the catalogue classifies the item; permitted whether it may
// be published as ordinary public stock. A non-module outfitting symbol is
// known and not permitted (outside the recipient schema).
func PublicInventorySymbol(kind, id, symbol string) (canonical string, permitted, known bool) {
	inventoryOnce.Do(loadInventory)
	if kind == "outfitting" && !eddnModuleSymbol.MatchString(symbol) {
		return "", false, true
	}
	if id != "" {
		e, ok := inventoryByID[kind][id]
		if !ok || !strings.EqualFold(e.symbol, symbol) {
			return "", false, false
		}
		return e.symbol, permittedEntitlement(kind, e.entitlement, e.symbol), true
	}
	e, ok := inventoryBySymbol[kind][strings.ToLower(symbol)]
	if !ok {
		return "", false, false
	}
	return e.symbol, permittedEntitlement(kind, e.entitlement, e.symbol), true
}
