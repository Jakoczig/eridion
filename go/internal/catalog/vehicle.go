package catalog

import "strings"

// VehicleKind classifies a journal "Ship"/"ShipType" symbol as the
// commander's main ship or another controlled entity. Unknown symbols are
// treated as genuine ships so a future hull is never rejected merely because
// it is missing from the catalogue. An empty symbol returns "".
func VehicleKind(symbol string) string {
	s := strings.ToLower(strings.TrimSpace(symbol))
	switch {
	case s == "":
		return ""
	case strings.Contains(s, "suit"):
		return "suit"
	case s == "testbuggy" || strings.Contains(s, "_srv"):
		return "srv"
	case strings.HasSuffix(s, "_fighter") || strings.Contains(s, "_fighter_"):
		return "fighter"
	case strings.HasSuffix(s, "_taxi"):
		return "taxi"
	default:
		return "ship"
	}
}
