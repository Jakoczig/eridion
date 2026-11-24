package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/Jakoczig/eridion/go/internal/history"
	"github.com/Jakoczig/eridion/go/internal/journal"
	"github.com/Jakoczig/eridion/go/internal/sidecar"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Restricted (PowerPlay) and commander-dependent items are omitted; unknown
// items now suppress the snapshot (TestAuditStationInventoryComplete).
func TestEDDNStationSchemasSanitizeAndPreserveExactMarketID(t *testing.T) {
	m := &Manager{uploader: "fixture", gameVersion: "4.1", gameBuild: "fixture"}
	for kind, items := range map[string]string{"Market": `"Items":[{"Name":"gold","MeanPrice":1,"BuyPrice":2,"Stock":3,"StockBracket":1,"SellPrice":4,"Demand":5,"DemandBracket":1,"Producer":true,"Rare":false,"id":123}]`, "Shipyard": `"PriceList":[{"ShipType":"SideWinder"},{"ShipType":"CobraMkIV"}]`, "Outfitting": `"Items":[{"Name":"int_hyperdrive_size5_class5"},{"Name":"int_shieldgenerator_size3_class5_strong"},{"Name":"PRIVATE_DECAL"}]`} {
		raw := `{"timestamp":"2026-11-15T12:00:00Z","StarSystem":"Fixture","StationName":"Fixture Port","MarketID":18446744073709551615,"Commander":"PRIVATE",` + items + `}`
		out, err := m.stationPayload(kind, object([]byte(raw)))
		if err != nil {
			t.Fatal(kind, err)
		}
		if !strings.Contains(string(out), "18446744073709551615") || strings.Contains(string(out), "PRIVATE") || strings.Contains(string(out), "Producer") || strings.Contains(string(out), "CobraMkIV") || strings.Contains(string(out), "_Strong") {
			t.Fatal("station fields escaped", string(out))
		}
	}
}
func TestEDDNStationRequiresCommittedMarkerDockAndExplicitLiveException(t *testing.T) {
	db, err := history.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	m := New(context.Background(), t.TempDir(), Options{RequireHistory: true, Manual: true, Vault: &testVault{}, Now: func() time.Time { return now }})
	defer m.Close()
	if err = m.AttachHistory(db, history.ProviderSeed{}); err != nil {
		t.Fatal(err)
	}
	src := journal.Source{ID: "station-fixture", Identity: "synthetic"}
	cp, _ := db.Resume(src)
	raws := []string{`{"event":"Fileheader","gameversion":"4.1","build":"fixture"}`, `{"event":"LoadGame","FID":"SYNTHETIC-A","Commander":"Fixture"}`, `{"event":"Location","StarSystem":"Fixture","SystemAddress":9007199254740993,"StarPos":[1,2,3]}`, `{"event":"Docked","StarSystem":"Fixture","StationName":"Fixture Port","MarketID":18446744073709551615}`, `{"event":"Market","timestamp":"2026-11-15T12:00:00Z","MarketID":18446744073709551615}`}
	events := []journal.Record{}
	for i, raw := range raws {
		e, err := journal.ParseLine([]byte(raw), journal.SourcePosition{FileID: src.ID, Sequence: uint64(i + 1), Offset: int64(i * 100), Length: 100})
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, journal.Record{Event: e, Phase: journal.Live})
	}
	cp.Offset = 500
	cp.Sequence = 5
	if err = db.Commit(context.Background(), cp, events, nil); err != nil {
		t.Fatal(err)
	}
	for _, e := range events[:4] {
		m.Observe(e.Event, e.Phase)
	}
	if err = m.Configure("eddn", Config{Upload: true, Categories: []string{"shipyard"}}); err != nil {
		t.Fatal(err)
	}
	// This sidecar is deliberately wrong kind: no Shipyard journal marker.
	c := sidecar.Change{Kind: "Shipyard", Valid: true, Timestamp: now, Raw: json.RawMessage(`{"event":"Shipyard","timestamp":"2026-11-15T12:00:00Z","StarSystem":"Fixture","StationName":"Fixture Port","MarketID":18446744073709551615,"PriceList":[{"ShipType":"SideWinder"}]}`)}
	if err = m.SetPublicationPolicy("SYNTHETIC-A", "After sale"); err != nil {
		t.Fatal(err)
	}
	m.ObserveSidecar(c)
	if len(m.states["eddn"].queue) != 0 {
		t.Fatal("sidecar without matching journal marker queued")
	}
	// Committed Market marker still cannot authorize Shipyard data.
	m.Observe(events[4].Event, journal.Live)
	m.ObserveSidecar(c)
	if len(m.states["eddn"].queue) != 0 {
		t.Fatal("wrong marker kind queued")
	}
	// Publish a matching committed Shipyard marker, then prove the privacy hold.
	e, _ := journal.ParseLine([]byte(`{"event":"Shipyard","timestamp":"2026-11-15T12:00:00Z","MarketID":18446744073709551615}`), journal.SourcePosition{FileID: src.ID, Sequence: 6, Offset: 500, Length: 100})
	cp.Offset = 600
	cp.Sequence = 6
	if err = db.Commit(context.Background(), cp, []journal.Record{{Event: e, Phase: journal.Live}}, nil); err != nil {
		t.Fatal(err)
	}
	m.Observe(e, journal.Live)
	if len(m.states["eddn"].queue) != 0 {
		t.Fatal("After sale station disclosure leaked")
	}
	if err = m.SetLiveException("SYNTHETIC-A", "eddn", "shipyard", true, true); err != nil {
		t.Fatal(err)
	}
	m.ObserveSidecar(c)
	if len(m.states["eddn"].queue) != 1 {
		t.Fatalf("exact live exception did not authorize station payload: %s", m.states["eddn"].lastError)
	}
	if err = m.SetLiveException("SYNTHETIC-A", "eddn", "shipyard", false, false); err != nil {
		t.Fatal(err)
	}
	if len(m.states["eddn"].queue) != 0 {
		t.Fatal("revoked exception left unsent disclosure")
	}
}

// One accepted publication attempt saves its station evidence once; failed
// source persistence queues nothing, and consent-off evidence is retained.
func TestPerfStationSingleDecodeAndSave(t *testing.T) {
	for _, tc := range []struct {
		name          string
		consent, fail bool
		wantQueue     int
		wantAttempts  int
	}{{"authorized", true, false, 1, 1}, {"consent off keeps evidence", false, false, 0, 1}, {"failed persistence", true, true, 0, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			db, err := history.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
			m := New(context.Background(), t.TempDir(), Options{RequireHistory: true, Manual: true, Vault: &testVault{}, Now: func() time.Time { return now }})
			defer m.Close()
			if err = m.AttachHistory(db, history.ProviderSeed{}); err != nil {
				t.Fatal(err)
			}
			src := journal.Source{ID: "station-perf", Identity: "synthetic"}
			cp, _ := db.Resume(src)
			raws := []string{`{"event":"Fileheader","gameversion":"4.1","build":"fixture"}`, `{"event":"LoadGame","FID":"SYNTHETIC-A","Commander":"Fixture"}`, `{"event":"Location","StarSystem":"Fixture","SystemAddress":9007199254740993,"StarPos":[1,2,3]}`, `{"event":"Docked","StarSystem":"Fixture","StationName":"Fixture Port","MarketID":18446744073709551615}`, `{"event":"Shipyard","timestamp":"2026-11-15T12:00:00Z","MarketID":18446744073709551615}`}
			var records []journal.Record
			for i, raw := range raws {
				e, err := journal.ParseLine([]byte(raw), journal.SourcePosition{FileID: src.ID, Sequence: uint64(i + 1), Offset: int64(i * 100), Length: 100})
				if err != nil {
					t.Fatal(err)
				}
				records = append(records, journal.Record{Event: e, Phase: journal.Live})
			}
			cp.Offset, cp.Sequence = 500, 5
			if err = db.Commit(context.Background(), cp, records, nil); err != nil {
				t.Fatal(err)
			}
			for _, r := range records[:4] {
				m.Observe(r.Event, r.Phase)
			}
			if tc.consent {
				if err = m.Configure("eddn", Config{Upload: true, Categories: []string{"shipyard"}}); err != nil {
					t.Fatal(err)
				}
				if err = m.SetPublicationPolicy("SYNTHETIC-A", "After sale"); err != nil {
					t.Fatal(err)
				}
				if err = m.SetLiveException("SYNTHETIC-A", "eddn", "shipyard", true, true); err != nil {
					t.Fatal(err)
				}
			}
			raw, err := sql.Open("sqlite3", filepath.Join(dir, "daedalus.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			defer raw.Close()
			action := "UPDATE perf_attempts SET n=n+1"
			if tc.fail {
				action = "SELECT RAISE(FAIL,'injected persistence failure')"
			}
			if _, err = raw.Exec("CREATE TABLE perf_attempts(n INTEGER); INSERT INTO perf_attempts VALUES(0); CREATE TRIGGER perf_count BEFORE INSERT ON sidecar_snapshots BEGIN " + action + "; END"); err != nil {
				t.Fatal(err)
			}
			m.Observe(records[4].Event, journal.Live)
			m.ObserveSidecar(sidecar.Change{Kind: "Shipyard", Valid: true, Timestamp: now, Raw: json.RawMessage(`{"event":"Shipyard","timestamp":"2026-11-15T12:00:00Z","StarSystem":"Fixture","StationName":"Fixture Port","MarketID":18446744073709551615,"PriceList":[{"ShipType":"SideWinder"}]}`)})
			if got := len(m.states["eddn"].queue); got != tc.wantQueue {
				t.Fatalf("queue = %d, want %d (%s)", got, tc.wantQueue, m.states["eddn"].lastError)
			}
			var attempts, stored int
			if err = raw.QueryRow("SELECT n FROM perf_attempts").Scan(&attempts); err != nil {
				t.Fatal(err)
			}
			if err = raw.QueryRow("SELECT count(*) FROM sidecar_snapshots WHERE kind='Shipyard' AND instr(raw_json,'18446744073709551615')>0").Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if tc.fail {
				// The failing attempt rolls back with its transaction.
				if stored != 0 {
					t.Fatal("failed persistence left evidence")
				}
				return
			}
			if attempts != tc.wantAttempts || stored != 1 {
				t.Fatalf("station evidence save attempts = %d (stored %d), want %d", attempts, stored, tc.wantAttempts)
			}
		})
	}
}

// Task 8 / F09: EDDN station snapshots carry every ordinary public module and
// ship (they used to keep only eight core-internal families and a fixed list
// of 20 hulls), omit commander/rank-dependent items, keep the exact MarketID,
// and are suppressed rather than truncated when an item cannot be classified.
func TestAuditStationInventoryComplete(t *testing.T) {
	m := &Manager{uploader: "fixture", gameVersion: "4.1", gameBuild: "fixture"}
	head := `{"timestamp":"2026-11-15T12:00:00Z","StarSystem":"Fixture","StationName":"Fixture Port","MarketID":18446744073709551615,`
	modules := func(raw []byte) []string {
		var p struct {
			Message struct {
				MarketID json.Number `json:"marketId"`
				Modules  []string    `json:"modules"`
				Ships    []string    `json:"ships"`
			} `json:"message"`
		}
		d := json.NewDecoder(strings.NewReader(string(raw)))
		d.UseNumber()
		if err := d.Decode(&p); err != nil {
			t.Fatal(err)
		}
		if p.Message.MarketID != "18446744073709551615" {
			t.Fatalf("MarketID not exact: %s", p.Message.MarketID)
		}
		return append(p.Message.Modules, p.Message.Ships...)
	}
	out, err := m.stationPayload("Outfitting", object([]byte(head+`"Items":[{"id":128049513,"Name":"hpt_chafflauncher_tiny","BuyPrice":1},{"id":128064272,"Name":"int_shieldgenerator_size3_class5","BuyPrice":1},{"id":128666676,"Name":"int_fuelscoop_size1_class5","BuyPrice":1},{"id":128049381,"Name":"hpt_pulselaser_fixed_small","BuyPrice":1},{"id":128049292,"Name":"dolphin_armour_grade1","BuyPrice":1},{"id":128671325,"Name":"int_shieldgenerator_size3_class5_strong","BuyPrice":1},{"id":128672317,"Name":"int_planetapproachsuite","BuyPrice":1},{"Name":"paintjob_cobramkiii_default_52","BuyPrice":1}]}`)))
	if err != nil {
		t.Fatal("outfitting:", err)
	}
	got := strings.Join(modules(out), ",")
	if got != "Hpt_ChaffLauncher_Tiny,Int_ShieldGenerator_Size3_Class5,Int_FuelScoop_Size1_Class5,Hpt_PulseLaser_Fixed_Small,Dolphin_Armour_Grade1" {
		t.Fatalf("outfitting modules = %s", got)
	}
	out, err = m.stationPayload("Shipyard", object([]byte(head+`"PriceList":[{"id":128049291,"ShipType":"dolphin","ShipPrice":1},{"id":128816567,"ShipType":"krait_mkii","ShipPrice":1},{"ShipType":"SideWinder","ShipPrice":1},{"ShipType":"CobraMkIV","ShipPrice":1}]}`)))
	if err != nil {
		t.Fatal("shipyard:", err)
	}
	if got = strings.Join(modules(out), ","); got != "Dolphin,Krait_MkII,SideWinder" {
		t.Fatalf("shipyard ships = %s", got)
	}
	// One unclassifiable item suppresses the whole category snapshot.
	for kind, items := range map[string]string{
		"Outfitting": `"Items":[{"id":128049513,"Name":"hpt_chafflauncher_tiny"},{"id":999999999,"Name":"hpt_future_weapon"}]`,
		"Shipyard":   `"PriceList":[{"ShipType":"SideWinder"},{"ShipType":"Future_Hull"}]`,
	} {
		if out, err := m.stationPayload(kind, object([]byte(head+items+`}`))); err == nil || !strings.Contains(err.Error(), "not classified") {
			t.Fatalf("%s with an unknown item published (%v): %s", kind, err, out)
		}
	}
}

// Task 16 (G3I): with the refreshed catalogue, current ordinary station
// inventories that include newer modules and hulls become publishable, while
// unverified entitlements, ambiguous shared symbols and unknown items keep the
// snapshot suppressed with an accurate reason, and personal unlocks stay out.
func TestAuditStationNewInventory(t *testing.T) {
	m := &Manager{uploader: "fixture", gameVersion: "4.1", gameBuild: "fixture"}
	head := `{"timestamp":"2026-11-24T12:00:00Z","StarSystem":"Fixture","StationName":"Fixture Port","MarketID":18446744073709551615,`
	payload := func(kind, items string) (string, error) {
		out, err := m.stationPayload(kind, object([]byte(head+items+`}`)))
		return string(out), err
	}
	// Publishable: reviewed SCO drive, a standard module by exact ID, Python Mk II
	// armour (reviewed ordinary entitlement), a standard module; a tech-broker
	// unlock (Enzyme Missile Rack) is omitted.
	out, err := payload("Outfitting", `"Items":[{"id":129030592,"Name":"int_hyperdrive_overcharge_size5_class5"},{"id":128049431,"Name":"hpt_beamlaser_fixed_huge"},{"id":129030466,"Name":"python_nx_armour_grade1"},{"id":128049513,"Name":"hpt_chafflauncher_tiny"},{"id":128833995,"Name":"hpt_causticmissile_fixed_medium"}]`)
	if err != nil {
		t.Fatal("new ordinary outfitting suppressed:", err)
	}
	for _, want := range []string{"Int_Hyperdrive_Overcharge_Size5_Class5", "Hpt_BeamLaser_Fixed_Huge", "Python_NX_Armour_Grade1", "Hpt_ChaffLauncher_Tiny", "18446744073709551615"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s in %s", want, out)
		}
	}
	if strings.Count(out, "Hpt_BeamLaser_Fixed_Huge") != 1 || strings.Contains(strings.ToLower(out), "causticmissile") {
		t.Fatalf("variant not de-duplicated or unlock published: %s", out)
	}
	if out, err = payload("Shipyard", `"PriceList":[{"id":129030464,"ShipType":"python_nx"},{"ShipType":"SideWinder"},{"id":128049291,"ShipType":"dolphin"}]`); err != nil || !strings.Contains(out, `"Python_NX"`) {
		t.Fatalf("new ordinary shipyard: %v %s", err, out)
	}
	// Suppressed with accurate reasons.
	for _, tc := range []struct{ kind, items, reason string }{
		{"Shipyard", `"PriceList":[{"ShipType":"SideWinder"},{"id":129030534,"ShipType":"type8"}]`, "availability not verified for ELITE_V_TYPE8"},
		{"Outfitting", `"Items":[{"id":128049513,"Name":"hpt_chafflauncher_tiny"},{"id":129030535,"Name":"type8_armour_grade1"}]`, "ELITE_V_TYPE8"},
		{"Outfitting", `"Items":[{"id":128049513,"Name":"hpt_chafflauncher_tiny"},{"Name":"hpt_causticmissile_fixed_medium"}]`, "shared symbol and no numeric ID"},
		{"Outfitting", `"Items":[{"id":128049513,"Name":"hpt_chafflauncher_tiny"},{"id":129999999,"Name":"hpt_future_weapon_fixed_small"}]`, "not classified by the bundled catalogue"},
		// G3I review F4: a pre-engineered variant added by the refresh with a
		// blank entitlement is not evidence of public stock.
		{"Outfitting", `"Items":[{"id":128049431,"Name":"hpt_beamlaser_fixed_huge"},{"id":129046189,"Name":"hpt_beamlaser_fixed_huge"}]`, "availability not verified for unreviewed item 129046189"},
		{"Outfitting", `"Items":[{"id":128049513,"Name":"hpt_chafflauncher_tiny"},{"id":129041074,"Name":"int_mkiiagileboost_engine_size5_class5"}]`, "unreviewed item 129041074"},
	} {
		if out, err := payload(tc.kind, tc.items); err == nil || !strings.Contains(err.Error(), tc.reason) {
			t.Fatalf("%s %s: published or wrong reason (%v): %s", tc.kind, tc.items, err, out)
		}
	}
}
