package atlas

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// nullRootVariants returns correctly shaped spatial manifests whose roots
// array contains JSON null (alone, before and after a valid root).
func nullRootVariants(t testing.TB) [][]byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/spatial-v1/spatial.json")
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	valid := v["roots"].([]any)
	var out [][]byte
	for _, roots := range [][]any{{nil}, append([]any{nil}, valid...), append(append([]any{}, valid...), nil)} {
		v["roots"] = roots
		b, _ := json.Marshal(v)
		out = append(out, b)
	}
	return out
}

// TestAuditSpatialNullRoot (Task 9): "roots":[null] decoded to a zero root and
// the raw-object check type-asserted nil, panicking the parser. It must be
// rejected with the established spatial error instead.
func TestAuditSpatialNullRoot(t *testing.T) {
	for i, b := range nullRootVariants(t) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("variant %d panicked: %v", i, r)
				}
			}()
			if _, e := parseSpatial(b); !errors.Is(e, errSpatial) {
				t.Fatalf("variant %d: want errSpatial, got %v", i, e)
			}
		}()
	}
}

// The same manifest delivered through a correctly signed chain (pointer ->
// distribution -> digest-pinned spatial.json) is refused without installing
// anything; the signature rules are unchanged.
func TestAuditSpatialNullRootSignedInstall(t *testing.T) {
	files, key := spatialFixture(t)
	manifest := nullRootVariants(t)[0]
	var ptr, dist map[string]any
	json.Unmarshal(files["latest.json"], &ptr)
	pp := ptr["payload"].(map[string]any)
	json.Unmarshal(files[pp["distribution"].(string)], &dist)
	dp := dist["payload"].(map[string]any)
	sp := dp["spatial"].(map[string]any)
	sp["bytes"], sp["sha256"] = len(manifest), digest(manifest)
	files[sp["path"].(string)] = manifest
	descriptor := spatialSignTest(dp)
	files[pp["distribution"].(string)] = descriptor
	pp["distribution_sha256"] = digest(descriptor)
	files["latest.json"] = spatialSignTest(pp)

	client := &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		b, ok := files[strings.TrimPrefix(r.URL.Path, "/atlas/spatial/")]
		code := 200
		if !ok {
			code = 404
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(bytes.NewReader(b)), ContentLength: int64(len(b)), Header: http.Header{}}, nil
	})}
	m := newSpatial(context.Background(), t.TempDir(), client, key, time.Now)
	defer m.Close()
	if e := m.Configure(SpatialPreferences{Downloads: true}); e != nil {
		t.Fatal(e)
	}
	if e := m.Update(context.Background(), true); e == nil {
		t.Fatal("signed manifest with a null root installed")
	}
	if st := m.Status(); st.Installed != "" {
		t.Fatalf("installed after rejection: %+v", st)
	}
}

// FuzzAuditSpatialNoPanic: malformed spatial metadata never panics the
// manifest or index parsers. Run bounded: go test -fuzz=FuzzAuditSpatialNoPanic -fuzztime=10s
func FuzzAuditSpatialNoPanic(f *testing.F) {
	raw, _ := os.ReadFile("testdata/spatial-v1/spatial.json")
	f.Add(raw)
	for _, b := range nullRootVariants(f) {
		f.Add(b)
	}
	for _, s := range []string{`null`, `[]`, `{}`, `{"roots":[null]}`, `{"roots":[[]]}`, `{"roots":[{"cell":null}]}`, `{"roots":[1]}`, `{"source":null}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 1<<16 {
			return
		}
		if m, e := parseSpatial(b); e == nil {
			for _, r := range m.Roots {
				_, _ = parseSpatialIndex(b, r, m.RootLY)
			}
		}
		_, _ = parseSpatialIndex(b, spatialRoot{Bytes: int64(len(b)), SHA256: digest(b), Count: 1}, 800)
	})
}
