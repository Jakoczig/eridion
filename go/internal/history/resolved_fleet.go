package history

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Jakoczig/eridion/go/internal/catalog"
)

// FleetShip is one owned (or historically owned) main ship in the resolved
// inventory. Exact IDs stay decimal text; Sources names the source of each
// resolved field.
type FleetShip struct {
	ID         string            `json:"id"`
	Model      string            `json:"model"`
	Name       string            `json:"name"`
	System     string            `json:"system"`
	Station    string            `json:"station"`
	MarketID   string            `json:"marketId"`
	State      string            `json:"state"`
	ObservedAt string            `json:"observedAt"`
	Current    bool              `json:"current"`
	Sources    map[string]string `json:"sources"`
	details    map[string]any
	order      int64
}

// ResolvedFleet is the single fleet inventory used for rows, search, paging,
// the current ship and the Commander count. Coverage is "complete" only for a
// fresh validated Frontier ships collection (with its current ship),
// "partial" for any other evidence and "unknown" without evidence.
type ResolvedFleet struct {
	Ships           []FleetShip `json:"ships"`
	Owned           int         `json:"owned"`
	Coverage        string      `json:"coverage"`
	Revision        string      `json:"revision"`
	FrontierFetched string      `json:"frontierFetchedAt"`
	FrontierStale   bool        `json:"frontierStale"`
}

const maxFleetRows = 4096

func ownedState(state string) bool { return state != "sold" && state != "not_currently_reported" }

func frontierEvidence(facts []EffectiveFact, field string) *FactEvidence {
	candidates := factCandidates(facts, field)
	for i := range candidates {
		if candidates[i].Source == "frontier" {
			return &candidates[i]
		}
	}
	return nil
}

// ResolvedFleet unions journal fleet history with Frontier's ships collection
// under D0. Fresh Frontier wins for the fields it reports and can establish
// current ownership (including a previously sold, reused ID, whose sale stays
// in fleet history). A fresh complete collection marks absent ships as not
// currently reported without inventing a sale. Stale Frontier only fills
// gaps. Station sale stock is never part of the inventory.
func (s *Store) ResolvedFleet(ctx context.Context, fid string, facts []EffectiveFact) (ResolvedFleet, error) {
	out := ResolvedFleet{Ships: []FleetShip{}, Coverage: "unknown"}
	if fid == "" || len(fid) > 128 {
		return out, nil
	}
	if facts == nil {
		var e error
		if facts, e = s.EffectiveFacts(ctx, fid, "commander", time.Now()); e != nil {
			return out, e
		}
	}
	byID := map[string]*FleetShip{}
	ordered := []*FleetShip{}
	rows, e := s.reader.QueryContext(ctx, "SELECT rowid,ship_id,model,name,state,coalesce(system_name,''),coalesce(market_id,''),coalesce(observed_at,''),coalesce(metadata_json,'{}') FROM fleet_ships WHERE commander_fid=? ORDER BY rowid DESC LIMIT ?", fid, maxFleetRows)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var x FleetShip
		var raw string
		if e = rows.Scan(&x.order, &x.ID, &x.Model, &x.Name, &x.State, &x.System, &x.MarketID, &x.ObservedAt, &raw); e != nil {
			rows.Close()
			return out, e
		}
		if identifier(x.ID) == "" || catalog.VehicleKind(x.Model) != "ship" && x.Model != "" {
			continue
		}
		x.details = object([]byte(raw))
		if x.details == nil {
			x.details = map[string]any{}
		}
		x.Sources = map[string]string{"ownership": "journal"}
		x.Current = x.State == "known_active"
		ship := x
		byID[x.ID] = &ship
		ordered = append(ordered, &ship)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}

	shipsEv, currentEv := frontierEvidence(facts, "ships"), frontierEvidence(facts, "ship")
	var capi []map[string]any
	collection := false
	if shipsEv != nil && json.Unmarshal(shipsEv.Value, &capi) == nil && len(capi) <= 512 {
		collection = true
		out.FrontierFetched, out.FrontierStale = shipsEv.Fetched, !shipsEv.Eligible
	} else {
		capi = nil
	}
	fresh := collection && shipsEv.Eligible
	var current map[string]any
	if currentEv != nil && currentEv.Eligible {
		if json.Unmarshal(currentEv.Value, &current) == nil && current["currentReference"] == true && identifier(text(current["id"])) != "" {
			if out.FrontierFetched == "" {
				out.FrontierFetched = currentEv.Fetched
			}
		} else {
			current = nil
		}
	}
	reported := map[string]bool{}
	apply := func(ship map[string]any, ev *FactEvidence, eligible bool) {
		id := text(ship["id"])
		if identifier(id) == "" || catalog.VehicleKind(text(ship["model"])) != "ship" && text(ship["model"]) != "" {
			return
		}
		reported[id] = true
		reference := map[string]any{"source": "frontier/profile.ships", "fetchedAt": ev.Fetched, "observedAt": ev.Observed, "stale": !eligible, "loadoutComplete": false, "currentReference": ship["currentReference"]}
		x := byID[id]
		if x == nil {
			x = &FleetShip{ID: id, State: "capi_last_known", Sources: map[string]string{"ownership": "frontier"}, details: map[string]any{}, order: -1}
			byID[id] = x
			ordered = append(ordered, x)
		}
		set := func(field string, target *string, value string) {
			if value == "" {
				return
			}
			if eligible || *target == "" {
				*target = value
				x.Sources[field] = "frontier"
			}
		}
		set("name", &x.Name, text(ship["name"]))
		set("model", &x.Model, text(ship["model"]))
		// CAPI has no transfer concept: a witnessed in-transit state keeps its
		// unknown destination instead of an old CAPI location.
		if x.State != "in_transit" {
			set("system", &x.System, text(ship["systemName"]))
			set("station", &x.Station, text(ship["station"]))
		}
		if eligible && x.State == "sold" {
			x.State = "capi_last_known"
			x.Sources["ownership"] = "frontier"
		}
		if eligible && ship["currentReference"] == true {
			for _, other := range ordered {
				other.Current = false
			}
			x.Current = true
		}
		for _, key := range []string{"value", "hullValue", "modulesValue"} {
			target := map[string]string{"value": "Value", "hullValue": "HullValue", "modulesValue": "ModulesValue"}[key]
			if v := text(ship[key]); v != "" && (eligible || x.details[target] == nil || x.details[target] == "") {
				x.details[target] = v
			}
		}
		if modules, ok := ship["modules"].([]any); ok && len(modules) > 0 {
			x.details["capiModules"] = ship["modules"]
		}
		x.details["capiReference"] = reference
	}
	for _, ship := range capi {
		apply(ship, shipsEv, fresh)
	}
	// A fresh collection is complete only if it also contains Frontier's
	// verified current ship; a current-only profile is never complete.
	currentListed := current == nil || reported[text(current["id"])]
	if current != nil && !reported[text(current["id"])] {
		apply(current, currentEv, true)
	}
	switch {
	case fresh && currentListed:
		out.Coverage = "complete"
	case len(ordered) > 0 || collection || current != nil:
		out.Coverage = "partial"
	}
	if out.Coverage == "complete" {
		for _, x := range ordered {
			if !reported[x.ID] && ownedState(x.State) {
				x.State = "not_currently_reported"
				x.Current = false
			}
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.Current != b.Current {
			return a.Current
		}
		if (a.order >= 0) != (b.order >= 0) {
			return a.order >= 0
		}
		return a.order > b.order
	})
	sum := sha256.New()
	for _, x := range ordered {
		fmt.Fprintf(sum, "%s\x00%s\x00%s\x00%s\x00%s\x00%v\n", x.ID, x.State, x.Name, x.Model, x.System, x.Current)
		if ownedState(x.State) {
			out.Owned++
		}
		out.Ships = append(out.Ships, *x)
	}
	out.Revision = fmt.Sprintf("%x", sum.Sum(nil))[:16]
	return out, nil
}

func (x FleetShip) matches(search string) bool {
	if search == "" {
		return true
	}
	hay := strings.ToLower(strings.Join([]string{x.ID, x.Name, x.Model, catalog.ShipDisplayName(x.Model), x.System, x.Station}, " "))
	return strings.Contains(hay, strings.ToLower(search))
}

func (x FleetShip) row() OperationRow {
	data := map[string]any{}
	for k, v := range x.details {
		data[k] = v
	}
	data["shipName"], data["model"], data["system"], data["marketId"] = x.Name, x.Model, x.System, x.MarketID
	if x.Station != "" {
		data["station"] = x.Station
	}
	data["current"], data["sources"] = x.Current, x.Sources
	return OperationRow{ID: "ship:" + x.ID, Entity: x.ID, Kind: "Fleet", At: x.ObservedAt, State: x.State, Title: strings.TrimSpace(x.Model + " " + x.Name), Details: json.RawMessage(encoded(data))}
}

// fleetPage serves bounded search/paging over the entire resolved inventory.
// FleetAfter is an offset that is only valid for the revision it came from;
// a changed inventory restarts from the first page.
func (s *Store) fleetPage(ctx context.Context, fid string, q OperationQuery, out OperationPage) (OperationPage, error) {
	facts, e := s.EffectiveFacts(ctx, fid, "commander", time.Now())
	if e != nil {
		return out, e
	}
	out.Reference = facts
	fleet, e := s.ResolvedFleet(ctx, fid, facts)
	if e != nil {
		return out, e
	}
	out.FleetTotal, out.FleetCoverage, out.FleetRevision = fleet.Owned, fleet.Coverage, fleet.Revision
	offset := q.FleetAfter
	if q.FleetRevision != "" && q.FleetRevision != fleet.Revision {
		offset, out.FleetRestarted = 0, true
	}
	matched := []FleetShip{}
	for _, x := range fleet.Ships {
		keep := q.FleetState == "" || (q.FleetState == "owned" && ownedState(x.State)) || x.State == q.FleetState
		if keep && x.matches(q.Commodity) {
			matched = append(matched, x)
		}
	}
	out.FleetMatched = len(matched)
	bytes := 0
	for i := offset; i < len(matched); i++ {
		r := matched[i].row()
		if len(out.Rows) >= q.Limit || bytes+len(r.Details) > 256<<10 {
			out.FleetNext = i
			break
		}
		bytes += len(r.Details)
		out.Rows = append(out.Rows, r)
	}
	out.Notice = "One resolved fleet: fresh Frontier first, then journal, without inventing sales. Coverage " + fleet.Coverage + "."
	return out, nil
}
