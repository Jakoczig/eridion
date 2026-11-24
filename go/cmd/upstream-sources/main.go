// Command upstream-sources maintains the reviewed upstream data bundled with
// ERIDION (Task 23, catalogue and provider-contract scope).
//
//	upstream-sources verify            offline: bundled bytes match the lock
//	upstream-sources check             network: current / update available / unable to check
//	upstream-sources fetch  -source ID [-revision SHA] -out DIR
//	upstream-sources diff   -source ID -dir DIR       reviewable report (no writes)
//	upstream-sources refresh -source ID -dir DIR -revision SHA   explicit, rewrites bundled copies and the lock
//
// Run from the repository root. Nothing runs automatically: the application
// never downloads data at runtime, and refresh only happens when a developer
// invokes it and reviews the resulting git diff. Manual corrections live in
// separate reviewed files (for example go/internal/catalog/entitlement-review.csv)
// that refresh never writes.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const lockPath = "upstream-sources.lock.json"

// Lock is the reviewed state of every bundled upstream source.
type Lock struct {
	Version int      `json:"version"`
	Sources []Source `json:"sources"`
}

// Source is one upstream repository and the files ERIDION bundles from it.
type Source struct {
	ID           string `json:"id"`
	Repository   string `json:"repository"`
	Revision     string `json:"revision"`
	ObservedHead string `json:"observedHead"`
	CheckedAt    string `json:"checkedAt"`
	Status       string `json:"status"`
	License      string `json:"license"`
	Attribution  string `json:"attribution"`
	Notes        string `json:"notes,omitempty"`
	Files        []File `json:"files"`
}

// File is one upstream file and its byte-identical bundled copies.
type File struct {
	Path    string   `json:"path"`
	SHA256  string   `json:"sha256"`
	Bundled []string `json:"bundled"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: upstream-sources verify|check|fetch|diff|refresh [flags]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	source := fs.String("source", "", "source id")
	revision := fs.String("revision", "", "immutable upstream revision")
	out := fs.String("out", "", "output directory for fetch")
	dir := fs.String("dir", "", "directory holding fetched candidate files")
	_ = fs.Parse(os.Args[2:])
	lock, err := readLock()
	if err != nil {
		fail(err)
	}
	switch os.Args[1] {
	case "verify":
		problems := Verify(lock, ".")
		for _, p := range problems {
			fmt.Println("FAIL", p)
		}
		if len(problems) > 0 {
			os.Exit(1)
		}
		fmt.Println("ok: all bundled upstream files match", lockPath)
	case "check":
		for _, s := range lock.Sources {
			status, head := Check(s, http.DefaultClient)
			fmt.Printf("%-16s %-18s locked=%s head=%s\n", s.ID, status, short(s.Revision), short(head))
		}
	case "fetch":
		s := find(lock, *source)
		rev := *revision
		if rev == "" {
			rev = s.Revision
		}
		if *out == "" {
			fail(errors.New("-out required"))
		}
		if err := Fetch(s, rev, *out, http.DefaultClient); err != nil {
			fail(err)
		}
	case "diff":
		s := find(lock, *source)
		report, err := Diff(s, ".", *dir)
		if err != nil {
			fail(err)
		}
		fmt.Print(report)
	case "refresh":
		s := find(lock, *source)
		if *dir == "" || *revision == "" {
			fail(errors.New("refresh needs -dir and -revision; review the diff report first"))
		}
		if err := Refresh(&lock, s.ID, ".", *dir, *revision); err != nil {
			fail(err)
		}
		if err := writeLock(lock); err != nil {
			fail(err)
		}
		fmt.Println("refreshed", s.ID, "to", *revision, "- review `git diff` before committing")
	default:
		fail(errors.New("unknown command " + os.Args[1]))
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, "upstream-sources:", err); os.Exit(1) }
func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func readLock() (Lock, error) {
	var l Lock
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		return l, err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return l, d.Decode(&l)
}

func writeLock(l Lock) error {
	raw, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(lockPath, append(raw, '\n'), 0o644)
}

func find(l Lock, id string) Source {
	for _, s := range l.Sources {
		if s.ID == id {
			return s
		}
	}
	fail(fmt.Errorf("unknown source %q", id))
	return Source{}
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// Verify reports every bundled copy whose bytes differ from the locked hash.
func Verify(l Lock, root string) []string {
	var problems []string
	ids := map[string]bool{}
	for _, s := range l.Sources {
		if ids[s.ID] {
			problems = append(problems, "duplicate source id "+s.ID)
		}
		ids[s.ID] = true
		for _, f := range s.Files {
			if len(f.SHA256) != 64 {
				problems = append(problems, s.ID+"/"+f.Path+": missing sha256")
			}
			for _, b := range f.Bundled {
				raw, err := os.ReadFile(filepath.Join(root, b))
				if err != nil {
					problems = append(problems, b+": "+err.Error())
				} else if digest(raw) != f.SHA256 {
					problems = append(problems, b+": bytes differ from "+s.ID+"@"+short(s.Revision)+" "+f.Path+" (edit through refresh, keep manual corrections in reviewed override files)")
				}
			}
		}
	}
	return problems
}

var githubRepo = regexp.MustCompile(`^https://github\.com/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)$`)

// Check compares the locked revision with the repository's current default
// head using the read-only git smart-HTTP advertisement (no API token).
func Check(s Source, client *http.Client) (status, head string) {
	m := githubRepo.FindStringSubmatch(s.Repository)
	if m == nil || s.Revision == "" {
		return "unable_to_check", ""
	}
	c := *client
	c.Timeout = 20 * time.Second
	resp, err := c.Get(s.Repository + ".git/info/refs?service=git-upload-pack")
	if err != nil {
		return "unable_to_check", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "unable_to_check", ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "unable_to_check", ""
	}
	head = advertisedHead(body)
	switch {
	case head == "":
		return "unable_to_check", ""
	case head == s.Revision:
		return "current", head
	default:
		return "update_available", head
	}
}

var headLine = regexp.MustCompile(`([0-9a-f]{40}) HEAD`)

func advertisedHead(body []byte) string {
	if m := headLine.FindSubmatch(body); m != nil {
		return string(m[1])
	}
	return ""
}

// Fetch downloads the locked files at revision into out, bounded in size and
// time. It never writes inside the repository's bundled paths.
func Fetch(s Source, revision, out string, client *http.Client) error {
	m := githubRepo.FindStringSubmatch(s.Repository)
	if m == nil || !fullRevision.MatchString(revision) {
		return errors.New("fetch needs a GitHub repository and a 40-character revision")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	c := *client
	c.Timeout = 30 * time.Second
	for _, f := range s.Files {
		url := "https://raw.githubusercontent.com/" + m[1] + "/" + m[2] + "/" + revision + "/" + f.Path
		resp, err := c.Get(url)
		if err != nil {
			return err
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 {
			return fmt.Errorf("%s: HTTP %d %v", f.Path, resp.StatusCode, err)
		}
		dst := filepath.Join(out, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, raw, 0o644); err != nil {
			return err
		}
		fmt.Printf("%s %s\n", digest(raw), f.Path)
	}
	return nil
}

// Diff writes a reviewable report comparing candidate files in dir with the
// bundled copies: byte changes for every file and, for FDevIDs CSV data,
// added/removed/changed IDs, duplicate IDs, header changes and symbols
// shared by several IDs.
func Diff(s Source, root, dir string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s: bundled %s vs candidate %s\n\n", s.ID, short(s.Revision), dir)
	for _, f := range s.Files {
		cand, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f.Path)))
		if err != nil {
			return "", err
		}
		old, err := os.ReadFile(filepath.Join(root, f.Bundled[0]))
		if err != nil {
			return "", err
		}
		if digest(cand) == digest(old) {
			fmt.Fprintf(&b, "## %s: unchanged\n\n", f.Path)
			continue
		}
		fmt.Fprintf(&b, "## %s: changed (sha256 %s -> %s)\n\n", f.Path, short(digest(old)), short(digest(cand)))
		if strings.HasSuffix(f.Path, ".csv") {
			report, err := CSVDiff(old, cand)
			if err != nil {
				return "", err
			}
			b.WriteString(report)
		}
	}
	return b.String(), nil
}

// CSVDiff compares two FDevIDs-style CSV files keyed by their "id" column.
func CSVDiff(oldRaw, newRaw []byte) (string, error) {
	oh, old, err := readCSV(oldRaw)
	if err != nil {
		return "", err
	}
	nh, cur, err := readCSV(newRaw)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if strings.Join(oh, ",") != strings.Join(nh, ",") {
		fmt.Fprintf(&b, "- SCHEMA CHANGE: header %v -> %v (review the converter before refreshing)\n", oh, nh)
	}
	oldBy, dupOld := byID(old)
	newBy, dupNew := byID(cur)
	if len(dupOld)+len(dupNew) > 0 {
		fmt.Fprintf(&b, "- DUPLICATE IDS: bundled %v candidate %v\n", dupOld, dupNew)
	}
	var added, removed, changed []string
	for id, row := range newBy {
		if o, ok := oldBy[id]; !ok {
			added = append(added, id+" "+row["symbol"]+" \""+row["name"]+"\" "+row["entitlement"])
		} else if diff := rowDiff(o, row); diff != "" {
			changed = append(changed, id+" "+row["symbol"]+" "+diff)
		}
	}
	for id, row := range oldBy {
		if _, ok := newBy[id]; !ok {
			removed = append(removed, id+" "+row["symbol"])
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)
	fmt.Fprintf(&b, "- rows: %d -> %d; added %d, removed %d, changed %d\n", len(old), len(cur), len(added), len(removed), len(changed))
	for _, group := range []struct {
		label string
		rows  []string
	}{{"added", added}, {"removed", removed}, {"changed", changed}} {
		for _, r := range group.rows {
			fmt.Fprintf(&b, "  - %s %s\n", group.label, r)
		}
	}
	for _, c := range Collisions(cur) {
		fmt.Fprintf(&b, "- shared symbol %s\n", c)
	}
	b.WriteString("\n")
	return b.String(), nil
}

func readCSV(raw []byte) ([]string, []map[string]string, error) {
	rows, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
	if err != nil || len(rows) == 0 {
		return nil, nil, errors.New("unreadable CSV")
	}
	out := []map[string]string{}
	for _, r := range rows[1:] {
		m := map[string]string{}
		for i, h := range rows[0] {
			if i < len(r) {
				m[h] = r[i]
			}
		}
		out = append(out, m)
	}
	return rows[0], out, nil
}

func byID(rows []map[string]string) (map[string]map[string]string, []string) {
	out := map[string]map[string]string{}
	var dup []string
	for _, r := range rows {
		if _, ok := out[r["id"]]; ok {
			dup = append(dup, r["id"])
		}
		out[r["id"]] = r
	}
	return out, dup
}

func rowDiff(a, b map[string]string) string {
	var parts []string
	for k, v := range b {
		if a[k] != v {
			parts = append(parts, fmt.Sprintf("%s: %q -> %q", k, a[k], v))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}

// Collisions lists symbols (case-insensitive) used by more than one ID.
func Collisions(rows []map[string]string) []string {
	by := map[string][]string{}
	for _, r := range rows {
		k := strings.ToLower(r["symbol"])
		by[k] = append(by[k], r["id"]+"="+r["name"])
	}
	var out []string
	for k, v := range by {
		if len(v) > 1 {
			sort.Strings(v)
			out = append(out, k+": "+strings.Join(v, ", "))
		}
	}
	sort.Strings(out)
	return out
}

// Refresh copies verified candidate files into every bundled path and records
// the new revision and hashes in the lock. It touches only the locked files.
// manualFiles are reviewed local corrections and records. Refresh never
// writes them, whatever the lock says.
var manualFiles = map[string]bool{"entitlement-review.csv": true, "collisions-reviewed.txt": true, "PROVENANCE.md": true}

var fullRevision = regexp.MustCompile(`^[0-9a-f]{40}$`)

func refreshTarget(root, bundled string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(bundled))
	if filepath.IsAbs(bundled) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%s: bundled path outside the repository", bundled)
	}
	if manualFiles[filepath.Base(clean)] {
		return "", fmt.Errorf("%s: manual correction file; refresh never overwrites it", bundled)
	}
	return filepath.Join(root, filepath.FromSlash(clean)), nil
}

func Refresh(l *Lock, id, root, dir, revision string) error {
	if !fullRevision.MatchString(revision) {
		return errors.New("refresh needs the immutable 40-character revision the files were fetched at")
	}
	for si := range l.Sources {
		s := &l.Sources[si]
		if s.ID != id {
			continue
		}
		// Validate every target and read every candidate before writing
		// anything, so a refused refresh leaves the tree untouched.
		data := make([][]byte, len(s.Files))
		for fi, f := range s.Files {
			raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f.Path)))
			if err != nil {
				return err
			}
			for _, b := range f.Bundled {
				if _, err := refreshTarget(root, b); err != nil {
					return err
				}
			}
			data[fi] = raw
		}
		for fi := range s.Files {
			f := &s.Files[fi]
			for _, b := range f.Bundled {
				dst, _ := refreshTarget(root, b)
				if err := os.WriteFile(dst, data[fi], 0o644); err != nil {
					return err
				}
			}
			f.SHA256 = digest(data[fi])
		}
		s.Revision, s.ObservedHead = revision, revision
		s.CheckedAt = time.Now().UTC().Format("2006-01-02")
		s.Status = "current"
		return nil
	}
	return fmt.Errorf("unknown source %q", id)
}
