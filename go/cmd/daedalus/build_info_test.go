package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/Jakoczig/eridion/go/internal/buildinfo"
)

// TestBuildInfoPrintsIdentityWithoutSideEffects (G3L Task 10): --build-info
// prints the linker-embedded identity and touches no profile, window, assets
// or network; it cannot be combined with any other mode.
func TestBuildInfoPrintsIdentityWithoutSideEffects(t *testing.T) {
	home := t.TempDir()
	for _, k := range []string{"HOME", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(k, home)
	}
	saved := [4]string{buildinfo.Version, buildinfo.Revision, buildinfo.Milestone, buildinfo.SourceDate}
	defer func() {
		buildinfo.Version, buildinfo.Revision, buildinfo.Milestone, buildinfo.SourceDate = saved[0], saved[1], saved[2], saved[3]
	}()
	rev := "0123456789abcdef0123456789abcdef01234567"
	buildinfo.Version, buildinfo.Revision, buildinfo.Milestone, buildinfo.SourceDate = "2.2.0-g3l+0123456789ab", rev, "g3l", "2026-11-25T00:00:00Z"
	launch := func(context.Context, string, func()) error { t.Fatal("--build-info launched a desktop"); return nil }
	var out bytes.Buffer
	if err := run(context.Background(), []string{"--build-info"}, nil, launch, &out); err != nil {
		t.Fatal(err)
	}
	var id buildinfo.Identity
	if err := json.Unmarshal(out.Bytes(), &id); err != nil {
		t.Fatalf("not one JSON document: %q", out.String())
	}
	if id.Schema != buildinfo.Schema || id.Version != "2.2.0-g3l+0123456789ab" || id.Revision != rev || id.ShortRevision != rev[:12] || id.Milestone != "g3l" || id.Target.OS != runtime.GOOS || id.Target.Arch != runtime.GOARCH {
		t.Fatalf("identity %+v", id)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Fatalf("--build-info created profile state: %v", entries)
	}
	for _, args := range [][]string{{"--build-info", "--smoke-test"}, {"--build-info", "--data-dir", home}, {"--smoke-test", "--build-info"}} {
		if err := run(context.Background(), args, nil, launch, io.Discard); err == nil {
			t.Fatalf("combined %v accepted", args)
		}
	}
}
