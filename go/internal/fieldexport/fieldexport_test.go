package fieldexport

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Task 19 / F25: the native save bridge accepts exactly the filenames the web
// exports emit, and writes content safely to the chosen destination.
func TestAuditFieldExportNames(t *testing.T) {
	for _, name := range []string{Itinerary, PrivateActivity} {
		if !Allowed(name) {
			t.Fatalf("%s rejected", name)
		}
	}
	for _, name := range []string{"", "daedalus-itinerary.csv", "eridion-itinerary.csv.exe", "../eridion-itinerary.csv", `C:\eridion-itinerary.csv`, "eridion-private-activity.CSV "} {
		if Allowed(name) {
			t.Fatalf("%q accepted", name)
		}
	}
}

func TestAuditFieldExportWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "route.csv")
	content := "System,Name\r\n\"Sol\",\"=1+1\"\n"
	if err := Write(path, content); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(path); err != nil || string(raw) != content {
		t.Fatalf("content = %q %v", raw, err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Fatalf("export readable by others: %v", info.Mode())
	}
	// An existing file is preserved, never overwritten.
	if err := Write(path, "replacement"); err == nil {
		t.Fatal("existing file overwritten")
	}
	if raw, _ := os.ReadFile(path); string(raw) != content {
		t.Fatal("existing file changed")
	}
	// Cancellation, wrong extension, relative or directory paths and missing
	// directories are reported, not written.
	if err := Write("", content); !errors.Is(err, ErrCancelled) {
		t.Fatalf("cancel = %v", err)
	}
	for _, bad := range []string{filepath.Join(dir, "route.txt"), "relative.csv", dir, filepath.Join(dir, "missing", "route.csv")} {
		if err := Write(bad, content); err == nil {
			t.Fatalf("%q written", bad)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "route.txt")); !os.IsNotExist(err) {
		t.Fatal("non-CSV destination created")
	}
	if err := Write(filepath.Join(dir, "big.csv"), strings.Repeat("x", MaxBytes+1)); err == nil {
		t.Fatal("oversized export written")
	}
}
