package webapp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jakoczig/eridion/go/internal/fieldexport"
)

// Task 19 / F25: both exports produce their CSV and pass the native bridge's
// filename contract and safe write (the desktop controller applies the same
// fieldexport contract; see internal/desktop/field_export_test.go).
func TestAuditFieldExportsReachNativeBridge(t *testing.T) {
	s, _, _, _, _ := g2eFixture(t)
	dir := t.TempDir()
	saved := map[string]string{}
	var failNext error
	s.saveFieldCSV = func(name, data string) error {
		if !fieldexport.Allowed(name) {
			return errors.New("Invalid field export")
		}
		if failNext != nil {
			return failNext
		}
		path := filepath.Join(dir, name)
		if err := fieldexport.Write(path, data); err != nil {
			return err
		}
		saved[name] = path
		return nil
	}
	call := func(name string, v any) (any, error) {
		raw, _ := json.Marshal(v)
		return s.fieldRPC(context.Background(), name, raw)
	}
	if _, err := call("importGoItinerary", map[string]any{"name": "Export route", "csv": "system\nAlpha\nBeta\n"}); err != nil {
		t.Fatal(err)
	}
	if _, err := call("exportGoItinerary", map[string]any{"acknowledge": true}); err != nil {
		t.Fatal("itinerary export:", err)
	}
	if _, err := call("exportGoBGSTally", map[string]any{"context": "missions", "acknowledge": true}); err != nil {
		t.Fatal("private activity export:", err)
	}
	itinerary, _ := os.ReadFile(saved[fieldexport.Itinerary])
	activity, _ := os.ReadFile(saved[fieldexport.PrivateActivity])
	if !strings.HasPrefix(string(itinerary), "System Name,SystemAddress") || !strings.Contains(string(itinerary), "Alpha") || !strings.Contains(string(itinerary), "Beta") {
		t.Fatalf("itinerary CSV = %q", itinerary)
	}
	if !strings.HasPrefix(string(activity), "Commander,Event,SystemAddress") {
		t.Fatalf("activity CSV = %q", activity)
	}
	// Cancellation and I/O failures reach the caller unchanged.
	failNext = fieldexport.ErrCancelled
	if _, err := call("exportGoItinerary", map[string]any{"acknowledge": true}); !errors.Is(err, fieldexport.ErrCancelled) {
		t.Fatalf("cancel = %v", err)
	}
	failNext = nil
	if _, err := call("exportGoItinerary", map[string]any{"acknowledge": true}); err == nil {
		t.Fatal("existing destination overwritten")
	}
}
