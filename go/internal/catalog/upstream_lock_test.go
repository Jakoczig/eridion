package catalog

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const repoRoot = "../../.."

// Task 23: offline conformance of the bundled upstream data. Every bundled
// copy matches the locked revision byte-for-byte (so generated or hand edits
// cannot drift silently), CSV schemas match the converter, IDs are unique and
// every shared symbol is a reviewed collision.
func TestUpstreamLockConformance(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot, "upstream-sources.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Sources []struct {
			ID, Revision, Status string
			Files                []struct {
				Path, SHA256 string
				Bundled      []string
			}
		}
	}
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, s := range lock.Sources {
		if seen[s.ID] || (s.Status == "current" && len(s.Revision) != 40) {
			t.Fatalf("source %s: duplicate or unpinned current source", s.ID)
		}
		seen[s.ID] = true
		for _, f := range s.Files {
			for _, b := range f.Bundled {
				data, err := os.ReadFile(filepath.Join(repoRoot, b))
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(data)
				if hex.EncodeToString(sum[:]) != f.SHA256 {
					t.Errorf("%s differs from %s@%s %s; refresh through go/cmd/upstream-sources", b, s.ID, s.Revision, f.Path)
				}
			}
		}
	}
	for file, header := range map[string]string{
		"shipyard.csv":       "id,symbol,name,entitlement",
		"outfitting.csv":     "id,symbol,category,name,mount,guidance,ship,class,rating,entitlement",
		"commodity.csv":      "id,symbol,category,name",
		"microresources.csv": "id,symbol,category,English name",
	} {
		b, _ := data.ReadFile(file)
		rows, err := csv.NewReader(strings.NewReader(string(b))).ReadAll()
		if err != nil || len(rows) < 2 || strings.Join(rows[0], ",") != header {
			t.Fatalf("%s schema drift: %v", file, err)
		}
		ids := map[string]bool{}
		for _, r := range rows[1:] {
			if r[0] == "" || ids[r[0]] {
				t.Fatalf("%s: duplicate or empty id %q", file, r[0])
			}
			ids[r[0]] = true
		}
	}
	// Shared module symbols must match the reviewed list exactly.
	b, _ := data.ReadFile("outfitting.csv")
	rows, _ := csv.NewReader(strings.NewReader(string(b))).ReadAll()
	by := map[string][]string{}
	for _, r := range rows[1:] {
		by[strings.ToLower(r[1])] = append(by[strings.ToLower(r[1])], r[0])
	}
	var shared []string
	for k, v := range by {
		if len(v) > 1 {
			shared = append(shared, k)
		}
	}
	sort.Strings(shared)
	reviewed, _ := data.ReadFile("collisions-reviewed.txt")
	var want []string
	for _, l := range strings.Split(string(reviewed), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			want = append(want, strings.ToLower(strings.Fields(l)[0]))
		}
	}
	sort.Strings(want)
	if strings.Join(shared, ",") != strings.Join(want, ",") {
		t.Fatalf("shared symbols changed; review them in collisions-reviewed.txt:\n got  %v\n want %v", shared, want)
	}
}
