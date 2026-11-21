package webapp

import (
	"encoding/json"
	"github.com/Jakoczig/eridion/go/internal/privatefs"
	"testing"
)

func TestG3DRendererPreferencesAreIndependentAndPersisted(t *testing.T) {
	s, _, _, _, _ := g2eFixture(t)
	owner := socketPrincipal{role: "owner"}
	p := s.displayPolicy(owner)
	if p["galaxyMap"] != true || p["localAtlas"] != true || p["mapDuringFSD"] != false {
		t.Fatal(p)
	}
	for _, pair := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
		a, b := pair[0], pair[1]
		config := displayConfig{GalaxyMap: &a, LocalAtlas: &b, MapDuringFSD: true, Restore: true, CloseMode: "tray", Owner: true}
		raw, _ := json.Marshal(config)
		if err := s.saveDisplayConfig(raw); err != nil {
			t.Fatal(err)
		}
		got := s.displayPolicy(owner)
		if got["galaxyMap"] != a || got["localAtlas"] != b || got["mapDuringFSD"] != true {
			t.Fatal(got)
		}
		// Renderer preferences never select a remote device or authorize downloads.
		if s.displayPolicy(socketPrincipal{role: "viewer", deviceID: "unselected"})["selected"] != false {
			t.Fatal("viewer selected")
		}
		var loaded displayConfig
		data, err := privatefs.Read(s.profile, "companion.json")
		if err != nil || json.Unmarshal(data, &loaded) != nil || loaded.GalaxyMap == nil || *loaded.GalaxyMap != a || loaded.LocalAtlas == nil || *loaded.LocalAtlas != b {
			t.Fatal("preferences not persisted", err)
		}
	}
}
