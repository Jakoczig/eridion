// Command bioscan-rules regenerates go/internal/fieldops/bio-rules.json from
// the vendored, pinned EDMC-BioScan / ExploData data files. The files are
// parsed as literal data; no Python runs. Run from the go/ directory:
//
//	go run ./cmd/bioscan-rules            write bio-rules.json
//	go run ./cmd/bioscan-rules -check     fail if the committed file differs
//	go run ./cmd/bioscan-rules -manifest  write predicate-support.json
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/Jakoczig/eridion/go/internal/fieldops"
	"github.com/Jakoczig/eridion/go/internal/fieldops/bioconvert"
)

func main() {
	check := flag.Bool("check", false, "compare instead of writing")
	manifest := flag.Bool("manifest", false, "write the predicate-support manifest from the embedded rules")
	flag.Parse()
	if *manifest {
		writeManifest(*check)
		return
	}
	p, err := bioconvert.Convert("internal/fieldops/upstream")
	if err != nil {
		fmt.Fprintln(os.Stderr, "bioscan-rules:", err)
		os.Exit(1)
	}
	raw, err := bioconvert.Marshal(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bioscan-rules:", err)
		os.Exit(1)
	}
	const out = "internal/fieldops/bio-rules.json"
	if *check {
		cur, _ := os.ReadFile(out)
		if !bytes.Equal(cur, raw) {
			fmt.Fprintln(os.Stderr, "bioscan-rules: bio-rules.json differs from the pinned sources; regenerate and review")
			os.Exit(1)
		}
		fmt.Println("ok:", p.Version, len(p.Species), "species")
		return
	}
	if err := os.WriteFile(out, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "bioscan-rules:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", out, p.Version, len(p.Species), "species,", len(p.Genera), "genera")
	fmt.Println("run again to refresh", manifestPath, "from the new rules")
}

const manifestPath = "internal/fieldops/predicate-support.json"

// writeManifest is invoked with -manifest after the rules are current: the
// manifest is derived from the embedded rules of the built binary.
func writeManifest(check bool) {
	raw, err := fieldops.MarshalManifest(fieldops.SupportManifest())
	if err != nil {
		fmt.Fprintln(os.Stderr, "bioscan-rules:", err)
		os.Exit(1)
	}
	if check {
		cur, _ := os.ReadFile(manifestPath)
		if !bytes.Equal(cur, raw) {
			fmt.Fprintln(os.Stderr, "bioscan-rules: predicate-support.json is stale; run with -manifest")
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "bioscan-rules:", err)
		os.Exit(1)
	}
	m := fieldops.SupportManifest()
	fmt.Printf("wrote %s: %d species, %d without a fully supported alternative\n", manifestPath, m.Total, m.WithoutFull)
}
