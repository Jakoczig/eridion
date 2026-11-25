package buildinfo

import (
	"encoding/json"
	"runtime"
	"testing"
)

func TestCurrentReportsLinkerValuesAndTarget(t *testing.T) {
	saved := [4]string{Version, Revision, Milestone, SourceDate}
	defer func() { Version, Revision, Milestone, SourceDate = saved[0], saved[1], saved[2], saved[3] }()
	Version, Revision, Milestone, SourceDate = "2.2.0-g3l+0123456789ab", "0123456789abcdef0123456789abcdef01234567", "g3l", "2026-11-25T00:00:00Z"
	var id Identity
	if err := json.Unmarshal(JSON(), &id); err != nil {
		t.Fatal(err)
	}
	if id.Schema != Schema || id.Product != "ERIDION" || id.ShortRevision != "0123456789ab" || id.Target.OS != runtime.GOOS || id.Target.Arch != runtime.GOARCH || id.Milestone != "g3l" {
		t.Fatalf("%+v", id)
	}
	Revision = ""
	if Current().ShortRevision != "" {
		t.Fatal("short revision invented for an unstamped build")
	}
}
