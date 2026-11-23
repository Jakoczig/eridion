# G3G audit corrections — implementation progress ledger

Statuses: `planned`, `implemented`, `targeted checks passed`, `final CI passed`, `owner acceptance pending`, `owner acceptance confirmed`.

## Pins (verified 22 November 2026)

| Item | Value |
| --- | --- |
| Native base | PR #28, `go/g3f-galaxy-viewport-slices`, `864c724057a7fe28d9d42f5985fa5e1828d3b030` (open draft) |
| G3E parent | PR #27, `64ddd186f692e48b2e2c1f5a342be20de5d1ac12` (ancestor of G3F, verified) |
| G3D ancestor | PR #26, `79a034d6a65f9d83272f7655f35ed68f70eeecdf` (ancestor of G3E, verified) |
| Documentation publication (`ERIDION_PLAN_COMMIT`) | `ec89a14dfd52eca55b4653a73f561528d33c8eb6` on `main`. The owner supplied the handoff file without its link; the uploaded file is byte-identical to the copy at this commit, which is the only commit touching that path. |
| Implementation branch | `go/g3g-audit-corrections`, worktree created from the native base commit |

Open PR manifest at start: #1–#28 all open drafts; #26/#27/#28 heads match the pins above. No earlier PR was modified, retargeted or rebased.

Only the six planning-packet files were restored from the documentation commit; root README, package and workflow files were not touched.

## Execution environment and local verification limits

- Go: the session image had only Go 1.24.7/1.25.1 and proxy.golang.org was unreachable. Go `1.26.10` was built from the official `golang/go` tag `go1.26.10` (bootstrap 1.25.1). Modules were fetched with `GOPROXY=direct` through a temporary modfile outside the repository (`golang.org/x/sys` replaced by its GitHub mirror at the same version). Repository `go.mod`/`go.sum` are unchanged.
- Not locally testable: `internal/desktop` (GTK4/WebKitGTK headers absent) and `internal/frontend` (needs the real static export; a local placeholder export was used and excluded from git). Both are covered only by CI.
- Node: the npm registry is unreachable, so dependencies could not be installed. Component/socket tests ran under a local-only shim (esbuild in place of Next's Babel, React/ReactDOM 19 in place of 17). With the shim, baseline G3F and this branch show the same 14 environment failures (legacy service modules with missing dependencies, and React 19 rendering differences); every new or changed test passes. Lints (`lint:security`, `lint:go-assets`), the static export and the authoritative Node run are CI-only.
- Baseline at `864c724`: every locally runnable `go test ./internal/...` package passed.

## Slice 1 of project Batch A

| ID | Status | Commit | Evidence / notes |
| --- | --- | --- | --- |
| Task 0 | implemented | `632db6e` | Pins, ancestry and PR stack verified; packet committed |
| Task 1 / F01 | targeted checks passed | `ab984bc` | `TestAuditChronology*`, `TestValidObservedAt`, migration repair and rollback tests. All four chronology regressions fail on `864c724` (credits 100, fuel 4, construction 0.25, rank replaced by Progress 55; equal-time order dependence) and pass here. Schema 7 adds `observed_key`, indexes and a transactional backfill. |
| Task 2 / F02 | targeted checks passed | `814ab1e` | `TestAuditRestoreAllAccounts` (active A, B and a separately pinned account; two restarts) and provider-level `TestAuditRestoreAccountSwitchStaysPaused` (zero transport calls; renewing A does not renew B). Both fail on `864c724`. |
| Task 12 / F15 | targeted checks passed | `827c620` | `TestAuditDefaultCommander` (default B with reader-active A, no transient A display frame, offline restart), migration/recovery cases, ambiguous first-run import, RPC pinning across restart, launcher component tests. Existing tests that assumed single mode follows the journal-active commander now select the commander under test explicitly. |
| PERF-03 | targeted checks passed | `a7da00a` | `TestPerfSidecarVerifiedReuse`: zero envelope decodes across unchanged polls; same-size rewrite still detected. Reading and hashing still happen every poll. |
| PERF-04 | targeted checks passed | `f40b9e1` | `TestPerfStationSingleDecodeAndSave`: one evidence save per authorized attempt (baseline: two); consent-off evidence retained; failed persistence queues nothing. |
| PERF-05 (phase A: Status getter, survey relevance) | targeted checks passed | `c2c7bb5` | Detached `Status` getter test; exploration cache kept for irrelevant continuous events, invalidated by discontinuity/new session/arrival/shutdown/evidence. |
| EffectiveFacts bound | targeted checks passed | `f67f772` | Follow-up to Task 1: only existing field/source groups are queried. |
| PERF-01 | targeted checks passed | `01ef87f` | One `JSON.parse` per incoming frame (baseline: two); zero profiler parses while disabled. |
| PERF-02 (phase A: activity identity) | targeted checks passed | `01ef87f` | Socket state identity kept when the activity flag is unchanged. |

### CI history for draft PR #29

| Head | Legacy CI | Native Linux | Native Windows | Outcome |
| --- | --- | --- | --- | --- |
| `8c7d17b` | pass (5/5) | fail: `TestGameStateTwoFIDSessionsDoNotMix` raced the initial import (test adapted in this branch) | fail: G2P launcher visual, the new default card pushed actions below 680x580 | Fixed in `f1d9946` |
| `f1d9946` | pass | pass (all steps) | fail: G2Q "Current-system markers missing" on attempt 1 and on a rerun of the failed job | Investigated in `f67f772` |
| `f67f772` | pass | fail: pre-existing flake `TestG2MCStartupSurvivesRestartAndUpgradesLegacyCache` | pass (all steps, including G2Q) | See below |

The G2Q failure did not recur after `EffectiveFacts` was bounded to existing field/source groups. Before that change it made up to ~150 indexed lookups per call under the game-state mutex, which is the most plausible cause of the slower Windows runner missing the navigation context. Correlation, not proof. The G2Q failure message now carries diagnostics if it recurs.

`TestG2MCStartupSurvivesRestartAndUpgradesLegacyCache` is intermittent on the untouched baseline: 5 of 60 `-race` runs failed locally at `864c724`, and 2–4 of 60 on this branch. In failing runs the summary is stable and import-complete while the rebuilt workspace lacks startup snapshots, so this is a pre-existing defect in the legacy-cache upgrade path, not a timing-only test issue. Attempted mitigations did not change the rate and were not kept. It needs its own investigation.

### Open items carried from slice 1

- **Task 12 (resolved in slice 2, `2208fed`):** while no default is resolved (several cached commanders, or an unreadable preference file), the provider namespace stayed unpinned and followed the journal-active commander. The plan asks for provider activity to stay blocked until the launcher choice; that needs an explicit blocked state in the provider manager (a sentinel pin would create a stray account namespace). Remote account binding across a host-authorized switch for paired sessions is unchanged and belongs with Task 13.
- **Task 12:** when `g2e` persistence is disabled, the game service has no profile directory while providers still read the preference file (pre-existing asymmetry, unchanged).
- **Task 1:** the Commander component still shows Frontier-labelled ranks from `facts.ranks`; rank/progress presentation and cross-source authority follow in Task 5. `EffectiveFacts` now makes a small bounded set of indexed lookups per call (fields × sources) instead of one grouped query.
- **Pre-existing flake (resolved in slice 2, `dbe3de1`):** `TestG2MCStartupSurvivesRestartAndUpgradesLegacyCache` (see CI history).
- **Review:** this slice had a self-review of the full diff against the plan and the account/privacy boundaries. The independent whole-branch review in Task 24 is still pending.

Remaining Batch A scope after slice 1 was Tasks 3–8, 13–19, immediate parts of 9–11 and 23, and the whole-app performance baseline; slice 2 below covers Tasks 5, 13, 14, 15, 16 (identity part) and the baseline.

## Slice 2 of project Batch A

Environment for this slice: Windows 11 Pro 10.0.26300, AMD Ryzen 7 6800U (8 cores / 16 threads), 28.7 GB RAM, NVMe SSD, WebView2 157.0.4341.60. Toolchain installed for this session: Go 1.26.10 (official zip, SHA-256 verified), Node 24.22.1 (official zip, SHA-256 verified), WinLibs GCC 16.2.0 UCRT (CGO). The clone uses `core.autocrlf=false` so `gofmt`/`standard` match CI. Unlike slice 1, every package (including `internal/desktop` and `internal/frontend` with the real static export) and the authoritative Node suite ran locally.

Local baseline at `6bc4a3c` before any change: `npm ci`, `lint:security`, `lint:go-assets`, `go vet ./...` and `go test ./...` (all 23 packages) passed.

| ID | Status | Commit | Evidence / notes |
| --- | --- | --- | --- |
| Task 5 / F06 | targeted checks passed | `733440c` | `history/source_priority.go`: shared `ResolveEvidence` (tier → eligibility → in-tier newest observation, latest fetch, later stored reference; no provider-name preference), `ResolveRanks` (per category; journal progress only for the same level and not older than that level), `ResolveCurrentShip` (fresh CAPI current ship only). `EffectiveFacts` lost its explicit journal override; facts carry `authority`/`eligible`. Commander overview renders resolved ranks and ship/system identity; `goalCards` prefers fresh Frontier totals and a verified personal Frontier contribution, never global-only participation. `TestAuditSourcePriority` (fresh 100 beats journal 200 with unknown observedAt; valid zero; expired/future fall back; journal beats fresh Inara; other FID ignored), `TestAuditSourcePriorityEqualTier`, `TestAuditRankCoherence` (Frontier 8 + journal 5/55 → 8 with unknown progress), `TestAuditCurrentShipAuthority`; store/rank cases fail on `6bc4a3c`. Component test `tests/service/source-authority.test.js`. Legacy journal-first expectations updated to D0. |
| (memento drift) | targeted checks passed | `2839be7` | PERF-05 (`c2c7bb5`) changed explorestate state without rerunning `scripts/build/generate-go-persistence.go`; regenerated memento bumps `CacheFormat` so old caches rebuild from the journal. |
| Task 14 / F17 | targeted checks passed | `bedee89` | `catalog.VehicleKind` (suit/SRV/fighter/taxi; unknown symbols stay ships). `shipstate` keeps a per-commander memento of the last verified real ship: a suit/vehicle `LoadGame` shows it as last-known (or unknown) and sets `controlled`; vehicle fuel is never ship fuel; non-ship Loadouts and fleet rows ignored. `paritystate` reduces Odyssey suit/loadout events into account-scoped equipment that survives shutdown/gaps/new sessions (grade only from explicit `Class`); Status parses on-foot Health/Oxygen/Temperature/Gravity/weapon before the main-ship guard. `WorkspaceView.Exosuit` via the existing workspace path; live telemetry only from the final known, current-account Status. New `/cmdr/exosuit` tab. Tests: `TestVehicleKind`, `TestAuditSuitIsNotShip`, `TestAuditExosuitKnownEquipment`, `TestAuditOnFootTelemetry`, `tests/service/exosuit.test.js`. |
| Task 15 / F18 | targeted checks passed | `639b734` | `history.ResolvedFleet`: journal ∪ Frontier under D0; fresh Frontier wins reported fields and can re-own a sold/reused ID (sale kept in history); a fresh complete collection marks absent ships `not_currently_reported` (no sale); stale Frontier only fills gaps; witnessed transfers keep their unknown destination; coverage complete/partial/unknown; revision. Fleet page = bounded global search + offset paging bound to the revision (restart on change); `ReconciledFleetCount` = same owned total. Tests: `TestAuditFleetUnion`, `TestAuditFleetFrontierFallbacks`, `TestAuditFleetSearchAndPaging` (0/1/30/31/75, later-page search, revision restart), `tests/service/fleet-completeness.test.js`. |
| Task 16 / F19 (identity part) | targeted checks passed | `bedee89` | A Loadout omitting `ShipName` keeps the same ship's name; explicit empty clears; other ships never inherit. One resolver (`shipIdentityLocked`: fresh Frontier current ship → journal view → durable witnessed rename) used by Commander, account cards and the statistics header. `TestAuditShipNamePreserved`, `TestAuditShipIdentityEverywhere`. |
| Task 13 / F16 | targeted checks passed | `6cd542d` | `webapp/rpc_authority.go`: every dispatched RPC is gameplay (main and paired displays) or launcher management (local launcher only), enforced by one gate. New `getGoProviderGameplay` DTO (id, state, derived `readReady`, selected-commander/public-goal snapshot; no config, credentials, queues, errors). Gameplay components render identically for both roles; terminal has no settings gear/sidebar; launcher gained an Advanced category (data history, built-in modules, provider diagnostics). Automatic stale community refresh remains a main-display side effect (no per-display provider requests); explicit refresh is gameplay. Tests: `TestAuditGameplayParity`, `TestAuditRPCClassificationComplete`, `TestAuditGameplayProviderDTO`, `tests/service/gameplay-parity.test.js` (all 7 fail on the previous components). Legacy owner-only gameplay assertions rewritten; launcher-only, forged-session and foreign-account checks kept. |
| Task 12 follow-up | targeted checks passed | `2208fed` | Explicit `blocked` provider state (no sentinel FID): `PinAccount("")` blocks activation, observation, uploads, station sidecars, reads, `ActivateAccount` and Frontier authorization; status `awaiting_commander`. Startup pins the persisted default/selection; single-commander adoption pins through `gamestate.Options.ProviderAccount`. `TestAuditProvidersBlockedWithoutCommander`, `TestAuditProviderBlockFollowsDefaultResolution`. |
| Flake `TestG2MCStartupSurvivesRestartAndUpgradesLegacyCache` | targeted checks passed | `dbe3de1` | Root cause: restoring a cache that needs a startup rebuild installed the cached summary (`ImportInProgress=false`, `EventsRead=10`) and released the lock before `rebuildDerived` marked the import again; observers saw a "complete" import with empty startup projections. Fixed by reporting the rebuild as an import before releasing the lock. Deterministic regression via a nil-by-default hook in the former gap: `TestG2MCLegacyCacheRebuildNeverLooksComplete` fails without the fix. Locally the original flake did not reproduce (0/120 `-race` runs before the fix); 30 `-race` repetitions of both tests pass after it. |
| Task 11 (baseline only) | measured (baseline) | see git log | See `docs/testing/g3g-native-performance.md`. No candidate comparison, no improvement claims. |

| Task 5 follow-up (journal ranks) | targeted checks passed | `f7d2f8e` | Startup Rank/Progress written before LoadGame never reach commander provenance, so resolved ranks were empty and the Commander page lost journal ranks (G2M-C browser gate). `MergeJournalRanks` adds the live reducer's journal ranks at the journal tier, keeps resolved levels and attaches progress only for the same level. `TestAuditMergeJournalRanks`. |
| CI gate migration | targeted checks passed | `90a7d85` | CRLF-safe `TestAuditRPCClassificationComplete`; browser gates G2E/G2H/G2J/G2O open settings in the launcher, G2M-B checks `/missions/local-activity`, G2G/G2P expect the Advanced category, G2N waits for the adopted default before provider calls. All G2D–G3E browser gates except the CI-only baseline comparisons (G2K/G2L/G2M-A) pass locally. |

### CI history for slice 2

| Head | Legacy CI | Native Linux | Native Windows | Outcome |
| --- | --- | --- | --- | --- |
| `d3885e1` | [pass](https://github.com/Jakoczig/eridion/actions/runs/45080339797) | [fail](https://github.com/Jakoczig/eridion/actions/runs/45080339957): G2E browser gate clicked the removed terminal settings gear | fail: `TestAuditRPCClassificationComplete` could not parse the CRLF checkout of `socket.go` | Both caused by this slice; fixed in `f7d2f8e`/`90a7d85` after reproducing every runnable browser gate locally |
| `90a7d85` | [pass](https://github.com/Jakoczig/eridion/actions/runs/45083157439) (5/5) | [pass](https://github.com/Jakoczig/eridion/actions/runs/45083157467) (all steps) | pass (all steps) | Green on first attempt; the final ledger-only head is verified in the PR description |

### Open items and limitations after slice 2

- **Resolved (Task 13 / contract 4, owner decision 2026-11-23: displays follow host switches):** a host-authorized commander (launcher selection in multi mode, or the resolved default) now moves every trusted grant bound to a verified commander to the new commander (`devices.go` `hostSelectAccount`), with no re-pairing or prompt. Open sockets are still closed on the switch, so no data or pending action from the previous commander survives; the client reconnects within 5 s through `/auth/session`. A grant stranded on another commander by an earlier build is moved by the next host selection. Not moved: grants of unknown ownership (ambiguous legacy grants stay quarantined) and the journal-active fallback used before a default is chosen (`gamestate` reports `authoritative=false`). Tests (shown failing first): `TestHostCommanderSwitchMovesTrustedDisplays`, `TestHostSelectionMigratesStrandedGrant`, `TestHostSwitchKeepsAmbiguousLegacyGrantQuarantined`, `TestLauncherSelectionMovesPairedDisplay` (end to end through `setGoAccounts`), `TestDisplayAccountReportsHostAuthority`.
- Task 14: suit-event payload enrichment for historical rows with `{}` payloads, Frontier suit normalization and numeric suit shield/battery are not implemented. Exosuit equipment lives in the parity reducer (refinement of the planned separate `exosuitstate` package); it is delivered as `WorkspaceView.Exosuit`.
- Task 16: catalogue refresh (38→49 hulls, module ID collision indexes), CAPI/journal module reconciliation and module-class/slot-capacity separation remain open (they depend on Task 23's manifest process).
- Task 15: the owner's exact missing vessel IDs remain a real-account acceptance comparison.
- Task 13: the launcher "Advanced" category reuses the existing components; paired displays still lack the Spansh/planner parity work of Task 20 and key actions of Task 21 (not part of this slice). Account-switch update of paired sessions: resolved above.
- Tasks 3, 4, 6, 7, 8, 18, 19 were not started in this slice.
- `g2e` disabled persistence asymmetry from slice 1 unchanged.

## Measurements

Slice 1 had no desktop environment; its PERF figures are operation counts from targeted tests. Slice 2 recorded the first native baseline on a Windows machine (three repetitions per case): see [`docs/testing/g3g-native-performance.md`](../testing/g3g-native-performance.md). Summary: 100k-event headless cold import 16.3 s wall / 19.6 s CPU (range 16.1–16.6 s); +1k incremental restart 0.84 s; live append to service summary p50 350 ms (dominated by the 500 ms reader poll); launcher idle with WebView2 children 5.2 % of one core, ~429 MB working set, < 4 MB growth over 5 min. No candidate comparison yet, so no improvement claims.
