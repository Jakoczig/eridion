# G3H provider and export corrections — progress ledger

Statuses: `planned`, `implemented`, `targeted checks passed`, `final CI passed`, `owner acceptance pending`, `owner acceptance confirmed`.

## Pins and setup

| Item | Value |
| --- | --- |
| Base | PR #29, `go/g3g-audit-corrections`, `aeb2ca9687a34417b63cc35484a65d6e59954cb2` (draft, targets G3F; both workflows passed on it) |
| Documentation baseline | `ec89a14dfd52eca55b4653a73f561528d33c8eb6` |
| Branch | `go/g3h-provider-export-corrections`, isolated worktree created from `aeb2ca9` |
| Draft PR | targets `go/g3g-audit-corrections` (link in the PR section below) |

Verified at start: the G3G and G3F heads matched their pins, the working tree was clean, and no G3H branch existed. Earlier PRs were not modified.

Environment: Linux x86_64, Go 1.26.10 (module toolchain line), CGO with gcc and SQLite. Node 22.22.1 locally (CI uses the pinned 24.22.1). GTK4/WebKitGTK are not installed, so `internal/desktop` and `cmd/daedalus` are type-checked locally with `GOOS=windows go vet` and run only in CI. Browser gates run locally with a downloaded Chromium 131 headless; this host has no usable Chromium sandbox, so the local launcher adds `--no-sandbox` (CI keeps the sandboxed launch).

## 1. Independent review of PR #29 (quality gate)

A separate reviewer examined `864c724..aeb2ca9` with emphasis on account switching, source priority, launcher/gameplay permissions, journal history, provider consent and navigation state.

| ID | Severity | Finding | Resolution |
| --- | --- | --- | --- |
| R1 | high (confirmed) | Frontier/Inara snapshots have no observation time, so all their provenance rows share `observed_key=''`. `currentFactEvidence` read that key with `LIMIT 256` and no order, i.e. the oldest 256 rows: after 256 fetches the newest snapshot was never read, the effective value froze and was marked stale, and fresh Frontier permanently lost to the journal (ranks, current ship, credits, fleet count). The external tie-break also compared raw RFC3339 text (`…:00Z` sorted after `…:00.5Z`). | Fixed in `a0ab761`: the window is the newest 256 rows, and external rows compare a normalized fetch instant. Red→green: `TestAuditLatestSnapshotWinsBeyondWindow` (300 snapshots; fractional same-second fetches). |
| R2 | low | A failed `setGoAccounts` rolled back only the settings; `SetDefaultAccount` had already cleared the damaged-preference/unverified-default reasons, so the launcher gate vanished and the unknown default was published (and pinned) as host-authorized until the next validation. | Fixed in `a5e9a9f`: `SnapshotAccounts`/`RestoreAccountsState` restore the reasons before re-applying settings. Red→green: `TestAuditAccountRollbackKeepsRecoveryState` (red pinned the unknown commander). |
| R3 | low | A different query submitted while a search was pending was dropped, and the earlier reply was then discarded, so no search ran for the new text. | Fixed in `a5e9a9f`: only a duplicate of the pending query is blocked. Red→green: "a new query submitted while a search is pending supersedes it". |

Checked and found sound by the reviewer: epoch and lock ordering (game → account → sockets → remote/providers; no inversion), stale-reply handling, launcher RPC/HTTP scope, provider blocked state, source-priority logic itself, append-only migrations, Exosuit retention, identity-bound coordinate resolution.

### Final review of the G3H range

A second independent review covered `aeb2ca9..` the G3H task commits.

| ID | Severity | Finding | Resolution |
| --- | --- | --- | --- |
| F-1 | medium (confirmed) | Entitlement filter blocked only PowerPlay/Guardian/Cobra Mk IV/removed, so tech-broker unlocks (meta-alloy reinforcement, shock cannons, flechette, caustic, corrosion racks) were published as public EDDN stock, leaking a commander's unlocks. | Fixed: allowlist of no entitlement or Horizons planetary landings only. Red→green via the classification test's tech-broker cases. |
| F-2 | medium (decision) | The bundled catalogue predates Type-8, Mandalay, Cobra Mk V, Python Mk II, Corsair, Panther and newer modules, so most current shipyard/outfitting snapshots are suppressed. | Accepted as the plan's mandated complete-or-suppressed behaviour; the suppression reason is shown in the provider status. Lifting it is Task 16/23 (catalogue refresh). |
| F-3 | low | Historical Inara Statistics rows were held with a generic "incomplete fields" reason. | Precise reason now (needs the same session's LoadGame cash and loan). |
| F-4 | low | A 429/503 racing a cancellation was labelled "refused repeatedly". | Reason now states the cancellation. |

## 2. Tasks

| Task | Status | Commit | Change and evidence |
| --- | --- | --- | --- |
| 19 / F25 native CSV exports | targeted checks passed | `57840c4` | The controller accepted only `daedalus-*` names, so both ERIDION exports failed before the dialog. New `internal/fieldexport` holds the approved names, 2 MiB bound and safe write (absolute `.csv` path, create-new, owner-only, fsync, incomplete file removed, `ErrCancelled`), used by the web exports, the controller and the native dialog binding. Tests: `TestAuditFieldExportNames`, `TestAuditFieldExportWrite`, `TestAuditFieldExportsReachNativeBridge` (webapp, content and error propagation), `TestAuditFieldExportThroughController` (desktop, CI only; fails on the old allowlist). |
| 7 / F08 EDSM receipts | targeted checks passed | `f14446f` | Per-event outcomes from the EDSM receipts (1xx confirmed; 2xx/3xx rejected; 4xx rejected as ignorable discard; top-level 2xx rejects the batch; top-level 5xx `accepted_for_later`; uncorrelatable or unreadable replies `unknown_delivery`). Only confirmed events count as sent. Exhausted 429/503 → terminal `failed`, empty queue. The candidate upsert preserves every terminal outcome (it used to let `failed` become eligible again → duplicate delivery). Contract basis: EDSM journal API as cited by the EDMC EDSM plugin (the EDSM page refuses automated fetches). Tests: `TestAuditReceiptMatrix` (8 cases), `TestAuditRefusalExhaustion` (429 and 503, outcome kept after reopening history). |
| 8 / F09 EDDN station stock | targeted checks passed | `a0ed884` | Outfitting kept 8 core-internal families and shipyard a fixed 20-hull list. `catalog.PublicInventorySymbol` classifies each item by exact numeric ID; ordinary stock is published in full; PowerPlay, Guardian unlocks, the Cobra Mk IV pre-order, removed items, `Int_PlanetApproachSuite` and non-module symbols are omitted; one unclassifiable item suppresses the snapshot with an explicit provider-status reason. MarketID stays exact. Tests: `TestAuditPublicInventoryClassification`, `TestAuditStationInventoryComplete`. |
| 8 / F10 Inara credits | targeted checks passed | `a0ed884` | `Current_Wealth` (total assets) was sent as cash. Now cash and loan from the same session's LoadGame plus assets; nothing is sent when cash or loan is unknown. Test: `TestAuditInaraCreditsCoherent`. |
| 17 / F24 Inara contract | targeted checks passed | `c8845ff` | Inara levels kept only within documented bounds (pilot 0–13, navy 0–14) with their `rankProgress` as exact percent text; `ResolveRanks` attaches a provider's progress only to that provider's level from the same snapshot; the rank upload uses the same bounds. Tests: `TestAuditInaraRankNormalization`, `TestAuditInaraRankUploadBounds`, `TestAuditInaraRanksWithProgress`. |
| CI filters | done | see git log | Both workflows' `pull_request` filters gain `go/g3g-audit-corrections` (plan Task 0); no gate changes. |

Not in this batch: Task 17's EliteBGS `count` removal and the service-contract document (`docs/integrations/service-contracts.md`).

## 3. Verification

Local, on the final source commit (Linux, Go 1.26.10, Node 22.22.1): `lint:security`, `lint:go-assets`, `npm test` 205/205, `test:go-assets` 29/29, `build:go-assets`, gofmt clean, `go vet` and `go test` on every package except `internal/desktop`/`cmd/daedalus` (21 packages ok), `GOOS=windows go vet ./internal/desktop` clean, `-race` on catalog, fieldexport, gamestate, history, providers and webapp; browser gates G3E and G2E passed locally. No baseline failures were observed on `aeb2ca9` for these suites.

CI: see section 6.

## 4. External acceptance still required

- EDSM, EDDN and Inara outcomes are verified only against synthetic transports. A real receipt needs a deliberate, consented upload by the owner for a specific account and category; none was made.
- Inara stays behind its application-approval gate; rank/credits contracts are fixture-verified only.
- Native CSV export through the real Windows/Linux save dialog needs a manual check (the controller and write path are tested; the dialog itself is not automated).
- EDDN outfitting/shipyard snapshots for stations selling items newer than the bundled catalogue are suppressed until the catalogue refresh (Task 16/23).

## 5. Remaining tasks and recommended next batch

Next planned batch: BioScan correctness (Tasks 6/18), catalogue and API maintenance (Tasks 16/23, which also lifts the EDDN suppression for newer items), then PERF-06–10. Also open: Task 17 EliteBGS/service contracts, Task 9, Task 10 (build identity; preview artifacts are still named by the older milestone), Tasks 20–22, Task 24 acceptance.

## 6. CI and draft PR

Draft PR: [#30](https://github.com/Jakoczig/eridion/pull/30), `go/g3h-provider-export-corrections` → `go/g3g-audit-corrections`.

Source commit `e80f6f838f9de1f40022d3998c70b90cdff33809` (all code and this ledger before this section), first attempt:

- [Go desktop foundation 45272476660](https://github.com/Jakoczig/eridion/actions/runs/45272476660): success on Linux and Windows (every step, including Go tests and race detector for `internal/desktop` with the new controller export test, native smoke and all browser and visual gates).
- [ERIDION legacy compatibility CI 45272476914](https://github.com/Jakoczig/eridion/actions/runs/45272476914): success.

Unsigned preview artifacts from run 45272476660: `eridion-g3e-Windows-x64-experimental-unsigned-preview` (id 14866144423) and `eridion-g3e-Linux-x64-experimental-unsigned-preview` (id 14865124092). They still carry the older G3E name because build identity is Task 10.

No baseline failures and no new regressions were observed. The docs-only commit adding this section is verified in the PR description.
