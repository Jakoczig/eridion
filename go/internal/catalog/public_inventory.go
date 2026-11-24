package catalog

import (
	"encoding/csv"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Public station inventory classification for EDDN shipyard/outfitting
// publication (Tasks 8 and 16). Items are classified by exact numeric ID (or
// symbol when the journal gives no ID). The entitlement decides availability
// through the reviewed table entitlement-review.csv: "ordinary" stock is
// public, "excluded" items depend on the commander's purchases, unlocks or
// rank (the EDDN schemas ask senders to omit them), and any entitlement
// without a review is unverified, which callers must treat like an unknown
// item: never publish a snapshot that silently drops it.

// Inventory classifications.
const (
	InventoryPublic     = "public"
	InventoryExcluded   = "excluded"
	InventoryUnverified = "unverified" // entitlement availability not reviewed
	InventoryUnknown    = "unknown"    // not in the catalogue, or ID/symbol mismatch
	InventoryAmbiguous  = "ambiguous"  // shared symbol without a numeric ID
)

type inventoryEntry struct{ id, symbol, entitlement string }
type entitlementRule struct{ pattern, class string }

var inventoryOnce sync.Once
var inventoryByID map[string]map[string]inventoryEntry
var inventoryBySymbol map[string]map[string][]inventoryEntry
var entitlementRules []entitlementRule

// idReview holds per-ID decisions from inventory-id-review.csv; IDs above
// idWatermark (added by the G3I refresh) with a generic entitlement need one.
var idReview map[string]string
var idWatermark uint64

// genericEntitlement marks entitlement values that say nothing about how an
// individual item is unlocked: the empty value and the Horizons SKU.
func genericEntitlement(e string) bool {
	return e == "" || e == "ELITE_HORIZONS_V_PLANETARY_LANDINGS"
}

// eddnModuleSymbol is the outfitting/2 module pattern; other symbols
// (paint jobs, decals, bobbleheads, ...) are not station modules.
var eddnModuleSymbol = regexp.MustCompile(`(?i)(^hpt_|^int_|_armour_)`)

func loadInventory() {
	inventoryByID = map[string]map[string]inventoryEntry{}
	inventoryBySymbol = map[string]map[string][]inventoryEntry{}
	for kind, file := range map[string]string{"outfitting": "outfitting.csv", "shipyard": "shipyard.csv"} {
		inventoryByID[kind] = map[string]inventoryEntry{}
		inventoryBySymbol[kind] = map[string][]inventoryEntry{}
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
			e := inventoryEntry{id: row[cols["id"]], symbol: row[cols["symbol"]], entitlement: row[cols["entitlement"]]}
			inventoryByID[kind][row[cols["id"]]] = e
			k := strings.ToLower(e.symbol)
			inventoryBySymbol[kind][k] = append(inventoryBySymbol[kind][k], e)
		}
	}
	b, _ := data.ReadFile("entitlement-review.csv")
	rows, _ := csv.NewReader(strings.NewReader(string(b))).ReadAll()
	for i, row := range rows {
		if i == 0 || len(row) < 2 {
			continue
		}
		entitlementRules = append(entitlementRules, entitlementRule{row[0], row[1]})
	}
	idReview = map[string]string{}
	b, _ = data.ReadFile("inventory-id-review.csv")
	rows, _ = csv.NewReader(strings.NewReader(string(b))).ReadAll()
	for i, row := range rows {
		if i == 0 || len(row) < 2 {
			continue
		}
		if row[0] == "watermark" {
			idWatermark, _ = strconv.ParseUint(row[1], 10, 64)
			continue
		}
		idReview[row[0]] = row[1]
	}
}

// entitlementClass applies the reviewed table; exact rows win over prefix rows.
func entitlementClass(entitlement string) string {
	for _, r := range entitlementRules {
		if r.pattern == entitlement {
			return r.class
		}
	}
	for _, r := range entitlementRules {
		if strings.HasSuffix(r.pattern, "*") && strings.HasPrefix(entitlement, strings.TrimSuffix(r.pattern, "*")) {
			return r.class
		}
	}
	return "unverified"
}

func classify(kind string, e inventoryEntry) string {
	if kind == "outfitting" && strings.EqualFold(e.symbol, "Int_PlanetApproachSuite") {
		return InventoryExcluded // fitted to every Horizons ship; not station stock
	}
	class := entitlementClass(e.entitlement)
	if c, ok := idReview[e.id]; ok {
		class = c
	} else if n, err := strconv.ParseUint(e.id, 10, 64); err == nil && n > idWatermark && genericEntitlement(e.entitlement) {
		class = "unverified" // added after the reviewed catalogue; needs a per-ID review
	}
	switch class {
	case "ordinary":
		return InventoryPublic
	case "excluded":
		return InventoryExcluded
	}
	return InventoryUnverified
}

// ClassifyInventory classifies one station inventory item of kind
// "outfitting" or "shipyard". id is the journal's numeric item ID ("" when
// absent). canonical is the catalogue symbol; entitlement is returned for
// accurate suppression reasons. A non-module outfitting symbol is excluded
// (outside the recipient schema).
func ClassifyInventory(kind, id, symbol string) (canonical, class, entitlement string) {
	inventoryOnce.Do(loadInventory)
	if kind == "outfitting" && !eddnModuleSymbol.MatchString(symbol) {
		return "", InventoryExcluded, ""
	}
	if id != "" {
		e, ok := inventoryByID[kind][id]
		if !ok || !strings.EqualFold(e.symbol, symbol) {
			return "", InventoryUnknown, ""
		}
		return e.symbol, classify(kind, e), e.entitlement
	}
	es := inventoryBySymbol[kind][strings.ToLower(symbol)]
	switch {
	case len(es) == 0:
		return "", InventoryUnknown, ""
	case len(es) > 1:
		// Variants of one symbol can differ in availability; without the
		// numeric ID the item cannot be classified safely.
		classes := map[string]bool{}
		for _, e := range es {
			classes[classify(kind, e)] = true
		}
		if len(classes) > 1 {
			return es[0].symbol, InventoryAmbiguous, ""
		}
		return es[0].symbol, classify(kind, es[0]), es[0].entitlement
	}
	return es[0].symbol, classify(kind, es[0]), es[0].entitlement
}

// PublicInventorySymbol is the G3H interface: permitted means public; known
// means classified as public or excluded (unverified, ambiguous and unknown
// items are not known and must suppress the snapshot).
func PublicInventorySymbol(kind, id, symbol string) (canonical string, permitted, known bool) {
	canonical, class, _ := ClassifyInventory(kind, id, symbol)
	switch class {
	case InventoryPublic:
		return canonical, true, true
	case InventoryExcluded:
		return canonical, false, true
	}
	return "", false, false
}
