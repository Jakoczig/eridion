package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Jakoczig/eridion/go/internal/fieldexport"
)

// Task 19 / F25: both web exports reach the real controller boundary. The
// controller used to accept only the pre-rebrand daedalus-* names, so both
// ERIDION exports failed with "Invalid field export" before any dialog.
func TestAuditFieldExportThroughController(t *testing.T) {
	c := &Controller{}
	if err := c.SaveFieldCSV(fieldexport.Itinerary, "a"); err == nil {
		t.Fatal("export succeeded without a native save dialog")
	}
	dir := t.TempDir()
	var chosen string
	c.BindCSV(func(name, data string) error {
		// Stand-in for the native dialog: the user picks a destination, then
		// the production write path runs.
		if chosen == "cancel" {
			return fieldexport.Write("", data)
		}
		return fieldexport.Write(filepath.Join(dir, chosen), data)
	})
	for _, name := range []string{fieldexport.Itinerary, fieldexport.PrivateActivity} {
		chosen = name
		content := "System Name\r\n'=cmd\r\n"
		if err := c.SaveFieldCSV(name, content); err != nil {
			t.Fatalf("%s rejected at the controller: %v", name, err)
		}
		if raw, err := os.ReadFile(filepath.Join(dir, name)); err != nil || string(raw) != content {
			t.Fatalf("%s content = %q %v", name, raw, err)
		}
	}
	for _, name := range []string{"daedalus-itinerary.csv", "../" + fieldexport.Itinerary, "evil.exe"} {
		if err := c.SaveFieldCSV(name, "x"); err == nil || err.Error() != "Invalid field export" {
			t.Fatalf("%q accepted: %v", name, err)
		}
	}
	chosen = "cancel"
	if err := c.SaveFieldCSV(fieldexport.Itinerary, "x"); !errors.Is(err, fieldexport.ErrCancelled) {
		t.Fatalf("cancel = %v", err)
	}
	chosen = fieldexport.Itinerary // already exists: preserved, error reported
	if err := c.SaveFieldCSV(fieldexport.Itinerary, "replacement"); err == nil {
		t.Fatal("existing export overwritten")
	}
	chosen = filepath.Join("missing", "route.csv")
	if err := c.SaveFieldCSV(fieldexport.Itinerary, "x"); err == nil {
		t.Fatal("I/O failure not reported")
	}
}
