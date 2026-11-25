// Package buildinfo carries the build identity embedded at link time from the
// single generated identity record (scripts/build/go-build-identity.cjs). It
// is read by --build-info, which starts no window, profile or network.
package buildinfo

import (
	"encoding/json"
	"runtime"
	"runtime/debug"
)

// Schema is the identity record schema shared with BUILD-IDENTITY.json.
const Schema = "eridion-build-identity/1"

// Set with -ldflags "-X github.com/Jakoczig/eridion/go/internal/buildinfo.Version=...".
// Unset values mean a local development build without a generated identity.
var (
	Version    = ""
	Revision   = ""
	Milestone  = ""
	SourceDate = ""
)

// Identity is what the executable reports about itself.
type Identity struct {
	Schema        string `json:"schema"`
	Product       string `json:"product"`
	Version       string `json:"version"`
	Milestone     string `json:"milestone"`
	Revision      string `json:"revision"`
	ShortRevision string `json:"shortRevision"`
	SourceDate    string `json:"sourceDate"`
	Target        Target `json:"target"`
	// VCS is the Go toolchain's own record of the checked-out source; it is
	// independent of the linker values and must agree with them.
	VCS VCS `json:"vcs"`
}
type Target struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}
type VCS struct {
	Revision string `json:"revision"`
	Modified bool   `json:"modified"`
	Time     string `json:"time"`
}

// Current returns the embedded identity.
func Current() Identity {
	id := Identity{Schema: Schema, Product: "ERIDION", Version: Version, Milestone: Milestone, Revision: Revision, SourceDate: SourceDate, Target: Target{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	if len(Revision) >= 12 {
		id.ShortRevision = Revision[:12]
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				id.VCS.Revision = s.Value
			case "vcs.modified":
				id.VCS.Modified = s.Value == "true"
			case "vcs.time":
				id.VCS.Time = s.Value
			}
		}
	}
	return id
}

// JSON renders the identity as one indented JSON document.
func JSON() []byte {
	b, _ := json.MarshalIndent(Current(), "", "  ")
	return append(b, '\n')
}
