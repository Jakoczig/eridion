package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jakoczig/eridion/go/internal/catalog"
	"github.com/Jakoczig/eridion/go/internal/journal"
	"github.com/Jakoczig/eridion/go/internal/sidecar"
	"sort"
	"strings"
	"time"
)

type stationSession struct {
	Market, Name, System string
	Markers              map[string]journal.Event
	Files                map[string]sidecar.Change
}

// stationEventLocked consumes the payload Observe already decoded (with
// UseNumber), so exact MarketIDs are preserved without a second decode.
func (m *Manager) stationEventLocked(e journal.Event, phase journal.Phase, v map[string]any) {
	if e.Type == "Fileheader" || e.Type == "LoadGame" || e.Type == "Shutdown" || e.Type == "Undocked" || e.Type == "StartJump" {
		m.station = stationSession{Markers: map[string]journal.Event{}, Files: map[string]sidecar.Change{}}
		return
	}
	if e.Type == "Docked" && phase == journal.Live && m.identity != "" && validID(v["MarketID"]) && text(v["StationName"]) != "" && text(v["StarSystem"]) == m.system {
		n, _ := v["MarketID"].(json.Number)
		m.station = stationSession{Market: string(n), Name: text(v["StationName"]), System: m.system, Markers: map[string]journal.Event{}, Files: map[string]sidecar.Change{}}
	}
	if e.Type != "Market" && e.Type != "Shipyard" && e.Type != "Outfitting" {
		return
	}
	age := m.opts.Now().Sub(e.Timestamp)
	if phase != journal.Live || e.Timestamp.IsZero() || age < -5*time.Second || age > 2*time.Minute || m.station.Market == "" || m.station.Markers == nil {
		return
	}
	m.station.Markers[e.Type] = e
	m.publishStationLocked(e.Type)
}
func (m *Manager) ObserveSidecar(c sidecar.Change) {
	if (c.Kind != "Market" && c.Kind != "Shipyard" && c.Kind != "Outfitting") || len(c.Raw) > 1<<20 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.blocked || m.history == nil || m.station.Files == nil {
		return
	}
	if !c.Valid {
		delete(m.station.Files, c.Kind)
		return
	}
	m.station.Files[c.Kind] = c
	m.publishStationLocked(c.Kind)
}
func (m *Manager) publishStationLocked(kind string) {
	s := m.states["eddn"]
	c, ok := m.station.Files[kind]
	e, marked := m.station.Markers[kind]
	if !ok || !marked || !c.Valid || c.Timestamp.IsZero() || !c.Timestamp.Equal(e.Timestamp) || m.history == nil || m.identity == "" {
		return
	}
	age := m.opts.Now().Sub(c.Timestamp)
	if age < -5*time.Second || age > 2*time.Minute {
		return
	}
	v := object(c.Raw)
	n, valid := v["MarketID"].(json.Number)
	if !valid || string(n) != m.station.Market || text(v["StationName"]) != m.station.Name || text(v["StarSystem"]) != m.station.System {
		return
	}
	// Persist local evidence once, before any upload eligibility, so it is
	// retained even when publication consent is off. A failure stops here:
	// nothing is queued without its committed source evidence.
	if err := m.history.SaveSidecar(m.fid, e, kind, c.Raw); err != nil {
		if s != nil && s.config.Upload {
			s.lastError = "Station source persistence failed"
		}
		return
	}
	if !strings.HasPrefix(m.gameVersion, "4.") || strings.ContainsAny(strings.ToLower(m.gameVersion), "abcdefghijklmnopqrstuvwxyz") || s == nil || !s.config.Upload || s.config.Paused || s.config.CommanderHash != m.identity {
		return
	}
	cat := map[string]string{"Market": "commodities", "Shipyard": "shipyard", "Outfitting": "outfitting"}[kind]
	permitted := false
	for _, v := range s.config.Categories {
		if v == cat {
			permitted = true
		}
	}
	if !permitted {
		return
	}
	candidate, err := m.history.Candidate(m.fid, "eddn", cat, e, journal.Live, false, m.opts.Now())
	if err != nil || candidate.Status != "eligible" {
		return
	}
	payload, err := m.stationPayload(kind, v)
	if err != nil {
		s.lastError = "Station message lacked verified public fields; not queued"
		if strings.Contains(err.Error(), "snapshot not published") {
			s.lastError = err.Error() // honest suppression reason (Task 8 / F09)
		}
		return
	}
	key := hash("eddn/station/" + m.identity + "/" + string(c.Raw))
	if _, seen := m.ledger[key]; seen {
		return
	}
	for _, q := range s.queue {
		if q.key == key || q.candidateID == candidate.ID {
			return
		}
	}
	total := len(payload)
	for _, q := range s.queue {
		total += len(q.body)
	}
	if len(s.queue) >= 64 || total > 2<<20 {
		return
	}
	if err = m.history.CandidateOutcome(candidate.ID, "queued", "Fresh dock-matched sidecar authorized under exact recipient/category policy"); err != nil {
		return
	}
	s.queue = append(s.queue, queued{candidateID: candidate.ID, key: key, category: cat, body: payload, endpoint: m.cap("eddn").Endpoint, contentType: "application/json", generation: s.generation})
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (m *Manager) stationPayload(kind string, v map[string]any) ([]byte, error) {
	fail := func() ([]byte, error) { return nil, errors.New("Unsupported station snapshot") }
	if !validID(v["MarketID"]) || text(v["StationName"]) == "" || text(v["StarSystem"]) == "" {
		return fail()
	}
	msg := map[string]any{"systemName": v["StarSystem"], "stationName": v["StationName"], "marketId": v["MarketID"], "timestamp": v["timestamp"]}
	schema := ""
	items, _ := v["Items"].([]any)
	if len(items) > 2048 {
		return fail()
	}
	switch kind {
	case "Market":
		schema = "commodity/3"
		out := []any{}
		for _, item := range items {
			o, _ := item.(map[string]any)
			if illegal, _ := o["Illegal"].(bool); illegal {
				continue
			}
			if text(o["Name"]) == "" {
				return fail()
			}
			row := map[string]any{"name": o["Name"]}
			for key, target := range map[string]string{"MeanPrice": "meanPrice", "BuyPrice": "buyPrice", "Stock": "stock", "StockBracket": "stockBracket", "SellPrice": "sellPrice", "Demand": "demand", "DemandBracket": "demandBracket"} {
				n, ok := o[key].(json.Number)
				if !ok || !validID(n) {
					return fail()
				}
				row[target] = n
			}
			out = append(out, row)
		}
		if len(out) == 0 {
			return fail()
		}
		msg["commodities"] = out
	case "Shipyard", "Outfitting":
		// Task 8 / F09: every item is classified against the reviewed
		// catalogue; ordinary public stock is published in full, commander-
		// or rank-dependent items are omitted, and one unclassifiable item
		// suppresses the whole snapshot instead of publishing a truncated one.
		catalogKind, nameKey, listKey, field, limit := "outfitting", "Name", "Items", "modules", 2048
		schema = "outfitting/2"
		if kind == "Shipyard" {
			catalogKind, nameKey, listKey, field, limit = "shipyard", "ShipType", "PriceList", "ships", 256
			schema = "shipyard/2"
		}
		items, _ = v[listKey].([]any)
		if len(items) > limit {
			return fail()
		}
		out := []string{}
		seen := map[string]bool{}
		unknown, ambiguous := 0, 0
		unverified := map[string]bool{}
		for _, item := range items {
			o, _ := item.(map[string]any)
			name := text(o[nameKey])
			if name == "" {
				return fail()
			}
			id := ""
			if n, ok := o["id"].(json.Number); ok {
				id = n.String()
			}
			canonical, class, entitlement := catalog.ClassifyInventory(catalogKind, id, name)
			switch class {
			case catalog.InventoryPublic:
				if !seen[canonical] {
					out = append(out, canonical)
					seen[canonical] = true
				}
			case catalog.InventoryExcluded:
				// Commander-, rank- or unlock-dependent: omitted by design.
			case catalog.InventoryUnverified:
				unverified[entitlement] = true
			case catalog.InventoryAmbiguous:
				ambiguous++
			default:
				unknown++
			}
		}
		if unknown+ambiguous+len(unverified) > 0 {
			var parts []string
			if unknown > 0 {
				parts = append(parts, fmt.Sprintf("%d item(s) not classified by the bundled catalogue", unknown))
			}
			if ambiguous > 0 {
				parts = append(parts, fmt.Sprintf("%d item(s) with a shared symbol and no numeric ID", ambiguous))
			}
			if len(unverified) > 0 {
				names := []string{}
				for e := range unverified {
					names = append(names, e)
				}
				sort.Strings(names)
				parts = append(parts, "availability not verified for entitlement(s) "+strings.Join(names, ", "))
			}
			return nil, fmt.Errorf("%s snapshot not published: %s; a partial inventory is never sent", kind, strings.Join(parts, "; "))
		}
		if len(out) == 0 {
			return fail()
		}
		msg[field] = out
	default:
		return fail()
	}
	payload := map[string]any{"$schemaRef": "https://eddn.edcd.io/schemas/" + schema, "header": map[string]any{"uploaderID": m.uploader, "softwareName": software, "softwareVersion": version, "gameversion": m.gameVersion, "gamebuild": m.gameBuild}, "message": msg}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if len(raw) > 256<<10 {
		return fail()
	}
	if err = validateEDDNStation(schema, object(raw)); err != nil {
		return nil, err
	}
	return raw, nil
}
