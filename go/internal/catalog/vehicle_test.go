package catalog

import "testing"

func TestVehicleKind(t *testing.T) {
	for symbol, want := range map[string]string{
		"FlightSuit": "suit", "ExplorationSuit_Class3": "suit", "UtilitySuit_Class1": "suit", "TacticalSuit_Class5": "suit",
		"testbuggy": "srv", "combat_multicrew_srv_01": "srv",
		"independent_fighter": "fighter", "empire_fighter": "fighter", "federation_fighter": "fighter", "gdn_hybrid_fighter_v2": "fighter",
		"adder_taxi": "taxi",
		"anaconda":   "ship", "Python_NX": "ship", "unfamiliar_future_hull": "ship",
		"": "",
	} {
		if got := VehicleKind(symbol); got != want {
			t.Fatalf("VehicleKind(%q) = %q, want %q", symbol, got, want)
		}
	}
}
