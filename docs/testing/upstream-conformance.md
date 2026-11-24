# Upstream data conformance

ERIDION bundles reviewed copies of upstream community data. Nothing is downloaded at runtime and nothing updates itself. Every bundled file is pinned in `upstream-sources.lock.json` at the repository root and changes only through the developer tool `go/cmd/upstream-sources`, followed by a reviewed git diff.

## Sources

| Lock id | Upstream | Bundled as | Reuse terms |
| --- | --- | --- | --- |
| `edcd-fdevids` | `EDCD/FDevIDs` CSVs (shipyard, outfitting, commodity, material, microresources) | `go/internal/catalog/*.csv` and byte-identical copies in `resources/data/edcd/fdevids/` | No license file at the pinned revision. These are community-collected Frontier identifiers, used as factual reference data with EDCD attribution kept (see `go/internal/catalog/PROVENANCE.md`). |
| `edcd-eddn-schemas` | `EDCD/EDDN` `schemas/*.json` | `go/internal/providers/schemas/` with the upstream `LICENSE` | BSD-3-Clause; licence file bundled. |
| `coriolis-modules` | Coriolis data (derived stock statistics) | `go/internal/catalog/modules.json` | Unpinned: the inherited derivation never recorded its revision. Game data belongs to Frontier; Coriolis code is MIT. Stock statistics only. ERIDION does not claim Coriolis or EDSY parity. |

## Statuses

`check` reports one status per source:

| Status | Meaning |
| --- | --- |
| `current` | The locked revision equals the upstream default-branch head. |
| `update_available` | The head moved. Run `fetch` and `diff` and review the changes; nothing changes until `refresh`. |
| `unable_to_check` | Offline, a non-200 reply, an unreadable advertisement, or an unpinned source. This is never treated as current. |

`check` reads only the public git smart-HTTP ref advertisement (`<repo>.git/info/refs?service=git-upload-pack`). It needs no token and makes no API calls.

## Commands

Run every command from the repository root.

```sh
(cd go && go build -o /tmp/upstream-sources ./cmd/upstream-sources)
/tmp/upstream-sources verify                       # offline
/tmp/upstream-sources check                        # network, read-only
/tmp/upstream-sources fetch -source edcd-fdevids -revision <40-hex> -out /tmp/fdevids
/tmp/upstream-sources diff  -source edcd-fdevids -dir /tmp/fdevids      # report only
/tmp/upstream-sources refresh -source edcd-fdevids -dir /tmp/fdevids -revision <40-hex>
```

| Command | What it does |
| --- | --- |
| `verify` | Offline. Every bundled copy must match its locked SHA-256, and source ids must be unique. |
| `fetch` | Downloads the locked files at an immutable revision (40-hex; branch names are refused) into a directory **outside** the bundled paths. Each file is limited to 16 MiB and the request times out after 30 s. |
| `diff` | Writes a reviewable report and changes nothing. For every file it shows byte changes; for the FDevIDs CSVs it also shows header changes, added, removed and changed IDs, duplicate IDs, and symbols shared by several IDs. |
| `refresh` | Copies the fetched files over every bundled copy and updates the hashes, revision and check date in the lock. It requires a 40-hex revision and validates every target before writing. It refuses paths outside the repository and the manual-correction files listed below. |

## Manual corrections are never overwritten

Upstream files are never edited by hand: `verify`, and `TestUpstreamLockConformance` in CI, fail on any hand edit. Local review decisions live in separate files that `refresh` refuses to write:

- `go/internal/catalog/entitlement-review.csv`: which FDevIDs entitlement values are ordinary public stock and which are excluded from EDDN. An entitlement not listed suppresses the station snapshot until it has been reviewed.
- `go/internal/catalog/collisions-reviewed.txt`: the reviewed set of symbols shared by several numeric IDs (for example standard and pre-engineered variants). A new shared symbol fails CI until it is reviewed.
- `go/internal/catalog/PROVENANCE.md`: provenance and licensing record.

## CI enforcement

The "Go tests" step (`go test ./...`, Linux and Windows) runs:

- `TestUpstreamLockConformance` (`go/internal/catalog`): checks that the lock is well formed, with each current source pinned and appearing once. Every bundled copy, both catalogue and resources copies, must match the locked hash (generated-data consistency). Each CSV must still have the header columns the catalogue parses (schema drift), with no duplicate or empty IDs. The set of shared symbols must equal `collisions-reviewed.txt`.
- `TestAuditModuleIdentityCollisions` and `TestAuditNewHullLabels`: numeric-ID-first resolution, ambiguous shared symbols, and labels for the newer hulls.
- `go/cmd/upstream-sources` tests: verify, check statuses (via a fake transport, with no network), CSV diff, and refresh guards.

No CI step uses the network for upstream data. `check` and `fetch` are run by a developer.

## Refresh procedure

1. Run `check`; if it reports `update_available`, `fetch` the new head into a temporary directory.
2. Run `diff` and read the report: new IDs, renamed items, new entitlement values, new shared symbols.
3. Run `refresh`, then `go test ./internal/catalog ./internal/providers`.
4. Review any new entitlement value or shared symbol, and record the decision in the manual files.
5. Add a dated note to `PROVENANCE.md`, then commit the data, lock and manual-file changes together.
