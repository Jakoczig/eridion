package history

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func saveFleetFacts(t *testing.T, s *Store, fid string, fetched time.Time, fields map[string]any) {
	t.Helper()
	if e := s.SaveExternal(context.Background(), ExternalFacts{Provider: "frontier", Entity: "commander", EntityID: fid, FID: fid, IdentityVerified: true, Fetched: fetched, Expires: fetched.Add(5 * time.Minute), Fields: fields}); e != nil {
		t.Fatal(e)
	}
}

func fleetIDs(f ResolvedFleet) map[string]FleetShip {
	out := map[string]FleetShip{}
	for _, s := range f.Ships {
		out[s.ID] = s
	}
	return out
}

func resolvedFleet(t *testing.T, s *Store, fid string) ResolvedFleet {
	t.Helper()
	facts, e := s.EffectiveFacts(context.Background(), fid, "commander", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	f, e := s.ResolvedFleet(context.Background(), fid, facts)
	if e != nil {
		t.Fatal(e)
	}
	return f
}

// TestAuditFleetUnion: one resolved inventory unions journal current, stored,
// remote, in-transit and Frontier-only ships under the D0 order.
func TestAuditFleetUnion(t *testing.T) {
	s := operationFixture(t,
		`{"event":"Loadout","ShipID":1,"Ship":"asp","ShipName":"Journal name","Modules":[]}`,
		`{"event":"StoredShips","StarSystem":"Here","MarketID":5,"ShipsHere":[{"ShipID":2,"ShipType":"cobramkiii","Name":"Twin A"}],"ShipsRemote":[{"ShipID":3,"ShipType":"cobramkiii","Name":"Twin B","StarSystem":"Far"},{"ShipID":4,"ShipType":"python","InTransit":true}]}`,
		`{"event":"ShipyardSell","SellShipID":6,"ShipType":"sidewinder"}`,
		`{"event":"Loadout","ShipID":7,"Ship":"adder_taxi","Modules":[]}`,
		`{"event":"Loadout","ShipID":1,"Ship":"asp","ShipName":"Journal name","Modules":[]}`,
	)
	// Journal only: partial coverage, taxi never counted, sold ship excluded.
	f := resolvedFleet(t, s, "SYNTHETIC")
	ids := fleetIDs(f)
	if f.Coverage != "partial" || f.Owned != 4 || ids["7"].ID != "" || ids["6"].State != "sold" {
		t.Fatalf("journal-only fleet = coverage %s owned %d ships %+v", f.Coverage, f.Owned, f.Ships)
	}
	if ids["2"].ID == "" || ids["3"].ID == "" || ids["2"].Model != ids["3"].Model {
		t.Fatal("identical models with distinct IDs merged")
	}
	if ids["4"].State != "in_transit" {
		t.Fatalf("in-transit ship = %+v", ids["4"])
	}
	if !f.Ships[0].Current || f.Ships[0].ID != "1" {
		t.Fatalf("current ship not first: %+v", f.Ships[0])
	}

	// Fresh Frontier: complete inventory with a Frontier-only ship, a re-owned
	// sold ID and a renamed current ship; ship 3 is absent from the snapshot.
	now := time.Now()
	saveFleetFacts(t, s, "SYNTHETIC", now, map[string]any{
		"ships": []any{
			map[string]any{"id": "1", "model": "asp", "name": "Frontier name", "currentReference": true},
			map[string]any{"id": "2", "model": "cobramkiii", "name": "Twin A"},
			map[string]any{"id": "4", "model": "python"},
			map[string]any{"id": "6", "model": "sidewinder", "name": "Bought back"},
			map[string]any{"id": "9007199254740993", "model": "krait_mkii", "name": "Frontier only", "systemName": "Remote"},
		},
		"ship": map[string]any{"id": "1", "model": "asp", "name": "Frontier name", "currentReference": true},
	})
	f = resolvedFleet(t, s, "SYNTHETIC")
	ids = fleetIDs(f)
	if f.Coverage != "complete" {
		t.Fatalf("coverage = %s", f.Coverage)
	}
	if ids["1"].Name != "Frontier name" || ids["1"].Sources["name"] != "frontier" {
		t.Fatalf("fresh Frontier name lost: %+v", ids["1"])
	}
	if ids["6"].State == "sold" || ids["6"].Name != "Bought back" {
		t.Fatalf("fresh Frontier ownership of a reused ID ignored: %+v", ids["6"])
	}
	if ids["9007199254740993"].ID == "" || ids["9007199254740993"].Name != "Frontier only" {
		t.Fatal("Frontier-only ship missing or ID precision lost")
	}
	if ids["3"].State != "not_currently_reported" {
		t.Fatalf("absent ship = %+v, want not_currently_reported (no sale)", ids["3"])
	}
	if ids["4"].State != "in_transit" {
		t.Fatal("Frontier snapshot erased the witnessed transfer")
	}
	if f.Owned != 5 {
		t.Fatalf("owned = %d, want 5 (1,2,4,6,frontier-only)", f.Owned)
	}
	count, e := s.ReconciledFleetCount(context.Background(), "SYNTHETIC", nil)
	if e != nil || count != f.Owned {
		t.Fatalf("Commander count %d != resolved owned %d (%v)", count, f.Owned, e)
	}

	// Another commander never sees this inventory.
	if other := resolvedFleet(t, s, "OTHER"); len(other.Ships) != 0 || other.Coverage != "unknown" {
		t.Fatalf("fleet leaked across commanders: %+v", other)
	}
}

func TestAuditFleetFrontierFallbacks(t *testing.T) {
	journal := `{"event":"Loadout","ShipID":1,"Ship":"asp","ShipName":"Journal name","Modules":[]}`
	stored := `{"event":"StoredShips","StarSystem":"Here","MarketID":5,"ShipsHere":[{"ShipID":2,"ShipType":"cobramkiii","Name":"Stored"}],"ShipsRemote":[]}`

	// Expired Frontier: journal values win; Frontier-only ships stay listed as
	// last-known references, and nothing is marked absent.
	s := operationFixture(t, journal, stored)
	saveFleetFacts(t, s, "SYNTHETIC", time.Now().Add(-time.Hour), map[string]any{"ships": []any{map[string]any{"id": "1", "model": "asp", "name": "Old CAPI"}, map[string]any{"id": "8", "model": "eagle"}}})
	f := resolvedFleet(t, s, "SYNTHETIC")
	ids := fleetIDs(f)
	if ids["1"].Name != "Journal name" || ids["2"].State == "not_currently_reported" || ids["8"].ID == "" || f.Coverage != "partial" {
		t.Fatalf("stale fallback = %+v coverage %s", f.Ships, f.Coverage)
	}

	// Current-only profile (no ships collection) still includes the current
	// ship and reports incomplete coverage; stored ships are not removed.
	s = operationFixture(t, stored)
	saveFleetFacts(t, s, "SYNTHETIC", time.Now(), map[string]any{"ship": map[string]any{"id": "11", "model": "mamba", "currentReference": true}})
	f = resolvedFleet(t, s, "SYNTHETIC")
	ids = fleetIDs(f)
	if ids["11"].ID == "" || !ids["11"].Current || ids["2"].ID == "" || f.Coverage != "partial" || f.Owned != 2 {
		t.Fatalf("current-only profile = %+v coverage %s owned %d", f.Ships, f.Coverage, f.Owned)
	}

	// An explicit empty collection is a complete snapshot: known ships are
	// kept but marked not currently reported, never deleted or sold.
	s = operationFixture(t, stored)
	saveFleetFacts(t, s, "SYNTHETIC", time.Now(), map[string]any{"ships": []any{}})
	f = resolvedFleet(t, s, "SYNTHETIC")
	if ids := fleetIDs(f); ids["2"].State != "not_currently_reported" || f.Owned != 0 || len(f.Ships) != 1 {
		t.Fatalf("explicit empty collection = %+v owned %d", f.Ships, f.Owned)
	}
}

// Global search and pagination cover the whole resolved inventory.
func TestAuditFleetSearchAndPaging(t *testing.T) {
	for _, total := range []int{0, 1, 30, 31, 75} {
		remote := []string{}
		for i := 2; i <= total; i++ {
			remote = append(remote, fmt.Sprintf(`{"ShipID":%d,"ShipType":"cobramkiii","Name":"Ship %03d","StarSystem":"Far"}`, i, i))
		}
		lines := []string{}
		if total > 0 {
			lines = append(lines, `{"event":"Loadout","ShipID":1,"Ship":"anaconda","ShipName":"Flagship","Modules":[]}`)
		}
		lines = append(lines, `{"event":"StoredShips","StarSystem":"Here","MarketID":5,"ShipsHere":[],"ShipsRemote":[`+strings.Join(remote, ",")+`]}`)
		s := operationFixture(t, lines...)
		seen := map[string]bool{}
		q := OperationQuery{Kind: "fleet", FleetState: "owned", Limit: 30}
		var revision string
		for pages := 0; pages < 10; pages++ {
			page, e := s.Operations(context.Background(), "SYNTHETIC", q)
			if e != nil {
				t.Fatal(e)
			}
			if page.FleetTotal != total || page.FleetMatched != total {
				t.Fatalf("total=%d: page totals %d/%d", total, page.FleetTotal, page.FleetMatched)
			}
			revision = page.FleetRevision
			for _, r := range page.Rows {
				if seen[r.Entity] {
					t.Fatalf("total=%d: duplicate %s", total, r.Entity)
				}
				seen[r.Entity] = true
			}
			if page.FleetNext == 0 {
				break
			}
			q.FleetAfter, q.FleetRevision = page.FleetNext, revision
		}
		if len(seen) != total {
			t.Fatalf("total=%d: traversed %d ships", total, len(seen))
		}
		if total == 75 {
			// A later-page ship is found by global search.
			page, e := s.Operations(context.Background(), "SYNTHETIC", OperationQuery{Kind: "fleet", FleetState: "owned", Limit: 30, Commodity: "ship 074"})
			if e != nil || len(page.Rows) != 1 || page.Rows[0].Entity != "74" || page.FleetTotal != 75 || page.FleetMatched != 1 {
				t.Fatalf("global search = %+v %v", page.Rows, e)
			}
			// A changed inventory restarts a stale continuation instead of
			// silently shifting positions.
			saveFleetFacts(t, s, "SYNTHETIC", time.Now().Add(-time.Hour), map[string]any{"ships": []any{map[string]any{"id": "900", "model": "eagle"}}})
			page, e = s.Operations(context.Background(), "SYNTHETIC", OperationQuery{Kind: "fleet", FleetState: "owned", Limit: 30, FleetAfter: 30, FleetRevision: revision})
			if e != nil || !page.FleetRestarted || len(page.Rows) == 0 || page.Rows[0].Entity != "1" {
				t.Fatalf("revision change did not restart: %+v %v", page, e)
			}
			var d map[string]any
			_ = json.Unmarshal(page.Rows[0].Details, &d)
			if d["shipName"] != "Flagship" {
				t.Fatalf("row details = %v", d)
			}
		}
	}
}
