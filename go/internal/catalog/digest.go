package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"sort"
	"sync"
)

var digestOnce sync.Once
var dataDigest string

// DataDigest identifies the embedded catalogue data and review files. Caches
// of resolved identities include it, so refreshing upstream data or changing
// a review decision rebuilds them (G3I review F6).
func DataDigest() string {
	digestOnce.Do(func() {
		h := sha256.New()
		names, _ := fs.Glob(data, "*")
		sort.Strings(names)
		for _, n := range names {
			b, _ := data.ReadFile(n)
			sum := sha256.Sum256(b)
			h.Write([]byte(n))
			h.Write(sum[:])
		}
		dataDigest = hex.EncodeToString(h.Sum(nil))[:16]
	})
	return dataDigest
}
