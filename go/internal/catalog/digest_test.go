package catalog

import "testing"

func TestDataDigestCoversReviewFiles(t *testing.T) {
	d := DataDigest()
	if len(d) != 16 || d != DataDigest() {
		t.Fatalf("digest %q", d)
	}
	for _, n := range []string{"outfitting.csv", "shipyard.csv", "modules.json", "entitlement-review.csv", "inventory-id-review.csv", "collisions-reviewed.txt"} {
		if _, err := data.ReadFile(n); err != nil {
			t.Errorf("%s not embedded, so not covered by the digest: %v", n, err)
		}
	}
}
