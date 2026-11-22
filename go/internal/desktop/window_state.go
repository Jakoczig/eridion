package desktop

import (
	"encoding/json"
	"github.com/Jakoczig/eridion/go/internal/privatefs"
)

type windowSize struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized"`
}

func validWindowSize(v windowSize) bool {
	return v.Width >= 320 && v.Width <= 16384 && v.Height >= 240 && v.Height <= 16384
}
func loadWindowSize(profile string) windowSize {
	if profile == "" {
		return windowSize{}
	}
	b, err := privatefs.Read(profile, "dashboard-window.json")
	var v windowSize
	if err != nil || json.Unmarshal(b, &v) != nil || !validWindowSize(v) {
		return windowSize{}
	}
	return v
}
func saveWindowSize(profile string, v windowSize) error {
	if profile == "" || !validWindowSize(v) {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return privatefs.Write(profile, "dashboard-window.json", b)
}
func dashboardSize(workWidth, workHeight int, saved windowSize) windowSize {
	if workWidth <= 0 || workHeight <= 0 {
		workWidth, workHeight = 1600, 1000
	}
	if validWindowSize(saved) {
		saved.Width = min(saved.Width, workWidth)
		saved.Height = min(saved.Height, workHeight)
		return saved
	}
	// Logical monitor work area includes desktop scaling. Reserve window chrome.
	units := min(90, min(max(1, (workWidth-32)/16), max(1, (workHeight-48)/9)))
	return windowSize{Width: units * 16, Height: units * 9}
}
