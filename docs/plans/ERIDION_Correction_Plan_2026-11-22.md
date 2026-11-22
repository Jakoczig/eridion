# ERIDION Audit Corrections Implementation Plan

**Revision 5 — performance tasks recorded for Claude, 22 November 2026:** D0 is Frontier → journal/Status → Inara and all other providers equally. D1 gives every paired display the same trust and gameplay actions as the main display, with no extra per-device game-control opt-in. D2 selects LandingPad. Settings remain launcher-only. Findings F01–F27 and Tasks 0–24 remain the staged implementation scope; decision gates for D0–D2 are closed. This is an implementation plan; application changes have not yet shipped.

> **For agentic workers:** The selected coding recipient is **Claude**. Follow the [portable coding handoff](../handoffs/2026-11-22-claude-coding-handoff.md) and track these checkboxes. Use an installed plan-execution skill if available; the instructions are also executable directly and do not require installing a plugin. The agreed working preference is coherent batches, targeted intermediate checks, and one final branch review/CI/owner acceptance gate per implemented batch; a fresh review agent for each small change is not required.

**Goal:** Correct G3F data, identity, consent, map, export and provider defects; implement the requested commander/display/Exosuit behavior, efficient native updates, full Spansh planning and configured game actions while preserving native operation, retained history and the draft PR stack.

**Architecture:** Keep the single journal ingestion pipeline and serialized SQLite writer. Correct derived projections, account-scoped restore state, map request state and recipient receipts at their existing boundaries; use synthetic transports and fixtures until explicit owner acceptance.

**Tech Stack:** Go/SQLite, native Windows and Linux desktop, React 17/static Next.js export, Node test harness, existing CI and browser fixtures.

**Spec:** The accompanying `ERIDION_Audit_2026-11-22.md` and `ERIDION_Integration_Performance_Roadmap_2026-11-22.md`, including evidence limits, source contracts and confirmed D0–D2. Baseline: [G3F commit 864c724](https://github.com/Jakoczig/eridion/tree/864c724057a7fe28d9d42f5985fa5e1828d3b030). All paths below are repository-relative. New paths are explicitly identified; proposed interfaces may be refined during implementation without changing the product contract.

**Status:** Coding handoff prepared; no native implementation has been performed by this documentation update. Commands below are future execution instructions for the coding session the owner starts. That session may make the scoped source changes and prepare a new draft PR under the handoff; earlier PRs, releases, deployments, production services and private profiles remain protected.

## Documentation status and Claude handoff — Revision 5

The owner requested that the additional performance findings be recorded and the coding work prepared for Claude. This revision publishes the complete planning set in the repository and links the [detailed native performance plan](../superpowers/plans/2026-11-22-native-performance.md) and [Claude coding handoff](../handoffs/2026-11-22-claude-coding-handoff.md). Start with the [planning index](README.md). **Runtime corrections and PERF-01–10 remain unimplemented in this handoff.** Performance opportunities were confirmed by source inspection; no new runtime speedup or G3F benchmark result is claimed.

Keep two baselines distinct:

- **Native implementation source:** G3F `864c724057a7fe28d9d42f5985fa5e1828d3b030`, PR #28, `go/g3f-galaxy-viewport-slices`; still open, draft and unmerged when rechecked on 22 November 2026.
- **Main documentation/configuration baseline:** `ffc5f663c0cfafff1acd41b1af65cc5b4ff056db`. README rewrite `eff5fe7`, inherited-file consolidation `40d07eb`, and package/legacy-workflow correction `ffc5f66` are already published there. Main still lacks the native `go/` tree. These commits do not mark native findings or feature tasks complete.

The original audit observations below describe their pinned source and inspection time. In particular, main's old stable-release workflow has since become an artifact-only legacy packaging workflow; historical native branches retain their own copies. Read the [package/workflow maintenance record](https://github.com/Jakoczig/eridion/blob/ffc5f663c0cfafff1acd41b1af65cc5b4ff056db/docs/maintenance/package-and-workflows.md) before adapting metadata or workflows. Preserve native test scripts and gates; do not merge main into the native stack or replace native workflows wholesale.

## Global constraints

- Preserve the existing draft PR stack; 28 open draft PRs were recorded in the original audit snapshot. Refresh the manifest before coding. Stack proposed `go/g3g-audit-corrections` on `go/g3f-galaxy-viewport-slices` / PR #28 at `864c724057a7fe28d9d42f5985fa5e1828d3b030`.
- Preserve Windows/Linux support, offline access, indefinite user history, exact identifiers, source provenance, account isolation, one journal reader and one SQLite writer. Introduce no Python runtime.
- Preserve the working production Frontier callback, publisher infrastructure and legacy application. Keep secrets in existing OS vault namespaces.
- Replay, restoration and account changes must not synthesize uploads. “After sale” remains hold plus manual review; separately consented live publication remains possible for live-only recipients.
- The optional historical reconciliation wizard is an approved concept. The dormant, approval-gated Inara path for eligible records within 30 days is not automatically a policy breach. Preserve its gate and exact owner review for each batch; neither activate nor remove it by assumption.
- Keep original events/evidence immutable. Append migrations and their tests; never edit an already checksummed migration.
- Use fabricated data and transports for implementation. Existing Frontier read evidence remains valid; new credentials, grant revocation and live publication are separate owner actions.

## Revised source and display contract

1. **Source authority, confirmed D0:** fresh validated Frontier → valid journal/attributable Status → Inara and all other third-party providers at equal priority. This supersedes Revision 2's separately preferred Inara tier. Missing, invalid or expired Frontier fields fall back. Unknown observation time does not disqualify fresh Frontier: show actual fetch/expiry without fabricating observedAt. Keep cooldown/consent; CAPI is a queried snapshot rather than continuous telemetry.
2. **Field/coherence boundaries:** validate account/entity/units first. Resolve nested fields or coherent groups; zero/false are valid. A partial object does not clear unrelated information. Frontier rank 8 cannot inherit progress belonging to journal rank 5. Equal-tier providers use documented freshness/ties, retaining conflict provenance. F01 orders observations within a source; it must not restore the old cross-source preference.
3. **Default account:** single mode automatically uses persisted defaultFID, independent of multi-mode selectedFID and journal import order. Valid cached defaults work offline. Initial ambiguity is resolved once in the launcher; ordinary startup has no repeated account-choice gate.
4. **Gameplay parity and trust, confirmed D1:** main and paired screens have the same trusted-display permissions for selected-account data, navigation and gameplay operations, including planner/import/activation/progress/export, host copy, explicit lookups and configured game actions. Independent page/view choices remain. Host-authorized commander changes update all trusted displays atomically; arbitrary foreign-account requests remain invalid.
5. **Launcher settings:** account/provider configuration, credentials/consent, journal paths, pairing, module enables, data installation and recovery/service tools live in the native launcher. Neither main nor paired terminals have a settings gear/sidebar. Ordinary gameplay operations are equally available on all trusted displays under the same configured account/provider policy; no secondary-only restrictions or extra per-device input opt-in remain.
6. **Exosuit:** show known suit/loadout even offline and attributable live values while on foot. Controlled entity, real main ship and suit identities remain distinct. Unknown metrics stay unknown.
7. **Fleet completeness:** one resolved inventory drives rows, current ship, global search, pagination and Commander count. Keep complete/partial/unknown coverage distinct and preserve historical sales independently of current ownership.
8. **Identity and equipment:** custom ship name, identifier and hull model are separate. All screens use the same resolved identity. Module ID/symbol/variant, size, slot capacity, rating, stock data and observed engineering remain distinct; ambiguous data remain unknown rather than guessed.
9. **Algorithms:** check current upstream source at build/maintenance time and pin reviewed inputs. Track implemented behavior and unsupported predicates separately from source freshness. Runtime uses native code and immutable data; it does not fetch/execute arbitrary plugin code.
10. **Performance modes:** low latency is off by default. Normal mode maintains visible views and dependencies; on keeps finite enabled view models ready. In both, ingestion, durable history, commander/source state, core Status/FSD, organic recording, active route progress and action acknowledgement continue. Hidden animation remains paused; one host shares demand and provider budgets across displays.
11. **Spansh:** full in-app forms/jobs/results, one-click current location and typed route imports. Active selected-account Spansh plans take precedence in Route; actual game NavRoute/target remain separately observed. This replaces the old website-only limit.
12. **Key Controls:** launcher-only bindings/calibration enable explicit typed operational actions equally on main and all paired gameplay screens. First landing gear, then a bounded next-target map/search/select/plot sequence. Pairing establishes display trust; the same globally configured controls are available without another per-device permission step.
13. **Native helpers, confirmed D2:** implement LandingPad as the docking helper, using the launcher catalogue with per-capability enable/disable, evidence/version/provenance and shared services. Broader candidates remain recorded in the roadmap for their own scoped implementation.

## Review focus

1. Older files arriving after newer ones, equal timestamps and fractional timestamps: current facts must be independent of ingestion order (Task 1).
2. A restored nonactive account becoming selected after another restart: no old effective consent revives (Task 2).
3. Delayed spatial/search responses during axis edits or FSD remounts: controls remain usable and stale results never become current (Tasks 3–4).
4. Non-English genus labels and interrupted organic cycles: inference identity and completion remain accurate (Task 6).
5. Mixed acknowledgments, uncertain delivery and exhausted refusals: every candidate has a truthful durable outcome (Task 7).
6. Frontier wins in the resolver and actual UI; missing fields fall back without incoherent rank/progress or ship/loadout combinations (Tasks 5, 15).
7. Default B with reader-active A stays B everywhere; paired gameplay matches local without management controls (Tasks 12–13).
8. On-foot startup cannot turn a suit into a ship; known equipment survives offline, while live telemetry expires correctly (Task 14).

## Execution order and credit budget

| Batch | Priority and purpose | Tasks |
|---|---|---|
| Batch A: correctness | F01–F10/F15–F20/F22/F25; small F21/F24 corrections; native performance baseline | 0–8, 12–19; immediate parts of 9–11 and 23 |
| Batch B: performance and contracts | F23 subscriptions/modes; F11–F14 maintenance; complete implemented-service ledger | Remaining 9–11, 17; shared version/manifest work from 23 |
| Batch C: route and game actions | F26 native Spansh and route model; F27 configurable host actions | 20–21 |
| Batch D: helper coverage | Selected docking helper; full sourced BioScan predicates; native catalogue | 22–23; remaining Task 6 coverage |
| Final gate per implemented batch | One full CI run, whole-branch review and consolidated owner scenario on that batch's final head | 24 |

Finding IDs and priorities match the audit; they are not CVSS scores. P1 denotes immediate correctness or its performance foundation; P2 denotes staged capability/maintenance or a prerequisite before activation. Order A as: branch identity; chronology/restore/default account; shared source/identity model; Exosuit/Fleet/catalog; equal display/settings. Map, receipts, BioScan race and CSV corrections can proceed independently once shared interfaces settle. Establish performance instrumentation early, then finish demand/mode work in B before heavier integrations. Each batch uses Task 24 once at its final head, with reruns only for concrete failures. This stages the expanded scope instead of putting every helper into one enormous G3G change.

### Task 0 — Verify and preserve the branch stack

- [ ] Refresh refs and record the open/draft/base/head manifest before implementation:

```bash
git fetch origin
gh pr list --repo Jakoczig/eridion --state open --limit 100 --json number,isDraft,baseRefName,headRefName,headRefOid
git rev-parse origin/go/g3f-galaxy-viewport-slices
git merge-base --is-ancestor 79a034d6a65f9d83272f7655f35ed68f70eeecdf 64ddd186f692e48b2e2c1f5a342be20de5d1ac12
git merge-base --is-ancestor 64ddd186f692e48b2e2c1f5a342be20de5d1ac12 864c724057a7fe28d9d42f5985fa5e1828d3b030
```

- [ ] Require PR #26 G3D, #27 G3E and #28 G3F to retain those exact heads and their expected stack. Reconcile any drift before applying this plan.
- [ ] After implementation is authorized, create an isolated checkout with `git worktree add -b go/g3g-audit-corrections ../eridion-g3g 864c724057a7fe28d9d42f5985fa5e1828d3b030`. Leave the audit checkout untouched.
- [ ] Check `.github/workflows/go-desktop.yml` and `.github/workflows/ci.yml`: their `pull_request.branches` filters already include the proposed PR base, `go/g3f-galaxy-viewport-slices`. Push filters do not cover G3G; do not assume a push alone runs acceptance. Preserve existing branch coverage and add G3G only if it becomes a base for later work.
- [ ] Open a draft correction PR against G3F when the batch is ready. No force push, autosquash, rebase of earlier PRs, retargeting, merge or release is part of this plan.

### Task 1 — F01: make derived current facts chronological

**Modify:** `go/internal/history/reconcile.go`, `go/internal/history/normalize.go`, `go/internal/history/navigation_evidence.go`, `go/internal/history/migrations.go`, `go/internal/gamestate/accounts.go`, `src/client/components/go-commander-overview.js`. **Create:** `go/internal/history/observation.go`, `go/internal/history/observation_test.go`, `go/internal/history/chronology_migration_test.go`. **Extend tests:** `go/internal/history/reconcile_test.go`, `go/internal/history/g2mb_test.go`, `go/internal/gamestate/persistence_test.go`.

**Interfaces:** Preserve `EffectiveFacts(ctx context.Context, fid, entity string, now time.Time) ([]EffectiveFact, error)` and `FieldNavigationEvidence(fid string) (NavigationEvidence, error)`. Add `validObservedAt(raw string, now time.Time) (time.Time, bool)` in `observation.go` for shared eligibility. Current-value queries must remain indexed/bounded.

- [ ] Add `TestAuditChronologyImportOrder`: import November 22 credits `500`, fuel `32`, construction complete/progress `1.0`, then October 22 credits `100`, fuel `4`, construction incomplete/`0.25`. Reverse import order in a second profile. Assert identical November current values, original row counts and zero publication requests after restart/rebuild.
- [ ] Add equal-time, missing/invalid-time, timestamp fraction, future `now+6s`, two-FID and modified/reimported-source cases. Within journal/same-source chronological selection, a valid plausible observation wins over unknown/future evidence; ties use immutable source identity, byte offset and content hash, never ingestion ID. Keep future/unknown evidence queryable and labelled; retain the existing five-second future tolerance. Cross-source authority is Task 5 and must follow the revised Frontier-first rule.
- [ ] Run `go test ./internal/history ./internal/gamestate -run '^TestAuditChronology' -count=1` from `go/`; record the baseline failure.
- [ ] Replace `max(id)` and latest-`event_id` authority with observation ordering; guard construction UPSERTs. Compare parsed instants or normalized fixed-width UTC values, including external timestamps. Add a transactional migration/backfill for already-derived wrong rows and required indexes. Do not lose independent alternatives or treat fetch time as observation time.
- [ ] Keep Rank levels in `ranks`; introduce distinct `rankProgress` provenance for Progress percentages. A later `Progress.Combat=55` must leave witnessed `Rank.Combat=5` intact. Reclassify retained evidence by its source event during backfill.
- [ ] Run the named tests plus existing reconciliation/persistence tests; assert migration rollback preserves the prior usable database and every original event remains queryable.

Evidence: [baseline reconciliation](https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/history/reconcile.go), audit F01.

### Task 2 — F02: pause effective consent for every restored account

**Modify:** `go/internal/history/restore.go`, `go/internal/history/consent.go`; `go/internal/providers/persistence.go` only where loading requires adjustment. **Create:** `go/internal/history/restore_consent_test.go`. **Extend:** `go/internal/providers/accounts_test.go`, `go/internal/history/recovery_test.go`.

**Interfaces:** Preserve `StageRecovery(ctx context.Context, dir, source string) error`, `StageRestore(ctx context.Context, source string) error`, and `ProviderDisk`. Add `pauseRestoredConsent(tx *sql.Tx) error` in `consent.go`, operating within the staged database transaction.

- [ ] Add `TestAuditRestoreAllAccounts`: backup A/B with different categories and read/upload flags; restore with A selected, B selected and a different persisted account pin. Assert every account configuration is paused with effective reads/uploads disabled, all policies `Never`, exceptions absent and pending candidates held.
- [ ] Run that test and `TestHistoryRestoreUsesPinnedAccountInsteadOfCachedReaderAccount`; the new fixture must expose baseline nonactive-account revival.
- [ ] Sanitize active rows and every `provider-ledger.accounts` entry together; validate bounds, serialize through the shared contract and recompute the ledger checksum. Retain dedupe/history and vault identity; do not migrate or erase credentials.
- [ ] Reopen twice and switch A/B. A counting transport must observe zero calls before explicit renewal; renewing A must not renew B. Run the two affected packages with `-run 'TestAuditRestore|TestHistoryRestore|TestAccountRestart' -count=1`.

### Task 3 — F03/F04: restore FSD geometry and persistent XYZ controls

**Modify:** `src/client/css/go-g3d.css`, `src/client/css/panels/go-navigation-launcher.css`, `src/client/components/go-unified-galaxy.js`, `src/client/components/go-terminal.js`. **Extend:** `tests/service/g3e-map.test.js`, `scripts/testing/g3e-visual.cjs`.

**Interfaces:** Preserve `adjustSlice(frame, camera, axis, direction)` and the shared `GoGalaxy` inspection contract. Separate selected local inspection from verified data readiness; keep camera math unchanged unless the regression proves otherwise.

- [ ] Extend the actual FSD fixture: charge/jump, route/no route, owner/viewer, compact/touch/laptop/desktop. Assert computed grid display, three visible nonzero SVG bounds, square XZ, matching XY width/YZ height, and fixed rectangles through edits. Baseline fails the cascade assertion.
- [ ] Add a delayed spatial-response test: all six XYZ controls survive pending and `{complete:false,pending:false,notice:'Point budget exceeded'}`; a single inverse edit recovers. Superseded results cannot render; the camera stays fixed.
- [ ] Scope CSS fixes to the unified map while covering both stronger legacy selectors. Present loading and unavailable/budget notices separately. Exercise local-only, density-only, both-on and both-off without granting download consent.
- [ ] Reproduce View System navigation while FSD is visible. If the overlay covers the newly selected route, connect that action to the existing manual episode-dismissal policy; preserve inspection on return. Keep the G3F compressed side-view scale explicit in captions; do not silently recenter searches.
- [ ] Run the focused Node test file with `--require ./tests/helpers/no-external-network.cjs`; run the updated visual script once in the consolidated fixture gate, retaining screenshots and geometry JSON.

### Task 4 — F05: bind search success, provenance and actions to the current query

**Modify:** `src/client/components/go-unified-galaxy.js`, `go/internal/webapp/navigation_context.go`, `go/internal/community/community.go`. **Extend:** `tests/service/g3e-map.test.js`, `go/internal/community/community_test.go`. **Create:** `go/internal/webapp/navigation_search_test.go`.

**Interfaces:** Retain `searchSystem(sendEvent, query, role)` and `systemView(ctx context.Context, name string, lookup bool) any`. Add `Resolve(ctx context.Context, name, expectedAddress string) View` to the community manager, using existing consent, cache and cooldown controls.

- [ ] Add `TestAuditSearchMissingCoordinates`: an identity-known journal point without StarPos plus existing owner consent must obtain coordinates through the resolver. A wrong ID must not merge; complete journal coordinates must trigger zero lookup requests.
- [ ] Add UI assertions for success then failure, stale cache/fetch date, changed text while a request is pending, and FSD unmount/remount. Previous results must never remain actionable as the new query's result. Main and trusted paired explicit lookups use the same consent/cooldown policy; consent-off, paused, unpaired and revoked requests do not call the provider. Passive rendering alone does not trigger an extra lookup.
- [ ] Run the new regressions red, then replace name-only success with validated finite coordinates. Keep query/result/source/freshness together and reject superseded completions. Enrich only identity-matching missing coordinates; preserve witnessed positions.
- [ ] Run the focused map file and `go test ./internal/community ./internal/webapp -run 'TestAuditSearch' -count=1`. Preserve single Search/Enter equivalence, whitespace rejection, duplicate guard and useful cooldown messages.

### Task 5 — F06 revised: implement common Frontier-first source authority

**Modify:** `go/internal/history/reconcile.go`, `go/internal/history/frontier_fleet.go`, `go/internal/history/operations_query.go`, `go/internal/gamestate/accounts.go`, `go/internal/providers/reconcile.go`, `go/internal/providers/read.go`, `src/client/components/go-commander-overview.js`, `src/client/components/go-community-goals.js`, `src/client/components/go-provider-reference.js`, `src/client/components/go-fleet.js`. **Extend:** `go/internal/history/reconcile_test.go`, `go/internal/history/frontier_fleet_test.go`, `go/internal/history/goals_snapshot_test.go`, `tests/service/frontier-audit.test.js`, `tests/service/g2n-completion.test.js`. **Create:** `go/internal/history/source_priority.go`, `go/internal/history/source_priority_test.go`.

**Interfaces:** Keep `EffectiveFacts` and `goalCards(page)`; share a named resolver contract for candidate eligibility, source tier, within-source latest value and coherent field groups. Add metadata to effective values rather than making components independently choose authority. Where other providers use domain stores instead of `SaveExternal`, apply the same comparison contract only to equivalent overlapping fields; do not force unrelated entities into commander facts.

- [ ] Implement confirmed D0 in `TestAuditSourcePriority`: fresh Frontier `100` wins over journal `200` even with unknown Frontier observedAt; missing/expired/invalid Frontier falls back to journal, then the equal tier containing Inara and every other provider. Test Inara against an equal-tier peer without a hidden name preference, zero/false, exact IDs, partial fields, same-source chronology and deterministic ties. Preserve alternatives and immutable events.
- [ ] Remove the explicit journal override in addition to changing source scores. Use real fetch/expiry as Frontier freshness eligibility, not as a fabricated observation time. Retain current conservative cooldown/consent; stale cached Frontier must not be labelled live.
- [ ] Add actual component fixture: effective Frontier Combat rank `8` replaces journal rank `5`. Journal progress `55` associated with rank 5 must be unknown for rank 8 unless separately proven compatible. Resolve Commander ship/model/system through the shared view instead of a parallel journal-only path.
- [ ] Correct the original F06 test expectation: fresh CAPI goal total `100` wins over journal `200`; after expiry journal wins. Verified personal Frontier contribution can win under the same order; global-only data cannot create participation/rewards. Missing fields fall back independently with accurate provenance.
- [ ] Replace journal-authoritative/supplemental-only labels and old assertions. Coordinate Fleet with Task 15 and Exosuit with Task 14. Run the affected history/provider tests and named frontend files once after their new focused failures are resolved.

### Task 6 — F07/F21: correct BioScan identity and establish source conformance

**Modify:** `go/internal/explorestate/model.go`, `go/internal/explorestate/g2c.go`, `go/internal/explorestate/persistence.go`, `go/internal/gamestate/fieldops.go`; `go/internal/history/bioscan.go` if historical extraction needs alignment. **Extend:** `go/internal/explorestate/model_test.go`, `go/internal/fieldops/bio_test.go`, `go/internal/history/bioscan_test.go`, `go/internal/gamestate/persistence_test.go`. **Create:** `go/internal/gamestate/bioscan_test.go`.

**Interfaces:** Add `BodyView.GenusIDs []string` with JSON key `genusIds`; retain `Genuses` for localized display. Keep `Infer(scan map[string]any, genera []string) []Candidate`, supplying stable IDs.

- [ ] Add `TestAuditBioLocaleIdentity`: identical scans containing `$Codex_Ent_Bacterial_Genus_Name;` with English, alternate localized and missing labels yield identical candidate symbols/states; the display label remains localized. Baseline must fail the translated case.
- [ ] Deduplicate by ID, preserve labels separately and copy both slices safely. Bump the reducer cache format and rebuild missing IDs from retained journal evidence; never infer identity through a translation dictionary.
- [ ] Test offline upgrade/restart, missing original journal files, partial organic history and interrupted Log/Sample/Analyse cycles in live and historical paths. Keep FID/body/session separation and unknown sample locations.
- [ ] Extend `go/internal/fieldops/bio.go`, `bio-rules.json` and provenance with all 119 source literals, including the three defined in upstream `species.py`. Keep genus+species identity. Pin conversions/comparison boundaries with fixtures, explicitly documenting any intentional divergence from upstream and its evidence. Do not call the existing source pins outdated: both remain current at audit date.
- [ ] Create `docs/testing/bioscan-conformance.md` and a declarative supported-predicate manifest. Mark the 74/116 current species without a fully supported alternative; preserve three-valued possible/excluded/unknown behavior. Full regional/stellar/parent/composition/volcanism/variant coverage follows Task 23 with bounded evidence dependencies. Never claim that a source freshness check proves native equivalence.
- [ ] Run `go test ./internal/explorestate ./internal/fieldops ./internal/history ./internal/gamestate -run 'TestAuditBio|TestG2COrganic|TestPinnedCriteria|TestLiveSpacing' -count=1`. No new first-footfall, bonus or species-confirmation claims.

### Task 7 — F08: persist per-event EDSM receipts and terminal refusals

**Modify:** `go/internal/providers/manager.go`, `go/internal/providers/batch.go`, `go/internal/providers/types.go`, `go/internal/history/publication.go`, `src/client/components/go-provider-settings.js`. **Create:** `go/internal/providers/edsm_receipts.go`, `go/internal/providers/edsm_receipts_test.go`. **Extend:** `go/internal/providers/contracts_test.go`, `go/internal/history/publication_test.go`.

**Interfaces:** Preserve `dispatch(id string)` and `CandidateOutcome(id, status, reason string) error`. Add `edsmEventOutcomes(raw []byte, count int) ([]eventOutcome, error)`; define `eventOutcome` with `Status` and `Reason` strings in the new file. Keep account-read acceptance separate.

- [ ] Add `TestAuditReceiptMatrix`: outer `100`/event `304` yields zero confirmed; events `100,402` yield one confirmed and one rejected. Missing, short, extra or unprovably correlated replies remain unconfirmed. Pin documented positional correspondence; reject detectable identity/order mismatch rather than claiming detection of unobservable reordering.
- [ ] Define durable outcomes `confirmed`, `rejected`, `duplicate`, `accepted_for_later`, `unknown_delivery`; update status validation and terminal-state preservation together. Only genuine confirmation increases sent/category counters. Pin recipient-code meanings and the reviewed supported/discarded event set to primary contract evidence before mapping them.
- [ ] Run receipt tests red, then correlate every result with its original candidate. Preserve dedupe and suppress automatic retries of ambiguous or accepted-for-later delivery. Keep existing EDSM flightlog reads working.
- [ ] Add `TestAuditRefusalExhaustion`: four synthetic 429 responses, then separately four 503 responses, with injected time, leave terminal `failed` receipts and an empty queue; no fifth request and no durable `queued` residue. Cancellation/transport ambiguity retains its distinct outcome.
- [ ] Run `go test ./internal/providers ./internal/history -run 'TestAuditReceipt|TestAuditRefusal|TestG2HProviderPayload|TestG2HRetry|TestPublication' -count=1`; test receipt persistence after restart and accurate UI counters.

### Task 8 — F09/F10: make station snapshots complete; retain the Inara gate

**Modify:** `go/internal/providers/station.go`, `go/internal/providers/adapters.go`, `go/internal/catalog/catalog.go`. **Create:** `go/internal/catalog/public_inventory.go`, `go/internal/catalog/public_inventory_test.go`. **Extend:** `go/internal/providers/station_test.go`, `go/internal/providers/contracts_test.go`, `go/internal/providers/history_wizard_test.go`.

**Interfaces:** Keep `stationPayload(kind string, v map[string]any) ([]byte, error)`. Add `PublicInventorySymbol(kind, symbol string) (canonical string, permitted bool, known bool)` in catalog; distinguish prohibited items from unknown coverage.

- [ ] Add failing inventory fixtures for chaff, shield generator, fuel scoop, ordinary hardpoints/armour, Dolphin and Krait MkII alongside cosmetics and restricted items. Assert complete permitted lists, canonical case and exact MarketID `18446744073709551615`.
- [ ] Derive classification from reviewed licensed symbol/SKU metadata and explicit recipient exclusions. If any ordinary inventory is unclassifiable, suppress the entire category snapshot with an honest reason; never silently publish a truncated inventory. Preserve dock/marker/time/category consent checks.
- [ ] Add an isolated Inara contract fixture: witnessed cash `100`, assets `500`, debt `25` map to distinct fields. Unknown cash/debt must suppress an incoherent credits snapshot; do not substitute wealth or zero. Keep `InaraApplicationName` unset and assert zero default requests.
- [ ] Run targeted catalog/provider tests. Preserve exact historical preview/acknowledgment and live-only recipient restrictions; no application activation or live API test is required.

### Task 9 — Bound discovery and validate two reliability leads

**Modify:** `go/internal/journal/reader.go`, `go/internal/journal/durable.go`, `go/internal/history/sources.go`, `go/internal/atlas/spatial_format.go`, `go/internal/history/restore.go`. **Extend:** `go/internal/journal/reader_test.go`, `go/internal/journal/benchmark_test.go`, `go/internal/atlas/spatial_format_test.go`, `go/internal/history/recovery_test.go`. **Create:** `go/internal/journal/discovery_backlog_test.go`.

- [ ] **Follow-up B / F11:** add more than 4,096 valid journals and mixed-entry fixtures; prove known active appends and rollover continue while discovery advances. Implement paged discovery, bounded active cursors and durable known-source/backfill state; do not delete history or remove resource limits. Restart must resume without duplicate publication. Validate bounded working memory and visible backlog.
- [ ] Add `TestAuditSpatialNullRoot`: a valid-shaped manifest containing `roots:[null]` returns `errSpatial`, never panics. Confirm this suspected path before fixing checked assertions; use a short bounded parser fuzz run. Its trust prerequisite remains signed publisher data.
- [ ] Add `TestAuditRestoreRetry` with cancellation/write failure after staging creation. Establish whether retry is blocked; if confirmed, preserve failed evidence under a distinct safe name and make staged-intent ownership/recovery explicit. Retry and interrupted restart must preserve the original database. Never overwrite an unexplained existing stage.
- [ ] Run `go test ./internal/atlas ./internal/history -run 'TestAuditSpatialNullRoot|TestAuditRestoreRetry' -count=1`. Add `FuzzAuditSpatialNoPanic` in the spatial test file and run it with `-fuzztime=10s`. Defer `TestAuditDiscoveryBacklog` and the journal benchmark to B. Record disproved leads as closed investigations, not fixes.

### Task 10 — Correct dependency, documentation and build identity records

**Completed on main only:** README/inherited-document consolidation and package/lock/legacy CI correction are recorded in `eff5fe7`, `40d07eb`, and `ffc5f66`. Compare those reviewed changes with the native branch before adapting any overlapping item. Native package scripts, native/security jobs and G3F source identity must survive. Native artifact identity, dependency maintenance and native release readiness below remain pending.

**Modify:** `package.json`, `package-lock.json` only for selected fixes; `README.md`, `docs/PROJECT_SUMMARY.md`, `docs/PROJECT-SUMMARY.md`, `scripts/build/package-go-preview.cjs`, `.github/workflows/go-desktop.yml`, `go/cmd/daedalus/main.go`. **Create:** `scripts/build/go-build-identity.cjs`, `go/internal/buildinfo/buildinfo.go`, `docs/testing/g3g-audit-corrections-preview.md`, `docs/maintenance/dependency-triage-2026-11.md`, `docs/maintenance/release-readiness.md`. **Extend:** `tests/build/package-go-preview.test.cjs`, `go/cmd/daedalus/main_test.go`.

- [ ] Record the audited dependency baseline: 45 affected packages, 7 critical/26 high/9 moderate/3 low; 38 development-only and 7 legacy/mixed. Triage advisory reachability separately for build tools, legacy runtime and native static assets. Do not describe this count as 45 proven native exploits.
- [ ] **Follow-up B / F12:** plan compatible updates for reachable axios/ws and related paths, with targeted behavior regressions. Separate the unsupported Next 12.3.7/static-export migration into a tested maintenance change; do not run blind `npm audit fix --force`. Regenerate advisories at execution time and document each disposition.
- [ ] **Immediate / F13:** add package assertions before changing identity: artifact name, README, native Go build revision and packaged `BUILD-IDENTITY.json` agree. Generate identity once from the explicitly checked-out candidate head and package version; derive `2.2.0-g3g+<12-character-sha>`. Include full SHA/platform/milestone and manifest in checksums; Actions consumes this output rather than hardcoding G3E. A workflow's reported head SHA alone does not establish the binary revision: record any merge-test revision separately and verify Go VCS metadata matches the preview manifest.
- [ ] Add linker-populated `buildinfo.Version`, `buildinfo.Revision` and `buildinfo.Milestone`, consumed by a `--build-info` command. Test that it emits the same identity without opening profiles, windows or transports. Use the same generated record for linker values, packaging and Actions outputs.
- [ ] Consolidate current summaries, link historical handoffs, and distinguish native subsets from deferred parity. Run the package fixture and blocking build-script lint.
- [ ] **Follow-up B / F14:** recheck repository protection and required-check policy at execution time. Main’s legacy `release.yml` was changed to artifact-only/read-only in `ffc5f66`; the pinned native branch still carries its older definition. Preserve that distinction and document native release provenance/gates without dispatching a publishing workflow. Adapt only explicitly scoped configuration on the new correction branch; changing repository settings or enabling publication requires a separate owner decision.

### Task 11 — F23: establish native performance and implement both update modes

**Detailed task source:** [Native performance implementation plan — PERF-01–10](../superpowers/plans/2026-11-22-native-performance.md). Its phases A–D are performance phases, distinct from the full project batches A–D above. Small independent performance fixes and the baseline can accompany project Batch A. Shared subscriptions and mode behavior depend on Tasks 5, 12, 13 and 18; preserve the original correctness order and leave persistence redesign evidence-gated. Record each PERF task’s source commit, targeted checks and measurement status.

- [ ] **Performance phase A:** remove disabled-profiler/repeated socket parsing, redundant activity updates, unchanged-sidecar decoding and duplicate station decode/save work; add narrow Status access/relevance changes only with continuity evidence.
- [ ] **Performance phase B:** shared epoch-scoped frontend subscriptions, Go domain revisions, common navigation/provider notifications and normal/low-latency scheduling.
- [ ] **Performance phase C:** stable spatial revisions and unchanged responses, prefetch asset planning, bounded decoded caches, coalesced/reused canvas buffers and route lookup indexing.
- [ ] **Performance phase D:** profile the real durable ingestion/Status path, then implement only justified cache/persistence/lock changes with atomic checkpoint and crash-recovery proof. Record deferred investigations explicitly.

**Effective demand** is visible subscriptions plus the finite launcher-configured low-latency warm set. With low latency off, the last visible consumer releases view-only work. With it on, the enabled warm set remains current even when every gameplay display is hidden/disconnected while the host is running. Disabled modules release warm demand. Both modes keep core ingestion active, avoid hidden painting and share external provider budgets.

**Create:** `docs/testing/g3g-native-performance.md`, `docs/roadmap/native-parity.md`. **Extend:** `scripts/testing/g3e-visual.cjs`, `go/internal/gamestate/persistence_test.go` benchmarks only where measurement needs it.

- [ ] Measure baseline and candidate on the same Windows/Linux machines: launcher/dashboard idle five-minute CPU/RSS, 100k-event cold import, 1k-event incremental restart, live append latency, map interaction, renderer memory, one/two paired displays and reconnect/revocation. Include native process plus WebView children, cache size, request counts, hardware and run variance. A stable view must not repeat spatial queries. Component microbenchmarks alone do not establish application efficiency.
- [ ] Record three measured repetitions per whole-app case, without repeating all CI. Investigate repeatable regression before setting a claimed performance result; preserve raw measurements.
- [ ] Modify `go/internal/webapp/socket.go`, `go/internal/gamestate/g2b.go`, `src/client/components/go-projection.js` and `src/client/components/go-terminal.js` to add explicit replace/release topic subscriptions, connection/account epochs and domain revisions. Compute the union of demand across displays; share one immutable calculation per account/query/revision. Preserve latest-only bounded queues and compact essential Status/FSD updates.
- [ ] Make domain getters avoid unrelated full projections; filter demand before cloning/serialization. Separate static identity/loadout/inference from volatile telemetry/spacing. Cache stable BioScan inputs by body/evidence/rules revision and move expensive bounded work outside the service mutex with revision revalidation. Keep persistent source writes and recovery semantics unchanged unless profiling supports a separately proven checkpoint change.
- [ ] Replace repeat view polling with invalidation/single-flight refresh where revisions exist. Add visibility handling and atomic snapshot+delta resumption. Preserve Operations' pending guard; Task 18 corrects BioScan ordering. Parse socket frames once and avoid unchanged activity-context updates that rerender unrelated components.
- [ ] Add launcher **Low latency: keep views ready**, off by default. Normal mode serves active view dependencies; on maintains finite enabled view models, bounded pages and the relevant map viewport. Losing focus to the game does not hide a visible second-screen dashboard; use actual route/tab/document/native-window visibility, not blur alone. Both retain core ingestion/telemetry/progress and provider budgets. Disabled modules and hidden animation stay inactive. No additional journal reader, store or per-display provider worker is allowed.
- [ ] Create focused demand/revision tests: two screens same view share work; different views both update; the last effective-demand release stops view-only work, while explicit low-latency warm demand remains active with hidden windows; minimize/resume yields latest atomic state; old account/body/connection replies are ignored; route/sample/FSD tracking continues when their tabs are hidden. Measure full native baseline versus normal/low-latency candidates, including p50/p95 event-to-visible and tab-to-fresh-content, before claiming improvements.
- [ ] Keep arbitrary downloadable plugin lifecycle and full overlay as distinct later scope. Native Spansh, controls and relevant helper modules are now concrete Tasks 20–23 rather than indefinite parity notes.
- [ ] Resolve the saved-window shrink contract using `go/internal/desktop/window_state.go` and `go/internal/desktop/window_state_test.go`: 1100×800 on 900×1400 currently becomes 900×800. Read the G3E aspect-preservation requirement, pin the intended result in a test and apply uniform scaling if that remains the contract. This routine contract check does not itself require another approval gate. Keep search-centering and side-view compression acceptance explicit.

### Task 12 — F15: persist and automatically select the default commander

**Modify:** `go/internal/gamestate/accounts.go`, `go/internal/gamestate/service.go`, `go/internal/webapp/accounts.go`, `go/internal/webapp/gamefeed.go`, `src/client/components/go-accounts.js`, `src/client/components/go-companion.js`. **Extend:** `go/internal/gamestate/accounts_test.go`, `go/internal/webapp/accounts_test.go`, `go/internal/webapp/g2n_test.go`, `tests/service/g2p-launcher.test.js`.

**Contract:** extend account settings with an independent `defaultFID`; use one resolved display FID for gameplay, provider pinning and remote association. Multi-mode selection and the default are separate preferences. Single mode currently skips the chooser but clears Selected; fix the missing persistent identity, not just a UI prompt.

- [ ] Add `TestAuditDefaultCommander`: default B, reader-active A, cold start and offline restart must display/pin B with no transient A frame or provider namespace. Later journal events/imports for A do not change the default. Show B's identity in the launcher with no unnecessary Select button.
- [ ] Add preference migration: valid default → prior explicit verified selection → sole verified commander or first verified live identity. When several cached identities remain ambiguous, perform one initial default setup in the launcher; no arbitrary filename/order choice. Preserve corrupt state for recovery and keep effective provider activity blocked until the identity is resolved.
- [ ] Strictly validate the versioned preference schema, retaining a missing default's usable cache and clear launcher recovery. Do not silently broaden upload consent. Test absent legacy settings, corrupt JSON, missing FID and no known commander.
- [ ] Toggle multi mode on/off: preserve the independent default; selecting another profile temporarily in multi mode does not overwrite it. A host-authorized selected-commander change updates the account epoch/context for every trusted display; migrate existing FID-bound grants through that authoritative switch without a new pairing or permission prompt. Reject stale old-account data/actions and arbitrary client-chosen foreign FIDs. Keep mismatch notices when another commander is active in the game.
- [ ] Run the affected account/startup tests and launcher component fixture. Automatic selection must not add an automatic new-window behavior that the request did not require.

### Task 13 — F16: share gameplay content; keep settings in the launcher

**Modify:** `go/internal/webapp/socket.go`, `go/internal/webapp/fieldops.go`, `go/internal/gamestate/fieldops.go`, `src/client/components/go-terminal.js`, `src/client/components/go-companion.js`, `src/client/components/go-commander-overview.js`, `src/client/components/go-operations.js`, `src/client/components/go-colonisation.js`, `src/client/components/go-ship-statistics.js`, `src/client/components/go-provider-reference.js`, `src/client/components/go-field-operations.js`, `src/client/components/go-spansh.js`, `src/client/components/go-parity-ship.js`. **Extend:** `go/internal/webapp/g2mc_test.go`, `go/internal/webapp/accounts_test.go`, `tests/service/g2mc-corrections.test.js`, `tests/service/g2p-launcher.test.js`. **Create:** `go/internal/webapp/gameplay_parity_test.go`, `tests/service/gameplay-parity.test.js`.

**Contract:** define one trusted-display gameplay capability set for main and paired sessions. Separate launcher management RPCs from both shared reads and operational gameplay RPCs currently grouped under owner-only checks. Retain names such as `getGoCommanderOverview`, `getGoOperations`, `getGoConstruction`, `getGoBGSTally`, `getGoItinerary`, `getGoFieldView` and `getGoBioScan` where serializers can be separated. Provider gameplay metadata gets a shared DTO; administrative settings such as `getGoProviders` belong to the launcher surface rather than either gameplay display.

- [ ] Add main/paired fixture comparisons for identity/portrait, balances/ranks, custom ship identity, Fleet/loadouts, Exosuit, carrier, CG, crew/activity, construction, BGS, saved routes and biological sample positions. Assert identical data/source labels and authorized outcomes for route import/activation/progress/export, host copy, explicit provider lookup and configured game actions, for the same host-authorized FID.
- [ ] Remove privacy placeholders and obsolete owner-only or read-only gameplay assertions. Retain launcher-only management, forged/expired session, foreign-FID and revocation tests for all gameplay clients. Separate launcher authority from common trusted-display authority; legacy role labels must not leave paired gameplay with fewer permissions.
- [ ] Remove terminal settings state, header gear/sidebar and management imports for both roles. Relocate remaining configuration/advanced history/provider/module/pairing tools into native `go-companion.js` launcher categories. Preserve legacy `pages/launcher.js`. Update messages to refer to the launcher.
- [ ] Keep simple connection/freshness status in the shared header. Tabs, zoom/XYZ, map inspection, filtering and pagination remain independent. Classify ordinary gameplay workflows separately from launcher configuration and give them equal main/paired access. Both use the same host-side consent, limits and action executor.
- [ ] Enforce bounded gameplay serializers without secrets, raw provider/admin responses, filesystem paths or arbitrary foreign-account enumeration. Existing valid paired sessions inherit equal gameplay/action access without re-pairing. Host-authorized account switches update every trusted session atomically and invalidate prior data/actions; revoked sessions stop receiving data and cannot execute actions.
- [ ] Run shared-read/RPC and actual component tests. Check every gameplay route has no configuration form; confirm two displays do not create additional journal readers or per-display provider polling. Retain the normal one-shot final browser/device gate.

### Task 14 — F17: separate Exosuit, controlled entity and real ship

**Modify:** `go/internal/shipstate/model.go`, `go/internal/shipstate/persistence.go`, `go/internal/history/normalize.go`, `go/internal/history/migrations.go`, `go/internal/gamestate/g2b.go`, `go/internal/gamestate/persistence.go`, `go/internal/gamestate/accounts.go`, `go/internal/paritystate/parse.go`, `go/internal/paritystate/types.go`, `go/internal/providers/reconcile.go`, `src/client/lib/navigation-items.js`, `src/client/components/go-parity.js`, `src/client/components/go-terminal.js`. **Create:** `go/internal/exosuitstate/model.go`, `go/internal/exosuitstate/model_test.go`, `go/internal/exosuitstate/persistence.go`, `go/internal/history/exosuit_enrichment.go`, `go/internal/history/exosuit_enrichment_test.go`, `src/client/components/go-exosuit.js`, `src/client/pages/cmdr/exosuit.js`, `tests/service/exosuit.test.js`. **Extend:** existing ship/parity/gamestate persistence and provider normalization tests.

**Interfaces:** add typed `WorkspaceView.Exosuit` with known equipment, activity context and current on-foot telemetry. Deliver it through existing `getGoWorkspace`/`goWorkspaceChanged` and the coalesced Status update path; extend clones/cache versions and account checks. Do not add another journal watcher or polling loop. Route `/cmdr/exosuit` to one shared component and add sidebar label **Exosuit** under Commander.

- [ ] Add `TestAuditSuitIsNotShip`: valid real-ship Loadout, then LoadGame with FlightSuit or ExplorationSuit_Class3. Neither Commander/card/header nor Fleet treats it as a ship; retain the real ship as last-known. Without prior real-ship evidence show unknown. Cover SRV, fighter, taxi and multicrew contexts; unfamiliar genuine hulls must not be rejected merely because absent from the catalogue.
- [ ] Retain bounded suit/loadout and relevant lifecycle event payloads. Reduce exact suit/loadout/weapon IDs, names/localized labels, reported grade/modifications and weapon slots. Cover full snapshots, switch/create/rename/delete, equip/remove and upgrades; partial deltas do not erase unrelated equipment. Never infer upgraded grade only from a class suffix.
- [ ] Add versioned payload enrichment for existing suit-event rows whose retained raw JSON is `{}`. Reparse only available unchanged original journals matched by source/offset/hash, recording separate enrichment provenance and leaving original events immutable. Ordinary duplicate insertion is insufficient. Test no duplicate counts/publication, missing files and hash mismatch; unavailable evidence stays unknown until a new trustworthy snapshot.
- [ ] Parse supported on-foot Health/Oxygen fractions, temperature, gravity, selected weapon and flags into a separate typed telemetry view before the main-ship-only numeric guard. Keep ship fuel/cargo/pips guarded. Valid Health=.75/Oxygen=.5 render 75%/50%; absent values remain unknown. Live status requires matching selected FID/session/current on-foot context.
- [ ] Embark, shutdown, source loss and account change remove the live claim; known equipment remains available offline. Do not promise numeric battery or suit shield percentages without a verified source contract. Bind telemetry equipment context so a current weapon does not attach to another suit/loadout.
- [ ] Add Frontier suit/loadout normalization only with verified account/entity shape fixtures, then apply Task 5 authority to overlapping fields. Its priority does not invent missing profile objects or authorize new calls. Run targeted reducer/enrichment/Status/component tests; include the shared display equality fixture from Task 13.

### Task 15 — F18: complete owned Fleet and search across its full inventory

**Modify:** `go/internal/providers/reconcile.go`, `go/internal/history/frontier_fleet.go`, `go/internal/history/operations_query.go`, `go/internal/gamestate/accounts.go`, `go/internal/webapp/operations.go`, `src/client/components/go-operations.js`, `src/client/components/go-fleet.js`, `src/client/components/go-commander-overview.js`. **Extend:** `go/internal/history/frontier_fleet_test.go`, existing provider normalization tests and operation/frontend fixtures. **Create:** `go/internal/history/resolved_fleet.go`, `go/internal/history/resolved_fleet_test.go`, `tests/service/fleet-completeness.test.js`.

**Contract:** one resolved fleet snapshot/revision drives current ship, rows, total and global search on both display roles. Carry coverage (complete/partial/unknown), per-field source/freshness and exact `(FID, entity kind, instance ID)`; handle reused ship IDs as distinct ownership history when evidence supports it. Retain both already-supported Frontier array/keyed shapes and bounded collections.

- [ ] Add `TestAuditFleetUnion`: verified current ship plus local/remote stored, in-transit and Frontier-only ships; current-only profile with missing ships collection still includes that current ship and reports incomplete coverage. Two identical models with distinct IDs remain separate. Suits/SRVs/fighters do not count as owned main ships.
- [ ] Apply Task 5 authority to current ownership and fields. Fresh positive Frontier inventory can establish current ownership despite an old sold row with that ID; preserve the old sale as history. A complete validated snapshot can mark an absent vessel not currently reported, without fabricating a sale. Missing/malformed/partial collections must not delete known ships. Never union station shipyard sale stock.
- [ ] Test stale Frontier fallback, explicit empty collection, mixed valid/bad items, map-key/ID mismatch, limits, incomplete modules and unrelated endpoint 204. Record coverage failures instead of silently treating a filtered list as a complete inventory.
- [ ] Replace page-local search with bounded search over the entire resolved fleet. Return total/loaded/coverage and revision with pages; stable cursors or restart-on-revision-change prevent profile refresh from changing positional continuation silently. Reset on filter/account changes.
- [ ] Add 0/1/30/31/75/512 fixtures, all-ID traversal, search for a later-page ship, refresh on page two, historical sale/reused ID, same ID in another FID and offline restart. Commander count equals total resolved inventory, not the current page. Existing stable 75-ship paging remains a regression, not evidence that old paging was universally broken.
- [ ] Run focused fleet/query/component tests and paired equality tests. The owner's exact missing vessel IDs remain a final acceptance comparison; do not claim a synthetic edge case proves the user's precise cause.

### Task 16 — F19/F20: resolve ship names and fitted equipment consistently

**Modify:** `go/internal/catalog/catalog.go`, `shipyard.csv`, `outfitting.csv`, `modules.json` and `PROVENANCE.md` within that directory; `go/internal/shipstate/model.go`, `go/internal/paritystate/parse.go`, `go/internal/providers/reconcile.go`, `go/internal/gamestate/accounts.go`; affected Commander/Fleet/Ship/header components. **Extend:** catalog, shipstate, paritystate and provider tests. **Create:** `go/internal/catalog/identity_test.go`, `tests/service/ship-identity.test.js` and a shared resolved identity DTO in the owning Go package.

- [ ] Define separate custom name, ident, model ID/symbol/label and unavailable reason. Preserve a same-ship known name when a later Loadout omits it; explicit clearing has distinct semantics. Use Task 5 authority and Task 15 account/vessel identity everywhere, including paired screens and planner ship selection.
- [ ] Create numeric-ID and symbol-candidate indexes before refreshing data. Preserve category/mount/ship restriction/variant metadata. Fail a refresh on an unresolved identity collision, changed schema or unintended loss; do not silently overwrite by normalized symbol.
- [ ] Import reviewed immutable current data through Task 23's manifest process. At the audited upstream baseline the coverage comparison is 38→49 hulls, 1,068→1,243 module IDs, 175 additions and five existing label/class/rating changes. Recheck current upstream at implementation time; these counts are dated evidence, not hard-coded future truth.
- [ ] Reconcile CAPI module ID with journal observations only after FID/ship/slot matching. Unknown ambiguous symbols remain unresolved. Update journal-only module-inspector notices and authority tests. Distinguish module class from slot capacity and engineering grade; final observed engineering is not multiplied a second time.
- [ ] Cover standard versus Overloaded Beam Laser sharing a symbol, SCO/pre-engineered/free/prebuilt variants, Mk II hangars, unknown future hull/module IDs, a 5-class module in a 6-capacity slot, missing mass/power data and two accounts reusing ShipID. Naming completeness must not fabricate full statistics coverage.
- [ ] Assert one resolved custom/model name in Commander, header, Ship, Fleet, account cards, planner and paired displays; test offline restart, missing-name Loadout and Frontier-only name. Retain old raw identity/evidence for migration and traceability.

### Task 17 — F24: complete service contract fixtures and targeted corrections

**Modify:** `reconcile.go`, `adapters.go`, `community_reference.go`, `read.go` within `go/internal/providers/`, and relevant Commander presentation. **Extend:** `contracts_test.go`, `community_reference_test.go`, read/reference tests and sanitized `testdata` within that same provider directory. **Create:** `docs/integrations/service-contracts.md` with one record for every row in the expanded roadmap.

- [ ] Normalize Inara ranks into the same typed rank/progress representation used for Frontier/journal. Test actual Commander rendering with only Inara rank evidence; preserve same-source progress and validate pilots 0–13 separately from Navy 0–14. Keep missing data unknown. Preserve F10 cash/assets/debt and Inara approval/identity gates.
- [ ] Remove EliteBGS `count` from current systems lookup; inspect faction/station route semantics before changing their shared builder. History requires its own bounded capability. Verify current page/empty/error envelopes.
- [ ] Record every implemented endpoint's method/encoding, source URL/revision/date, IDs, units, timestamp semantics, nullable/partial fields, pagination, response-level receipts, error/cooldown/retry/cancellation and consent. Cover Frontier, EDSM reads/journal, EDDN, Inara, EDAstro, EliteBGS, Canonn, Raven and Spansh. Public-read and upload scopes are separate records.
- [ ] Preserve EDAstro's persisted one-minute read gate, GEC cache and 429 host cooldown; five-second upload cadence is not a retrieval defect. Review current event-interest support as an explicit consent-bounded subset rather than widening publication automatically. Preserve Raven stats buildId matching and date-time last-change handling; use last-change for invalidation where beneficial.
- [ ] Distinguish valid empty responses, stale cache, partial coverage, authentication failure, malformed response and per-event refusal. No adapter may convert transport success alone into confirmed publication or known complete data.
- [ ] Mark each capability as source/spec reviewed, fixture verified, live accepted or blocked. Use synthetic transports for CI. Actual account/provider acceptance is separately deliberate; retain working Frontier authorization and dormant Inara gate.

### Task 18 — F22: prevent stale BioScan and other view responses

**Modify:** `src/client/components/go-field-operations.js` and the existing shared request/projection layer. **Create:** `tests/service/bioscan-ordering.test.js`; extend any affected account/body query fixture.

- [ ] Turn the demonstrated request-order case into a regression: request one remains pending, refresh fires, newer completion must never be replaced by old data. Use deferred promises and fake time; no real provider calls.
- [ ] Coalesce repeated invalidations with one in-flight query and account/body/query/rules/connection epochs. Cancellation complements epoch validation. Reset correctly on navigation, reconnect and disabled module; preserve unmount cleanup already present.
- [ ] Keep live spacing separate from cached candidates. Verify hidden tabs still record source samples and route progress through core ingestion. Retain Operations' overlap guard and remove only proven unnecessary polling via Task 11.

### Task 19 — F25: repair native CSV exports at the actual bridge

**Modify:** `go/internal/desktop/controller.go`, `go/internal/webapp/fieldops.go`; centralize approved names in an appropriate existing shared boundary. **Create:** `go/internal/desktop/field_export_test.go`; extend native wiring/web integration tests.

- [ ] Make both `eridion-itinerary.csv` and `eridion-private-activity.csv` valid through the actual `Controller.SaveFieldCSV` path. Retain size/name bounds and existing CSV cell protection.
- [ ] Test end-to-end callback wiring for both exports, one successful save, cancelled save and I/O failure. A fake web callback alone is insufficient. Keep account/export acknowledgement requirements and accurate result messages.

### Task 20 — F26: native Spansh planning, typed routes and display precedence

**Modify:** `go/internal/fieldops/route.go`, route/history persistence, `go/internal/webapp/fieldops.go`, provider allowlists, `src/client/components/go-spansh.js`, `src/client/components/go-terminal-route.js` and shared navigation projection. **Create:** a bounded planner adapter/service in `go/internal/providers/spansh_planner.go`, its fixtures/tests, versioned route migration and `docs/integrations/spansh-planning.md`. Extend route/component acceptance fixtures.

- [ ] Define routeID/kind/revision and stable stopID/ordinal with exact system ID/name plus optional body/station/market/trade/carrier details. Preserve A→B→A, multiple bodies in one system, refuel stops and trade actions. Migrate current plans without dropping progress. Exact 64-bit IDs remain strings at the JS boundary.
- [ ] Add **Use current system** from the selected commander's resolved location, with source/freshness and unknown state; do not overwrite manual input continuously. Distinguish carrier origin where applicable. Validate ship/FSD/range assumptions from resolved equipment; unknown range requires explicit input.
- [ ] Build complete in-app forms/results by supported planner family. Start neutron/galaxy, then riches/exobiology/carrier, then trade/tourist formats. Record unsupported modes honestly until tested. Reference pinned SpanshTools/NeutronDancer source, preserving source/license records.
- [ ] Keep published read API and source-observed planner contracts separate. Validate endpoint host/path/method/encoding and sanctioned usage before shipping; do not weaken a generic allowlist. One shared asynchronous job handles progress, bounded polling/backoff, cancellation, offline/resume and late-result rejection by account/input hash/request revision. Avoid blind POST retry after uncertain submission.
- [ ] Route tab chooses the active usable Spansh plan first, otherwise game NavRoute, otherwise no route. A completed active plan stays visibly complete until ended/suspended. Keep actual game target/route separately labelled. Copying/selecting does not advance progress; matching arrivals, objective events or explicit manual completion do.
- [ ] Cover direct and queued results, invalid/empty/malformed/partial data, 429/Retry-After, 5xx, timeout, cancellation, account switch, two displays viewing one job and repeated stops. Show results without resubmission on tab switching. One explicit action from any trusted main/paired display activates a reviewed plan; provider polling is not multiplied by displays.
- [ ] Keep manual website/import/export and host-clipboard fallback. No test submits a personal location or live route job without an intentional later acceptance action.

### Task 21 — F27: launcher Key Controls and deliberate game actions

**Create:** typed action/service and platform backends under `go/internal/gamecontrol/`, bounded bindings parser, fake-input tests, launcher Key Controls UI and platform capability report. **Modify:** existing desktop bridge, gameplay RPC checks, Ship actions and next-stop controls. Full behavior is specified in the expanded roadmap.

- [ ] Read supported Elite binding presets without modifying game files. Allow explicit override, conflict/missing-binding display, calibration and emergency cancel in the launcher. Separate scan codes/key identity from text entry and keyboard layout. Joystick-only bindings require a configured supported alternative.
- [ ] Expose only typed `landingGear.toggle` and `route.plotNext`-style actions, resolved on the host against current account/context/route/stop and operation ID. Never accept arbitrary shell commands or unrestricted remote key sequences. One active operation; duplicate/stale requests do not repeat input.
- [ ] Handle deliberate focus handoff because clicking a local ERIDION window removes game focus. Validate the target game window before each step; cancel on loss, exit, account/route change, revocation, timeout or manual stop. Release only keys owned by the operation; respect existing user-held modifiers.
- [ ] Landing gear sends one configured toggle and waits for fresh matching Status evidence. Timeout is unconfirmed, not permission to toggle again. Disable actions in unsuitable entity/UI context; retain ordinary display information even if input is unavailable.
- [ ] Implement **Copy to game / Plot next target** beside the active next stop: reserve exact stop; focus game; open map; search exact system; select; plot; require fresh post-action NavRoute evidence whose destination matches the intended stop before reporting a successful plot. The immediate FSD target can be an intermediate hop; target selection alone is not plot confirmation. Journal does not expose every UI step, so calibration and bounded waits are required. If fresh matching selection evidence exists but route evidence does not, report target-selected/plot-unverified. Otherwise report input-sent/selection-and-plot-unverified, or the actual failure/cancellation state. No automatic next jump follows.
- [ ] Validate Windows input integrity/modifier constraints and supported Linux X11/Wayland/Proton path separately. Wayland portal consent/capability is platform-mediated; no universal injection claim. Provide host clipboard/manual fallback for unsupported environments.
- [ ] Implement confirmed D1: every valid paired session can invoke the same configured game actions as the main display by default. Do not add an extra per-device enable/approval switch or require re-pairing. Use one host executor and identical context checks for all initiators; test simultaneous commands from two displays, duplicate operation IDs, session revocation and global control disable. Launcher configuration remains separate from gameplay screens.
- [ ] Test fake focus loss, double click, stale target, mismatched game state, held modifiers, operation cancellation, permission revocation and missing acknowledgement. Include a fixture where the target changes but plotting fails. Native gameplay acceptance verifies actual gear/plot result on each claimed platform; an OS API success or selected target alone is insufficient.

### Task 22 — F27: implement LandingPad natively

**Create:** optional event reducer/layout data under `go/internal/docking/`, shared gameplay projection/component and tests. Add a launcher module toggle and current capability/provenance entry.

- [ ] Implement confirmed D2: LandingPad's assigned-pad diagram is the selected docking-helper behavior. Use the identified source/layout catalogue and preserve provenance. Docking Denied Reason remains a separate catalogue candidate, not part of this selected helper's required parity.
- [ ] For LandingPad behavior, reduce DockingGranted with assigned pad and supported station/carrier type; clear/hide on docking, cancellation, timeout, jump, shutdown/menu/session changes. Unknown layout shows unknown rather than a guessed diagram. Reuse shared journal ingestion and event revisions; no idle provider loop.
- [ ] Cover starport/carrier variants, unknown pads, revoked docking permission, reconnect and two displays. Native source/assets retain applicable notices. This helper provides docking information; automatic flight control is outside the requested action scope.

### Task 23 — F21 and native capability maintenance: upstream checks and parity

**Create:** `upstream-sources.lock.json`, builder check/refresh commands, `docs/roadmap/native-capabilities.md`, `docs/testing/upstream-conformance.md` and per-source fixture manifests. **Modify:** build/data scripts, package metadata/CI as appropriate and launcher native-module catalogue. Paths for new scripts should follow the existing build-script conventions.

- [ ] Record repository/path, current release and default head, selected immutable revision, raw/normalized hashes, check time, license/notice, converter/schema version and implemented/unsupported capabilities. Current baseline pins and catalog counts are listed in the roadmap. A release tag and newer default head are not the same revision.
- [ ] Builder checks official upstream endpoints with bounded requests, conditional caching/rate handling; reports current/update-available/unable-to-check separately. Local offline builds reproduce locked inputs. Release preparation requires a dated check or a documented offline exception; never silently label an offline build latest.
- [ ] Generate reviewable changes for IDs/labels/variants, new rule keys/species, algorithm outputs and licenses. Refresh immutable data only after schema/collision/conformance checks. Runtime remains native and does not auto-execute Python plugins or fetch unreviewed algorithms.
- [ ] Complete remaining BioScan predicates/variants with necessary regional/stellar/parent/composition evidence and explicit unknowns; compare native outputs against pinned source fixtures and observed boundary cases. Share candidate caches and core sample history. Validate ship-statistic models separately from naming completeness.
- [ ] Maintain the capability catalogue in the roadmap: navigation/docking, exploration/BioScan/Pioneer/ExploData, route display, BGS/Powerplay, colonisation, surface/carrier navigation and engineering candidates. Per-capability launcher toggle must stop its private computation/network work while retaining user history and shared ingestion needed elsewhere. Avoid duplicate route/provider/database engines.
- [ ] Check per-file/source reuse terms and attribution. Coriolis data are not blanket MIT; EDSY has distinct noncommercial/source conditions; FDevIDs metadata does not itself grant a license. Prefer compatible native implementations and properly sourced data. Do not infer permission from a registry label alone.

### Task 24 — One final CI, review and owner acceptance gate per implemented batch

**Create:** `docs/testing/g3g-audit-acceptance.md`, containing candidate SHA, result links, artifact hashes, limitations and owner decisions.

Use this gate once for each implemented batch, with that batch's actual scope and final SHA. Later batches get their own acceptance record; a green Batch A cannot claim unimplemented B–D capabilities. D0–D2 are confirmed and create no remaining decision gate.

- [ ] Perform one independent whole-branch review against this plan and privacy/account boundaries. Confirm no prior PR heads, callback, publisher or legacy behavior changed unintentionally. Keep the correction PR draft.
- [ ] Run both existing workflows once on the final candidate head: all native Windows/Linux jobs and five legacy CI jobs, blocking lints, Go tests/vet/race, static export, native smoke, existing regressions and the extended map fixture. Preserve meaningful existing gates; diagnose failures before selective reruns. Inspect actual `head_sha`, attempt, conclusion and artifact identity using `gh run view <run-id> --json headSha,attempt,conclusion,jobs,url`.
- [ ] Use one consolidated owner scenario on the identified native builds: old/new imports; offline restart into saved default; restore/switch both accounts; conflicting Frontier/journal facts and coherent ranks/goals; complete Fleet/current ship/global search; on-foot Exosuit/live/offline transitions; route/no-route FSD charge/cancel/arrival; XYZ budget recovery; missing/stale/failed search; localized biology. Compare full main/paired gameplay with launcher-only settings, then revoke. Capture Windows/Linux window/DPI/touch behavior in that session.
- [ ] Verify mixed recipient receipts through the synthetic transport. A real confirmed provider receipt remains separately consented for the exact account/category/payload; if the owner declines, explicitly retain that live acceptance gap. Do not enable dormant Inara or sacrifice a working Frontier grant to manufacture coverage.
- [ ] Final acceptance requires the requested source order across backend and UI, stable default selection, equal gameplay displays without settings, suit/ship separation and attributable Exosuit telemetry, complete owned Fleet/search/count, truthful per-event receipts, no data regression or revived consent, usable maps, correct localized inference, complete-or-suppressed station stock and matching artifact identity. Record remaining gaps. Merging stacked PRs and releasing remain later owner decisions.
- [ ] For expanded correctness, verify shared custom/model names, exact module variants, omitted-name preservation, current catalog coverage, BioScan source/boundary coverage and stale-response rejection, Inara fallback ranks, current-reference parameters and both actual native CSV exports. Do not equate labels with verified engineering/statistic coverage.
- [ ] For Batch B, compare baseline/normal/low-latency whole-process metrics, same/different view demand across two displays, release/resume/reconnect, bounded caches and continuous essential route/sample/FSD state. Assert no additional provider requests solely because more displays or views are warm.
- [ ] For Batch C, verify in-app calculation/current-origin fill, cancel/restart/account-switch, typed repeated/body/trade stops, Spansh-first display with separate actual game target, key calibration and bounded gear/route actions. Verify equal main/paired operational access without another permission step, shared concurrent-action handling and each claimed platform's actual outcome. A sent key or completed provider job is not proof the game plotted a route.
- [ ] For Batch D, verify selected docking events/layouts, module enable/disable behavior, complete supported algorithm predicates and builder source/conformance manifest. Mark remaining catalogue entries as deferred/unavailable rather than complete.

## Evidence limits carried into execution

CI runs associated with the audited G3F PR head had two native and five legacy jobs green on first-attempt runs [45007583577](https://github.com/Jakoczig/eridion/actions/runs/45007583577) and [45007583599](https://github.com/Jakoczig/eridion/actions/runs/45007583599). Local audit checks passed 178 JavaScript tests, 29 asset tests and both blocking lints. These results are baseline evidence, not proof of the proposed corrections or independent attestation of packaged binary revision.

The audit environment lacked a genuine local Go compiler; new evidence used source inspection, the actual React harness, CSS cascade analysis and exact SQLite reproductions. Chromium rendering was blocked, so no local desktop geometry screenshot was obtained. Native compilation, platform vault behavior, complete gameplay and any new live receipt must be established by the final gate, with unsupported claims left explicitly unverified.

Revision 2 adds focused source review and actual React reproductions for Commander priority and owner/paired content/settings. It does not claim any application correction has been implemented, nor that a specific missing ship from the owner's real account has been identified. Primary contract references are linked in the companion audit, including CAPI inventory/current-ship semantics and Frontier's on-foot Status fields.

Revision 3 added catalog comparisons, BioScan literal/interpreter analysis, request-order/count and Inara-rendering probes, the CSV bridge defect, service contract checks and native integration design. Revision 4 records the owner's confirmed D0–D2 and replaces lower-trust paired gameplay restrictions with common trusted-display operations. Native performance, live planning/provider acceptance, real game input and physical second-display behavior remain future acceptance evidence. Catalog counts and upstream heads are dated inputs; no product clarification remains for these three decisions.
