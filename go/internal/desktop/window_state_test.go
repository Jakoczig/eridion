package desktop

import "testing"

func TestDefaultDashboardFitsWorkAreaAndSavedDimensions(t *testing.T) {
	for _, area := range [][2]int{{1920, 1040}, {1366, 728}, {900, 1400}, {640, 450}} {
		got := dashboardSize(area[0], area[1], windowSize{})
		if got.Width > area[0] || got.Height > area[1] || got.Width*9 != got.Height*16 {
			t.Fatal(area, got)
		}
	}
	saved := windowSize{Width: 1100, Height: 800, Maximized: true}
	if got := dashboardSize(1920, 1040, saved); got != saved {
		t.Fatal("saved geometry replaced", got)
	}
}
func TestDashboardWindowStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	saved := windowSize{Width: 1111, Height: 777, Maximized: true}
	if err := saveWindowSize(dir, saved); err != nil {
		t.Fatal(err)
	}
	if got := loadWindowSize(dir); got != saved {
		t.Fatal(got)
	}
}
