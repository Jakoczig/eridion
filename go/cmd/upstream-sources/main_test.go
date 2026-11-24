package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeTransport func(*http.Request) (*http.Response, error)

func (f fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func reply(status int, body string) *http.Client {
	return &http.Client{Transport: fakeTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
}

const locked = "c7c29f7a13df543d4bc8b947eab29f88e06100fa"

func TestCheckStatuses(t *testing.T) {
	s := Source{ID: "x", Repository: "https://github.com/EDCD/FDevIDs", Revision: locked}
	adv := func(head string) string {
		return "001e# service=git-upload-pack\n0000" + "00f4" + head + " HEAD\x00multi_ack\n"
	}
	cases := []struct {
		client *http.Client
		src    Source
		want   string
	}{
		{reply(200, adv(locked)), s, "current"},
		{reply(200, adv("153e62426eb58b6cd23def535cb4b7e1082e777f")), s, "update_available"},
		{reply(503, ""), s, "unable_to_check"},
		{reply(200, "garbage"), s, "unable_to_check"},
		{&http.Client{Transport: fakeTransport(func(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF })}, s, "unable_to_check"},
		{reply(200, adv(locked)), Source{ID: "unpinned", Repository: "https://github.com/EDCD/coriolis-data"}, "unable_to_check"},
	}
	for i, c := range cases {
		if got, _ := Check(c.src, c.client); got != c.want {
			t.Errorf("case %d: %s, want %s", i, got, c.want)
		}
	}
}

func fixtureRepo(t *testing.T) (string, Lock) {
	root := t.TempDir()
	raw := []byte("id,symbol,name\n1,a,Alpha\n")
	for _, p := range []string{"go/internal/catalog/x.csv", "resources/x.csv", "go/internal/catalog/entitlement-review.csv"} {
		_ = os.MkdirAll(filepath.Join(root, filepath.Dir(p)), 0o755)
		if err := os.WriteFile(filepath.Join(root, p), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, Lock{Version: 1, Sources: []Source{{ID: "s", Repository: "https://github.com/EDCD/FDevIDs", Revision: locked, Files: []File{{Path: "x.csv", SHA256: digest(raw), Bundled: []string{"go/internal/catalog/x.csv", "resources/x.csv"}}}}}}
}

func TestVerifyDetectsHandEditedCopy(t *testing.T) {
	root, l := fixtureRepo(t)
	if p := Verify(l, root); len(p) != 0 {
		t.Fatal(p)
	}
	_ = os.WriteFile(filepath.Join(root, "resources/x.csv"), []byte("id,symbol,name\n1,a,Edited\n"), 0o644)
	if p := Verify(l, root); len(p) != 1 || !strings.Contains(p[0], "resources/x.csv") {
		t.Fatal("hand edit not detected", p)
	}
	l.Sources = append(l.Sources, l.Sources[0])
	if p := Verify(l, root); !strings.Contains(strings.Join(p, "\n"), "duplicate source id") {
		t.Fatal("duplicate source not detected", p)
	}
}

func TestCSVDiffReportsReviewableChanges(t *testing.T) {
	report, err := CSVDiff([]byte("id,symbol,name\n1,a,Alpha\n2,b,Beta\n"), []byte("id,symbol,name\n1,a,Alpha Prime\n3,a,Alpha Variant\n3,c,Dup\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"2", "3", "Alpha Prime"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report lacks %q:\n%s", want, report)
		}
	}
	if !strings.Contains(strings.ToLower(report), "duplicate") {
		t.Fatalf("duplicate id not reported:\n%s", report)
	}
}

func TestRefreshKeepsManualCorrectionsAndNeedsRevision(t *testing.T) {
	root, l := fixtureRepo(t)
	dir := t.TempDir()
	next := []byte("id,symbol,name\n1,a,Alpha\n2,b,Beta\n")
	_ = os.WriteFile(filepath.Join(dir, "x.csv"), next, 0o644)
	if err := Refresh(&l, "s", root, dir, "main"); err == nil {
		t.Fatal("mutable revision accepted")
	}
	bad := l
	bad.Sources = []Source{l.Sources[0]}
	bad.Sources[0].Files = []File{{Path: "x.csv", SHA256: l.Sources[0].Files[0].SHA256, Bundled: []string{"go/internal/catalog/entitlement-review.csv"}}}
	if err := Refresh(&bad, "s", root, dir, "153e62426eb58b6cd23def535cb4b7e1082e777f"); err == nil {
		t.Fatal("refresh overwrote a manual correction file")
	}
	bad.Sources[0].Files[0].Bundled = []string{"../outside.csv"}
	if err := Refresh(&bad, "s", root, dir, "153e62426eb58b6cd23def535cb4b7e1082e777f"); err == nil {
		t.Fatal("refresh wrote outside the repository")
	}
	if err := Refresh(&l, "s", root, dir, "153e62426eb58b6cd23def535cb4b7e1082e777f"); err != nil {
		t.Fatal(err)
	}
	if p := Verify(l, root); len(p) != 0 || l.Sources[0].Revision != "153e62426eb58b6cd23def535cb4b7e1082e777f" {
		t.Fatal("refresh left the lock inconsistent", p)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "go/internal/catalog/entitlement-review.csv")); string(b) != "id,symbol,name\n1,a,Alpha\n" {
		t.Fatal("manual file changed")
	}
}

func TestFetchRefusesBundledDirAndTruncation(t *testing.T) {
	root, l := fixtureRepo(t)
	s := l.Sources[0]
	if err := Fetch(s, locked, root, "go/internal/catalog", reply(200, "x")); err == nil || !strings.Contains(err.Error(), "bundled directory") {
		t.Fatal("fetch into bundled dir:", err)
	}
	if err := Fetch(s, locked, root, filepath.Join(root, "resources"), reply(200, "x")); err == nil {
		t.Fatal("fetch into bundled dir (absolute) accepted")
	}
	big := strings.Repeat("a", maxFetchBytes+1)
	if err := Fetch(s, locked, root, t.TempDir(), reply(200, big)); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatal("oversized file accepted:", err)
	}
	out := t.TempDir()
	if err := Fetch(s, locked, root, out, reply(200, "id,symbol,name\n")); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshLeavesHeadForCheck(t *testing.T) {
	root, l := fixtureRepo(t)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "x.csv"), []byte("id,symbol,name\n"), 0o644)
	if err := Refresh(&l, "s", root, dir, "153e62426eb58b6cd23def535cb4b7e1082e777f"); err != nil {
		t.Fatal(err)
	}
	if s := l.Sources[0]; s.Status != "unchecked" || s.ObservedHead != "" {
		t.Fatalf("refresh claimed the head: %+v", s)
	}
}
