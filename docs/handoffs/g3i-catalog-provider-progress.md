# G3I catalogue and provider-contract modernization: progress ledger

Statuses: `planned`, `implemented`, `targeted checks passed`, `final CI passed`, `owner acceptance pending`, `owner acceptance confirmed`.

## Pins and setup

| Item | Value |
| --- | --- |
| Base | PR #30, `go/g3h-provider-export-corrections`, `603e9d3f4a55731999a5b7df0e6ab159accf7b23` (draft; both workflows passed) |
| Branch | `go/g3i-catalog-provider-modernization`, isolated worktree created from `603e9d3` |
| Draft PR | targets `go/g3h-provider-export-corrections` (see section 7) |
| Upstream pins | EDCD/FDevIDs `c7c29f7a13df543d4bc8b947eab29f88e06100fa`; EDCD/EDDN schemas `153e62426eb58b6cd23def535cb4b7e1082e777f` (`upstream-sources.lock.json`); elite-kode/elitebgs routes read at `d458e6fc2709e178b0d05bf89169de6d8f71717d` |

At the start, the G3H and G3G heads matched their pins, the working tree was clean, and `603e9d3` descended from `aeb2ca9`. Main was not merged and earlier PRs were not modified.

Environment: Linux x86_64, Go 1.26.10, CGO with SQLite, Node 22.22.1 locally (CI pins 24.22.1). `internal/desktop` is type-checked locally with `GOOS=windows go vet` and runs only in CI. Browser gates ran locally with Chromium 131 headless and `--no-sandbox`, because this host has no usable sandbox; CI keeps the sandboxed launch.

## 1. Tasks

| Task | Status | Commit | Change and evidence |
| --- | --- | --- | --- |
| 16/23 catalogue refresh and identity | targeted checks passed | `919db72` | FDevIDs CSVs refreshed byte-identical to `c7c29f7a`; the resources copies were refreshed too. Historical rows are preserved (0 removed; see section 2). The symbol map is replaced by a numeric-ID-first index plus a symbol→candidate-IDs index. `catalog.ResolveModule(symbol, id, engineered)` returns `exact_id`, `unique_symbol`, `standard_unengineered` (a shared symbol whose standard variant is chosen only when the journal reports no engineering), `ambiguous` (candidate names, no statistics), or `unknown` (future IDs stay unresolved). Engineering is never applied twice: an exact pre-engineered ID keeps its name, class and rating but carries no stock mass or power. Shared symbols are listed in the reviewed `collisions-reviewed.txt`. Tests: `TestAuditModuleIdentityCollisions` (standard vs pre-engineered vs special, SCO, ambiguous display, no statistics leak), `TestAuditNewHullLabels` (all 11 newer hulls). Red evidence: after the refresh, before the index change, `ModuleBySymbol("hpt_beamlaser_fixed_huge")` returned "Overloaded Beam Laser", which `TestAuditModuleIdentityCollisions` caught. |
| 16 EDDN station stock | targeted checks passed | `aa1f012`, `a598636` | `catalog.ClassifyInventory` classifies every shipyard and outfitting item by exact numeric ID as `public`, `excluded`, `unverified`, `unknown` or `ambiguous`, using the manual `entitlement-review.csv` (exact rows win over prefix rows). Excluded items: PowerPlay, Guardian, tech-broker (meta-alloy, caustic, flechette, shock cannon, corrosion-resistant racks), the Cobra Mk IV pre-order, removed items, `Int_PlanetApproachSuite`, and non-module symbols. Any unknown, ambiguous or unreviewed-entitlement item suppresses the whole snapshot. The reason names the counts and entitlements, ending "a partial inventory is never sent". MarketID stays exact. Tests: `TestAuditStationNewInventory` and the G3H `TestAuditPublicInventoryClassification` / `TestAuditStationInventoryComplete`. Red evidence: the G3H allowlist silently dropped `ELITE_V_*` items while still publishing, which is a partial snapshot. |
| 16 model and custom names across views | targeted checks passed | `f50a8a5`, `fa575df` | Workspace loadout modules carry `Identity`, `ModuleID` and `Candidates`. Frontier CAPI module IDs are applied by `paritystate.ReconcileModules` only when the Frontier ship ID equals the journal ship for the same FID (`frontierModuleRefsLocked`, 30 s cache, invalidated on account change). Slot and item must match case-insensitively, and only an exact ID is applied. Commander, Ship, Fleet, statistics and paired displays read the same reconciled workspace. The reducer cache format includes a digest of the embedded catalogue and review files, so cached projections are rebuilt whenever identities can change. Tests: three paritystate module-identity tests and `TestAuditFrontierModuleIDsReconcileVerifiedShipOnly` (another ship, another commander and a mismatched slot are all refused). |
| 17 EliteBGS current lookup | targeted checks passed | `7a9433e` | `count` (history records in v5) is no longer sent; current lookups carry `name` and `page` (1..10) only. Tests: `TestAuditEliteBGSCurrentLookupExcludesHistory` (red: `systems?count=20&...`) and `TestAuditEliteBGSCurrentResponses` (current, empty, error and oversized, pagination). |
| 17 service contracts | implemented | `d9fc46c` | `docs/integrations/service-contracts.md`: methods, parameters, auth, freshness, pagination, errors, rate limits, consent and verification status for Frontier, EDSM, EDDN, Inara, EDAstro, EliteBGS, Canonn, Raven and Spansh. No permission changes. |
| 23 upstream lock and tool | targeted checks passed | `919db72`, `d9fc46c`, `dae6d75` | `upstream-sources.lock.json` and `go/cmd/upstream-sources` (verify, check, fetch, diff, refresh). Statuses: `current`, `update_available`, `unable_to_check`. Refresh requires a 40-hex revision, validates before writing, and refuses manual-correction files and paths outside the repository. `TestUpstreamLockConformance` runs in CI and checks byte drift for both bundled copies, CSV schema drift, duplicate or empty IDs, and the reviewed collision set. Tool tests use a fake transport. Docs: `docs/testing/upstream-conformance.md`. |
| CI filters | done | `964c73a` | Both workflows' `pull_request` filters gain `go/g3h-provider-export-corrections`; no gate changes. |

## 2. Before/after coverage

| Data | G3H (`603e9d3`) | G3I | Notes |
| --- | --- | --- | --- |
| shipyard.csv rows | 38 | 49 | +11 hulls, 0 removed |
| outfitting.csv rows | 1068 | 1243 | +175, 0 removed |
| commodity.csv | 255 | 270 | +15 |
| microresources.csv | 79 | 196 | +117 |
| material.csv | 137 | 137 | unchanged |
| Hulls resolvable and labelled | 38 | 49 | Type-8, Mandalay, Cobra Mk V, Python Mk II, Corsair, Panther Clipper Mk II, Type-11, Caspian Explorer, Kestrel Mk II, Lynx Highliner, Anaconda Mk II |
| Shipyard rows publishable | 37 (11 hulls unknown → suppress) | 38 (adds Python Mk II) | 10 newer hulls unverified → suppress |
| Outfitting entitlement values | PowerPlay renumbered upstream (`POWER_1000xx` → `POWER_2000xx`) | still excluded by prefix | |

EDDN classification of every catalogue row at G3I (after review finding F4):

| Kind | public | excluded | unverified (suppresses) |
| --- | --- | --- | --- |
| shipyard (49) | 38 | 1 (Cobra Mk IV pre-order) | 10 (new hulls other than Python Mk II) |
| outfitting (1243) | 994 | 114 | 135 |

Of the 186 rows added upstream since the old catalogue, every one suppressed any station snapshot containing it at G3H (unknown). At G3I they all resolve for display. For EDDN, 36 are public: the Python Mk II hull, its 5 armours, and 30 SCO frame shift drives (sizes 2–7, ratings E–A, from the Frontier Update 18.02 notes). Another 5 are excluded (Guardian nanite pylons, experimental stabiliser). The remaining 145 stay unverified, and each suppression reason names the item or entitlement. A blank or Horizons entitlement is accepted as public only for IDs up to the reviewed watermark 129001930 (`inventory-id-review.csv`): the refresh added Mk II reward modules and pre-engineered variants that FDevIDs lists with no entitlement.

### Owner decision needed: availability of newer items

Only reviewed items are published. That means the Python Mk II (Frontier, 2024-07-31) and SCO drives sizes 2–7 (Update 18.02 notes). Still unreviewed:

- **Hulls:** Type-8, Mandalay, Cobra Mk V, Corsair, Panther Clipper Mk II, Type-11, Caspian Explorer, Kestrel Mk II, Lynx Highliner and Anaconda Mk II (`ELITE_V_<HULL>`; first sold for ARX as early access), plus their armours.
- **Modules:** SCO size 8; Mk II modules (some are PowerPlay 2.0 or community-goal rewards); pre-engineered variants; Enhanced AX V2 weapons; free fighter and buggy bays; the Mk II fighter bay; `?` entitlements.

Publishing them without evidence could leak commander-dependent unlocks, so any snapshot containing one is suppressed. To lift that, add evidence-backed rows to `entitlement-review.csv` (for an entitlement) or `inventory-id-review.csv` (for an ID). No code change is needed.

## 3. EDDN acceptance and suppression cases (synthetic fixtures)

| Case | Result |
| --- | --- |
| Ordinary modules, including newly added rows, with exact MarketID | published in full |
| Python Mk II hull and armour; SCO FSD 5A (129030592) | published |
| Standard Beam Laser (128049431) | published |
| Overloaded Beam Laser (pre-engineered 129046189, added by the refresh, blank entitlement) | whole snapshot suppressed: "availability not verified for unreviewed item 129046189" |
| Mk II Agile Boost Thrusters (129041074) | whole snapshot suppressed (unreviewed item) |
| PowerPlay, Guardian, tech-broker items (e.g. caustic 128833995), Cobra Mk IV, removed items, planet approach suite | omitted; the rest published |
| Type-8 hull or armour (unreviewed `ELITE_V_TYPE8`) | whole snapshot suppressed; reason names the entitlement |
| Unknown future ID | whole snapshot suppressed; reason counts unknown items |
| Symbol shared by several IDs without an exact ID | whole snapshot suppressed (ambiguous) |

No production uploads were made.

## 4. Independent final review

A separate reviewer examined `603e9d3..964c73a` and confirmed two findings with probe tests.

| ID | Severity | Finding | Resolution |
| --- | --- | --- | --- |
| F1 | high (confirmed) | The Frontier module cache (30 s, keyed by FID) checked the ship binding only when filled. After a ship swap inside the window, the new ship's same-slot module took the old ship's Frontier ID (a Support Focused PD shown on a standard PD, power left out of statistics). | `fa575df`: the cache stores Frontier's ship ID, and the binding to the journal ship is checked on every read. Red→green: `TestAuditFrontierModuleCacheBoundToVerifiedShip`. |
| F2 | medium (confirmed) | A verified CAPI ID unknown to the catalogue (or naming another module) was discarded, so the journal-only guess (`standard_unengineered`, stock statistics) stood. | `fa575df`: such modules become `unknown` with no stock statistics. Red→green: `TestAuditUnknownFrontierIDStaysUnresolved`. |
| F3 | medium (decision) | Most newer hulls and items are unverified, so many snapshots stay suppressed. | Kept (complete-or-suppressed). Reported as an owner decision in section 2 with the exact list; lifted by data rows only. |
| F4 | medium (suspected → fixed) | A blank entitlement was treated as public for IDs added by the refresh, including Mk II reward modules and pre-engineered variants. | `a598636`: per-ID review above watermark 129001930; reviewed SCO 2–7 ordinary, Guardian pylon and experimental stabiliser excluded; the rest unverified. Red→green: `TestAuditNewIDsNeedReview`, station cases for 129046189 and 129041074. |
| F5 | low | Variant identification without CAPI relies on the presence of an `Engineering` block; an engineered standard module with a shared symbol shows as ambiguous. | Kept as a limitation (section 6): ambiguous is safe (no statistics); Loadout-based fingerprinting needs real pre-built Loadout samples. |
| F6 | low | The reducer cache format was bumped by hand. | `dae6d75`: includes `catalog.DataDigest()` over all embedded data and review files. |
| F7 | low | Nothing in production invalidated the Frontier module cache or republished projections. | `fa575df`: `FrontierEvidenceChanged` after each successful Frontier read (webapp). |
| F8–F10 | low | Fetch truncated files over 16 MiB silently, could write into bundled directories, and refresh recorded itself as current head. | `dae6d75`: rejected, refused, and `unchecked` with `check -write`. Tests added. |
| F11, F12 | low | The Retry-After doc did not match the code; `material.csv` header was not checked. | Doc corrected; header added. |
| F13 | info | outfitting/2 carries symbols, so a public pre-built variant would publish the symbol of an excluded standard module. | Mostly resolved by F4: pre-built variants are now unverified (suppress). The symbol semantics are documented in service-contracts.md. |

Checked and found sound by the reviewer:

- ID-first index and the 22 reviewed shared symbols; statistics only for standard IDs; engineering applied once.
- Every station item classified; no path publishes an unverified, unknown or ambiguous item; MarketID exact.
- Account-scoped cache under `s.mu`; `ReconcileModules` is non-mutating.
- EliteBGS bounds; offline verify; check statuses.
- Docs match the allowlist, timeouts, retries and acceptance rules.
- CI change limited to the branch filter.

## 5. Verification

Local, on `964c73a` and again on the post-review head `dae6d75` (Linux): `lint:security`, `lint:go-assets`, `npm test` 205/205, `test:go-assets` 29/29, `build:go-assets`, gofmt clean. `go vet` and `go test` pass on every package except `internal/desktop` and `cmd/daedalus`. `GOOS=windows go vet ./internal/desktop` is clean. `-race` passes on catalog, providers, paritystate, history, gamestate, cmd/upstream-sources and webapp (`-p 1`). `GOOS=windows go vet ./cmd/daedalus` fails locally on `sqlite.Error`/`ErrNotADB`/`ErrCorrupt` without CGO; this is identical on the G3H base `603e9d3` (a cross-vet artifact), and CI builds natively on Windows. Browser gates: all 17 (`g2d` through `g3e`) passed locally against the loopback synthetic fixture. `g2k`, `g2l` and `g2ma` first needed their exact-baseline fixtures, which the CI step builds beforehand (`*-baseline.cjs` followed by `go build ./cmd/visual-fixture` in the baseline checkout). After those were built as in CI, all three passed. On `dae6d75`, g2e, g2h, g2l, g2ma, g2n and g3e were re-run and passed.

## 6. Limitations and external acceptance still required

- Module variants without Frontier data: an engineered standard module whose symbol is shared with a pre-built variant shows as ambiguous (no class or statistics), because the journal cannot distinguish them reliably (review F5).
- Catalogue: FDevIDs carries no licence file. It is used as factual identifiers with EDCD attribution (PROVENANCE.md). The EDDN schemas are BSD-3 with the licence bundled. The Coriolis-derived stock statistics stay unpinned and cover stock modules only; there is no Coriolis or EDSY parity claim, and new-hull stock statistics are not added.
- EDDN, EliteBGS and every provider are fixture-verified or source-verified only, with no live uploads or lookups. Live acceptance needs a deliberate, consented action by the owner.
- Frontier module reconciliation depends on approval of the Frontier application to run against real CAPI data.
- Availability review for newer items (section 2) is an owner decision.

## 7. CI and draft PR

Pending.

## 8. Next batch

G3J: BioScan correctness (Tasks 6/18), then PERF-06–10.
