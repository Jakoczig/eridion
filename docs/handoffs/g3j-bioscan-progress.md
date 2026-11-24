# G3J BioScan correctness and conformance: progress ledger

Statuses: `planned`, `implemented`, `targeted checks passed`, `final CI passed`, `owner acceptance pending`, `owner acceptance confirmed`.

## Pins and setup

| Item | Value |
| --- | --- |
| Base | PR #31, `go/g3i-catalog-provider-modernization`, `0f465ad7540d99717799b802f1fe1bec28ab3e4f` (draft; both workflows passed) |
| Branch | `go/g3j-bioscan-correctness`, isolated worktree created from `0f465ad` |
| Draft PR | targets `go/g3i-catalog-provider-modernization` (see section 8) |
| Upstream | Silarn/EDMC-BioScan `fc6b85c3de48c8fb79fa9e4f79ad780e48f56629`; Silarn/EDMC-ExploData `fd4e188defd6405e46398fb0a54ac8669c9e2189`. Both were still the upstream default heads on 2026-11-24 (`git ls-remote`; `upstream-sources check` reports `current`) |

Checks at the start:

- The G3I, G3H and G3G heads matched their pins, and G3H is an ancestor of the base.
- Worktrees were clean and no G3J branch existed.
- The draft PR stack #22–#31 was intact.

Main was not used and earlier PRs were not modified.

Environment: Linux x86_64, Go 1.26.10, CGO with SQLite, Node 22.22.1 locally (CI pins 24.22.1). `internal/desktop` is type-checked locally with `GOOS=windows go vet` and runs only in CI. Browser gates ran locally with Chromium 131 headless and `--no-sandbox`, because this host has no usable sandbox; CI keeps the sandboxed launch.

## 1. Tasks

| Task | Status | Commits | Before → after, evidence |
| --- | --- | --- | --- |
| 6 stable genus identity | targeted checks passed | `31f942c` | **Before:** the exploration reducer kept only `Genus_Localised`, and inference matched a symbol or an English name. A German client (`Bakterium`) got **zero** candidates for the same body that gave 13 in English. **After:** bodies carry `genusIds` (journal genus symbols) paired by index with the localized `genuses`. Duplicates deduplicate by symbol, and a missing label falls back to the symbol. Organisms expose `genusId`, `speciesId` and `variantId`. Inference matches symbols only, with no translation table. Red→green: `TestAuditBioLocaleCandidatesEndToEnd` (English vs German through the real service), `TestAuditBioLocaleIdentity` (English, German, missing and repeated labels, copy safety), `TestAuditBioInterruptedCycleIdentity` (language change mid-cycle, interruption by another species). |
| 6 persistence and upgrade | targeted checks passed | `31f942c` | **Before:** any reducer-format change put an upgraded install into `DB_ERROR` with manual recovery. That covers every G3I catalogue change, since the format includes the catalogue digest. **After:** a format mismatch rebuilds every derived view from retained SQLite history at startup, which also restores missing `genusIds`. Checksum or decode failures still go to recovery. The reducer format gains `bio-genus-ids-g3j-1`. Red→green: `TestAuditBioCacheUpgradeRebuildsGenusIDs` (older-format cache, original journal files deleted, restart: same candidates, same sample history, never `DB_ERROR`). This is the one correction touching shared G3I-era behaviour; it was a demonstrated regression for every format bump. |
| 6 / 23 algorithm conformance | targeted checks passed | `ea52b07` | Rules are generated from vendored, pinned upstream files by a literal-data parser (no execution). There are 119 species: 116 ruleset literals plus Bark Mound, Amphora Plant and Radicoida Unicus. Native three-valued evaluator for every rule key, using upstream conversion constants and boundary semantics, with colour (variant) exclusions. Full detail in `docs/testing/bioscan-conformance.md`. |
| 18 stale responses (UI) | targeted checks passed | `e203266` | **Before:** `useFieldRead` started a new request while one was pending, so an older reply could replace a newer one. That reproduced in the harness: all four new cases were red on the old hook. **After:**<br>• One request is in flight; invalidations coalesce into one follow-up.<br>• A reply is applied only if it answers the newest request of the current context (method, body, connection).<br>• A BioScan reply must name the requested body and carry the rule revision already shown for the connection.<br>• Unmount, body change and reconnect reset the context.<br>Test: `tests/service/bioscan-ordering.test.js`, covering a refresh while pending, coalescing, body change, mismatched body, reconnect, rule revision and unmount. |
| 18 stale responses (backend) | targeted checks passed | `ea52b07` | **Before:** `FieldBio` held the service mutex across database reads and full inference. **After:**<br>• The mutex is held only to snapshot and then revalidate the context (commander, system, body); a reply for a context that changed meanwhile is replaced by an empty "refreshed" reply.<br>• Inference runs in a bounded (64-entry) single-flight cache keyed by commander, system, body, rule revision and an evidence digest.<br>• Callers get copies.<br>Tests: `TestAuditBioInferenceSingleFlight` (8 concurrent reads compute once; new evidence recomputes; keys never cross commanders) and `TestAuditBioModuleToggleKeepsIngestion`. Commander switches stay guarded by the existing transport epoch, which rejects stale epochs and closes gameplay sockets. |
| 18 hidden tab / ingestion | targeted checks passed | `ea52b07` | Sample recording, route tracking and FSD handling stay in core journal ingestion and never depend on a BioScan read. Samples recorded while the module is disabled, or before any read, are returned later (`TestAuditBioModuleToggleKeepsIngestion`, `TestAuditBioCacheUpgradeRebuildsGenusIDs`). The panel only mounts when shown and stops polling on unmount. |
| Docs, credits, CI | done | `8cb7339` | `docs/testing/bioscan-conformance.md`; licence credits and `upstream-conformance.md` updated; the module catalogue reports the real rule revision; both workflows' PR filters gain `go/g3i-catalog-provider-modernization`, with no gate changes. |

## 2. Predicate coverage

Every sourced species is in `go/internal/fieldops/predicate-support.json`, which CI regenerates.

| Measure (same method before and after) | Before G3J | G3J |
| --- | --- | --- |
| Sourced species | 116 (hand-converted; 3 `species.py` records missing) | 119 |
| Ruleset literals without a fully supported alternative | 74 / 116 | **51 / 116** |
| `species.py` records without one | not sourced | 2 / 3 (Bark Mound, Amphora Plant; Radicoida Unicus is supported) |
| Natively evaluated keys | atmosphere, body type, gravity, temperature, pressure, `volcanism: None` | all 22 keys: 19 native, `nebula` partial, `regions` and `region` unsupported |
| Colour (variant) dependencies | not evaluated | 65 parent-star, 26 materials, 28 none |

**Unsupported, and why:**

- `regions` and the singular `region` (8 Brain Trees): the ExploData region map was not bundled (provenance). This accounts for 51 of the 53 species. Upstream ignores the singular key, so Brain Trees would pass anywhere; native keeps them unknown.
- Nebula proximity beyond nebula sector names: the reference-star tables were not bundled (Electricae Radialem, Bark Mound).
- Stratum Aranaemus: its upstream ruleset list is empty, so it is reported as unknown ("no sourced criteria").
- Black-hole colour sources: reported as unknown instead of excluded.

Insufficient evidence is always reported as unknown, never as possible. There is no claim of full algorithm parity.

**Conformance defects fixed (red evidence):** the pre-G3J evaluator, extracted from `0f465ad` and run on the same cases, returned "possible" for three cases upstream excludes:

- Tussock Stigmasis at 1012.8 Pa (the audit probe);
- Tussock Stigmasis at 2.7043 m/s²;
- Bacterium Nebulus without its colour element.

The new fixtures pin all three, plus the inclusive and exclusive boundaries.

**Intentional divergences** are listed with reasons in the conformance document:

- missing evidence → unknown;
- an incomplete system scan, or one with fewer retained scans than the FSS count, is not negative evidence;
- the singular `region` key stays unknown;
- `"No volcanism"` is normalized to none.

## 3. G3I owner decision (recorded; G3I unchanged)

Unverified newer ships and equipment remain suppressed from public EDDN inventory publication. Only reviewable, item-specific evidence of ordinary public availability can add an ID to the approved inventory tables (`entitlement-review.csv`, `inventory-id-review.csv`). The complete-or-suppressed policy is retained. Nothing in G3J bypasses it, broadens permissions or makes production uploads. G3I catalogue and provider code were not changed.

## 4. Independent final review

A separate reviewer examined `0f465ad..8cb7339`, compared the evaluator line by line with the pinned `load.py` and ExploData sources, and ran the Go (`-race`) and Node suites. No high-severity findings.

| ID | Severity | Finding | Resolution |
| --- | --- | --- | --- |
| F1 | medium | The singular `region` key (8 Brain Trees) passed, copying an upstream quirk. This was the one place where an uncheckable restriction became "possible". | Fixed: `region` is unsupported and stays unknown, documented as a divergence. Coverage is honestly 53/119 (51/116). |
| F2 | medium | The conformance doc said Brain Trees stay unknown while the code made them decidable. | Fixed with F1; numbers recomputed. |
| F3 | low | The cache format was checked only after a full gob decode, so a future type-changing format would still reach `DB_ERROR`. | Fixed: the format header is decoded first. Red→green: `TestAuditCacheFormatCheckedBeforeFullDecode` (an incompatible older cache gave a gob type error; corrupt bytes still error). |
| F4 | low | The upgrade test could pass through the sample-genus fallback without proving the genus IDs were rebuilt. | Test now asserts the rebuilt exploration view's `genusIds` and localized label. |
| F5 | low | A panic inside inference would leave the single-flight entry open, so later readers would wait forever. | Fixed: waiters are always released and failures are not cached. Test: `TestAuditBioInferenceFailureAndCompleteness` (the old code blocks on the second read). |
| F6 | low | A colour found while an earlier parent star was unscanned was shown as settled. | Fixed: marked provisional in evidence. Test: `TestBioProvisionalColour`. |
| F7 | low | Live spacing could be returned after the live source changed during the read. | Fixed: dropped unless the live state and source are unchanged at revalidation. |
| F8 | low | The context-changed reply emptied a valid view until the next poll. | Fixed: the reply is marked `stale`; the UI keeps the view and reads again (new ordering test). |
| F9 | low | Absence after `FSSAllBodiesFound` was treated as evidence even if retained scans were incomplete (gap, quarantined file, more than 512 bodies). | Fixed: completeness also requires the retained scans to cover the FSS body count (`fssComplete`, tested). |
| F10 | low | The literal parser ignored anything after the literal; nesting depth was unbounded. | Fixed: trailing expressions, later rebinding or mutation of the name, and nesting deeper than 64 are rejected (tests added). |

Checked and found sound by the reviewer:

- **Genus identity:** symbol-only matching, label pairing, organism IDs, and copy safety.
- **Rebuild:** history replay without journal files, untouched sample tables, no startup race; corruption still goes to recovery.
- **Upstream semantics:** all predicates, unit constants, `star_check`, luminosity suffixes, zones, parent derivation, and colour tables. No overlapping prefix keys, so key order is irrelevant.
- **Numbers and pins:** every number in the doc; byte-identical vendored files and their lock hashes; the parser executes nothing.
- **Ordering:** single in-flight with no lost follow-up, error reset, other `useFieldRead` users unaffected.
- **Cache:** key completeness and copy safety.
- **Privacy:** no new private fields reach paired displays.

## 5. Verification

Local, on `8cb7339` and again after the review fixes (Linux):

- `lint:security`, `lint:go-assets`, `build:go-assets`;
- `npm test` 209/209, then 210/210 after the review fixes; `test:go-assets` 29/29;
- gofmt clean; `go vet` and `go test` on all 25 packages except `internal/desktop` and `cmd/daedalus`;
- `GOOS=windows go vet ./internal/desktop` clean;
- `-race` on fieldops, explorestate, gamestate, history, catalog and webapp (`-p 1`);
- `go run ./cmd/bioscan-rules -check` and `upstream-sources verify` / `check`: all sources `current` except `coriolis-modules`, which is unpinned and reports `unable_to_check` as before.

Browser gates: all 17 (`g2d` through `g3e`) passed locally against the loopback synthetic fixture, with the `g2k`, `g2l` and `g2ma` exact-baseline fixtures built first, as the CI step does. After the review fixes, `g2e`, `g2k` (field operations) and `g3e` were re-run and passed.

## 6. Fixture verification versus gameplay acceptance

All results above come from synthetic journals, deterministic fixtures and controlled asynchronous responses. No private gameplay data or live provider calls were used. Real-game observations still needed from the owner:

- **Candidates:** possible, excluded and unknown candidates against bodies where biology is actually found, especially near the gravity and pressure boundaries in the conformance document.
- **Variants:** predicted colour variants against the variant actually logged.
- **Localized client:** a German or French client showing the same candidates as English for the same body.
- **Live cycle:** live sample spacing and history through a real Log, Sample, Analyse cycle, including one interrupted by leaving the body.
- **Upgrade:** an existing installation upgrading from a G3I build. The derived views should rebuild automatically without recovery, keeping samples and analysis history.
- **Displays:** main and paired displays showing the same BioScan information.

## 7. Commits

| Commit | Summary |
| --- | --- |
| `31f942c` | Task 6: stable genus identity; cache-format rebuild from history |
| `ea52b07` | Native BioScan evaluator and generated rules; Task 18 backend |
| `e203266` | Task 18: ordered BioScan reads in the UI |
| `8cb7339` | Docs, credits, module version, CI filters |
| `79b0a7f` | Review findings F1–F10 and this ledger |

## 8. CI and draft PR

Draft PR: [#32](https://github.com/Jakoczig/eridion/pull/32), `go/g3j-bioscan-correctness` → `go/g3i-catalog-provider-modernization`. Both workflows trigger for it (branch-filter change in `8cb7339`).

Source commit `79b0a7f2e1f31f863a7009d6c4ee4d0ecb839c78` (all code; this ledger before this section), attempt 1, no reruns:

- [Go desktop foundation 45371121270](https://github.com/Jakoczig/eridion/actions/runs/45371121270): success on `ubuntu-24.04` and `windows-latest`. This includes Go tests and vet, the race detector, native smoke, and every browser and visual gate. The new `-text` attributes kept the vendored BioScan files and generated rules byte-exact on Windows.
- [ERIDION legacy compatibility CI 45371121165](https://github.com/Jakoczig/eridion/actions/runs/45371121165): success (lint; regression tests on Ubuntu and Windows; client and app builds).

Unsigned preview artifacts from run 45371121270:

- `eridion-g3e-Windows-x64-experimental-unsigned-preview` (id 14910861504)
- `eridion-g3e-Linux-x64-experimental-unsigned-preview` (id 14910162307)

They still carry the older G3E name because build identity is Task 10.

The docs-only commit adding this section is verified in the PR description.

## 9. Next batch

G3K: PERF-06–10 (shared Go revisions, demand subscriptions, normal and low-latency modes, spatial efficiency, measured persistence work).
