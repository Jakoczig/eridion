# Upstream data conformance

ERIDION bundles reviewed copies of upstream community data. Nothing is downloaded at runtime and nothing updates itself. Every bundled file is pinned in `upstream-sources.lock.json` at the repository root and changes only through the developer tool `go/cmd/upstream-sources`, followed by a reviewed git diff.

## Sources

| Lock id | Upstream | Bundled as | Reuse terms |
| --- | --- | --- | --- |
| `edcd-fdevids` | `EDCD/FDevIDs` CSVs (shipyard, outfitting, commodity, material, microresources) | `go/internal/catalog/*.csv` and byte-identical copies in `resources/data/edcd/fdevids/` | No license file at the pinned revision. These are community-collected Frontier identifiers, used as factual reference data with EDCD attribution kept (see `go/internal/catalog/PROVENANCE.md`). |
| `edcd-eddn-schemas` | `EDCD/EDDN` `schemas/*.json` | `go/internal/providers/schemas/` with the upstream `LICENSE` | BSD-3-Clause; licence file bundled. |
| `silarn-bioscan`, `silarn-explodata` | EDMC-BioScan rule modules, regions, nebula sectors; ExploData `genus.py` | vendored under `go/internal/fieldops/upstream/` (with each LICENSE.txt); `bio-rules.json` and `predicate-support.json` generated from them by `go run ./cmd/bioscan-rules` | GPL-2.0-or-later, GPL-3.0 selected; see docs/testing/bioscan-conformance.md. |
| `coriolis-modules` | Coriolis data (derived stock statistics) | `go/internal/catalog/modules.json` | Unpinned: the inherited derivation never recorded its revision. Game data belongs to Frontier; Coriolis code is MIT. Stock statistics only. ERIDION does not claim Coriolis or EDSY parity. |

## Statuses

`check` reports one status per source:

| Status | Meaning |
| --- | --- |
| `current` | The locked revision equals the upstream default-branch head. |
| `update_available` | The head moved. Run `fetch` and `diff` and review the changes; nothing changes until `refresh`. |
| `unable_to_check` | Offline, a non-200 reply, an unreadable advertisement, or an unpinned source. This is never treated as current. |
| `unchecked` | Written by `refresh`: the new revision has not yet been compared with the upstream head. |

`check` reads only the public git smart-HTTP ref advertisement (`<repo>.git/info/refs?service=git-upload-pack`). It needs no token and makes no API calls. `check -write` records the observed head, status and date in the lock; an `unable_to_check` result never overwrites a recorded status.

## Commands

Run every command from the repository root.

```sh
(cd go && go build -o /tmp/upstream-sources ./cmd/upstream-sources)
/tmp/upstream-sources verify                       # offline
/tmp/upstream-sources check [-write]               # network; -write records the result
/tmp/upstream-sources fetch -source edcd-fdevids -revision <40-hex> -out /tmp/fdevids
/tmp/upstream-sources diff  -source edcd-fdevids -dir /tmp/fdevids      # report only
/tmp/upstream-sources refresh -source edcd-fdevids -dir /tmp/fdevids -revision <40-hex>
```

| Command | What it does |
| --- | --- |
| `verify` | Offline. Every bundled copy must match its locked SHA-256, and source ids must be unique. |
| `fetch` | Downloads the locked files at an immutable revision (40-hex; branch names are refused) into a separate directory: an `-out` inside any directory holding bundled copies is refused. A file over 16 MiB is rejected, never truncated, and each request times out after 30 s. |
| `diff` | Writes a reviewable report and changes nothing. For every file it shows byte changes; for the FDevIDs CSVs it also shows header changes, added, removed and changed IDs, duplicate IDs, and symbols shared by several IDs. |
| `refresh` | Copies the fetched files over every bundled copy and updates the hashes and revision in the lock. It sets the status to `unchecked` (run `check -write` afterwards). It requires a 40-hex revision and validates every target before writing. It refuses paths outside the repository and the manual-correction files listed below. |

## Manual corrections are never overwritten

Upstream files are never edited by hand: `verify`, and `TestUpstreamLockConformance` in CI, fail on any hand edit. Local review decisions live in separate files that `refresh` refuses to write:

- `go/internal/catalog/entitlement-review.csv`: which FDevIDs entitlement values are ordinary public stock and which are excluded from EDDN. An entitlement not listed suppresses the station snapshot until it has been reviewed.
- `go/internal/catalog/inventory-id-review.csv`: per-ID decisions. Its `watermark` row marks the last reviewed catalogue: an ID above it with a generic entitlement (blank or Horizons) is unverified until it has a row here.
- `go/internal/catalog/collisions-reviewed.txt`: the reviewed set of symbols shared by several numeric IDs (for example standard and pre-engineered variants). A new shared symbol fails CI until it is reviewed.
- `go/internal/catalog/PROVENANCE.md`: provenance and licensing record.

## CI enforcement

The "Go tests" step (`go test ./...`, Linux and Windows) runs:

- `TestUpstreamLockConformance` (`go/internal/catalog`): checks that the lock is well formed, with each current source pinned and appearing once. Every bundled copy, both catalogue and resources copies, must match the locked hash (generated-data consistency). Each of the five CSVs must still have the header columns the catalogue parses (schema drift), with no duplicate or empty IDs. The set of shared symbols must equal `collisions-reviewed.txt`.
- `TestAuditModuleIdentityCollisions` and `TestAuditNewHullLabels`: numeric-ID-first resolution, ambiguous shared symbols, and labels for the newer hulls.
- `TestAuditNewIDsNeedReview` and `TestInventoryIDReviewIsWellFormed`: the per-ID review rules.
- `TestBioRulesGeneratedFromPinnedSources` (`go/internal/fieldops`): the BioScan rules and predicate manifest equal the converter's output from the vendored files.
- `go/cmd/upstream-sources` tests: verify, check statuses (via a fake transport, with no network), CSV diff, the fetch and refresh guards.

Cached game-state projections include `catalog.DataDigest()` (a hash of every embedded catalogue and review file), so any data or review change rebuilds them.

No CI step uses the network for upstream data. `check` and `fetch` are run by a developer.

## Refresh procedure

1. Run `check`; if it reports `update_available`, `fetch` the new head into a temporary directory.
2. Run `diff` and read the report: new IDs, renamed items, new entitlement values, new shared symbols.
3. Run `refresh`, then `check -write` and `go test ./internal/catalog ./internal/providers`.
4. Review any new entitlement value, shared symbol or new ID, and record the decision in the manual files. Raise the watermark only after every ID below it has been reviewed.
5. Add a dated note to `PROVENANCE.md`, then commit the data, lock and manual-file changes together.
