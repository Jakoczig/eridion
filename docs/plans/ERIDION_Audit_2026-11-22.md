# ERIDION Project Audit — 22 November 2026

**Revision 5 — recorded performance actions and Claude handoff, 22 November 2026:** the owner confirmed **Frontier → journal/Status → Inara and all other providers equally**; every paired display has the same trust and gameplay controls as the main display; **LandingPad** is the selected docking helper. Pairing itself establishes display trust, with no extra per-device game-control opt-in. Settings remain launcher-only on every display. D0–D2 are resolved. Findings F01–F27 and their evidence remain the audited baseline; implementation is still planned in the companion correction plan and roadmap.

## Documentation status and Claude handoff — Revision 5

The owner requested that the additional performance findings be recorded and the coding work prepared for Claude. This revision publishes the complete planning set in the repository and links the [detailed native performance plan](../superpowers/plans/2026-11-22-native-performance.md) and [Claude coding handoff](../handoffs/2026-11-22-claude-coding-handoff.md). Start with the [planning index](README.md). **Runtime corrections and PERF-01–10 remain unimplemented in this handoff.** Performance opportunities were confirmed by source inspection; no new runtime speedup or G3F benchmark result is claimed.

Keep two baselines distinct:

- **Native implementation source:** G3F `864c724057a7fe28d9d42f5985fa5e1828d3b030`, PR #28, `go/g3f-galaxy-viewport-slices`; still open, draft and unmerged when rechecked on 22 November 2026.
- **Main documentation/configuration baseline:** `ffc5f663c0cfafff1acd41b1af65cc5b4ff056db`. README rewrite `eff5fe7`, inherited-file consolidation `40d07eb`, and package/legacy-workflow correction `ffc5f66` are already published there. Main still lacks the native `go/` tree. These commits do not mark native findings or feature tasks complete.

The original audit observations below describe their pinned source and inspection time. In particular, main's old stable-release workflow has since become an artifact-only legacy packaging workflow; historical native branches retain their own copies. Read the [package/workflow maintenance record](https://github.com/Jakoczig/eridion/blob/ffc5f663c0cfafff1acd41b1af65cc5b4ff056db/docs/maintenance/package-and-workflows.md) before adapting metadata or workflows. Preserve native test scripts and gates; do not merge main into the native stack or replace native workflows wholesale.

## Decision and recommended next step

ERIDION has a substantial native foundation that should be retained. Correct data identity, authority, restore behavior, maps, receipts and exports; deliver the default commander, equal gameplay displays, Exosuit and complete Fleet; then complete the performance subscription architecture before adding heavier native planning/helpers. Establish a whole-native performance baseline during the first correction batch. Native Spansh planning and deliberate game-control actions are now explicit product requirements.

The recommended next branch is **go/g3g-audit-corrections**, stacked on **PR #28 / go/g3f-galaxy-viewport-slices**, beginning at **864c724057a7fe28d9d42f5985fa5e1828d3b030**. This name is a proposal; no branch or PR was created during this audit. The companion document, **ERIDION_Correction_Plan_2026-11-22.md**, turns the findings into implementation tasks and acceptance checks.

The audit found reproducible functional defects despite passing tests. It did **not** prove a native owner/viewer authorization bypass, token disclosure, or arbitrary remote download/path escape in the examined code. That is a bounded review result, not a security certification.

**Preserved constraints:** no merge, release, deployment, publisher change, production upload, live account modification, or replacement of the original application. This audit did not change runtime source. The later main documentation/configuration commits are listed above. All 28 existing PRs were open drafts at the original audit snapshot; refresh that manifest before implementation.

## 1. Scope, baseline, and evidence

### Repository state

| Item | Verified state |
| --- | --- |
| Repository | [Jakoczig/eridion](https://github.com/Jakoczig/eridion) |
| Audited head | [864c724057a7fe28d9d42f5985fa5e1828d3b030](https://github.com/Jakoczig/eridion/commit/864c724057a7fe28d9d42f5985fa5e1828d3b030) |
| Current development PR | [#28 — G3F](https://github.com/Jakoczig/eridion/pull/28), draft, unmerged |
| Immediate base | [#27 — G3E](https://github.com/Jakoczig/eridion/pull/27), 64ddd186f692e48b2e2c1f5a342be20de5d1ac12 |
| Earlier G3D baseline | [#26](https://github.com/Jakoczig/eridion/pull/26), 79a034d6a65f9d83272f7655f35ed68f70eeecdf |
| Main | 06a69635623c5bfacff51785a33fbb5f09948c9d, legacy line |
| Inventory | 1,698 tracked files, including 324 Go files and 275 JS/JSX/TS/TSX files |
| Working tree | Clean at checkout and after audit checks |
| PR topology | Mostly a cumulative development stack; design PRs #1/#3 and parallel atlas PR #20 should not be assumed to be additional ancestors of #28 |

The G3E plan based only on PR #26 is now historical. G3E and G3F have been implemented since that baseline. Corrections should use the current cumulative source, preserving the earlier draft branches.

### Relationship to the former conversations

The audit compared owner instructions confirmed for this audit against code and repository evidence. Those instructions are a requirements source, not independent proof that a feature works.

The binding goals carried forward are:

- One lightweight Windows/Linux Go + Wails companion with shared journal processing and optional native modules.
- Restore useful original UI and gameplay functionality; use EDMC and its plugins as references rather than adding a Python/EDMC runtime requirement.
- Treat API correctness, Inara integration, exploration/BioScan, and second-screen usability as real requirements.
- Preserve account separation, exact identifiers, source provenance, offline operation, and indefinite retained journal history. Main and successfully paired displays share the same account-bound gameplay reads and actions, including configured host game controls. Configuration remains in the launcher for all displays.
- Use explicit per-provider/category consent. Replay and catch-up must never become fresh automatic uploads.
- Keep “After sale” as hold plus manual review; do not invent delayed-upload contracts.
- Prioritize Frontier before activating Inara. The earlier website/CSV-only Spansh scope is superseded by the latest request for in-app planning, current-location fill and active-plan precedence; retain website/CSV as fallback.
- Use larger coherent work batches, targeted intermediate checks, one consolidated final review/CI gate, and a practical gameplay checklist.

### Confirmed owner requirements

| Priority | Eligible source |
| --- | --- |
| 1 | Successfully refreshed, validated, account/entity-matched Frontier data within the relevant freshness policy |
| 2 | Valid journal evidence, including attributable Status sidecar telemetry where that is the actual source |
| 3 | Valid Inara and all other third-party provider data at equal source priority, within existing authorization/approval |

**D0 confirmed:** the owner explicitly confirmed one shared third-party tier, including Inara. The earlier separate Inara tier is superseded. Source validation, freshness and deterministic tie rules distinguish eligible candidates within the equal tier; provider name must not create a hidden preference.

Apply this per meaningful field or coherent group, not by replacing an entire object with a partial response. A missing, invalid or expired Frontier value falls back through the lower tiers. A legitimate zero or false is present data. Frontier may win without a known source observation timestamp; retain unknown `observedAt` and actual `fetchedAt` instead of inventing an observation. CAPI is a periodically queried snapshot, not a continuous telemetry feed. Preserve its cooldowns and consent. Equal-tier providers use documented freshness and deterministic tie handling, retaining conflicting alternatives rather than adding a hidden provider preference. Chronological selection within each source remains the separate F01 correction. [CAPI contract][capi-contract]

Related fields must remain coherent: Frontier rank 8 must not inherit journal progress 55% that belongs to rank 5; ship modules must belong to the same ship instance. Immutable journal events and sales remain historical evidence even when current authoritative values change.

When multi-commander mode is off, automatically select a persisted default commander. All gameplay displays expose the same selected-commander information, navigation and operational actions, including personal commander/fleet/loadout data, planning and configured game controls. Every paired display is trusted equally with the main display. Each display can choose its own page and map view. Configuration belongs in the local launcher, including account selection, provider connections, pairing, modules and recovery tools; neither main nor paired gameplay screens expose settings. Credentials remain backend-held and existing account binding applies equally. Add Commander → Exosuit for known equipment and current on-foot readings; never present a suit as a ship. Complete the owned-fleet list and search rather than merely changing ship labels.

An optional historical reconciliation wizard was approved in the earlier discussion. The source also contains a dormant, explicit, Inara-only historical publishing path for records up to 30 days old. This audit does not treat its existence as authorization to send anything. Keep the approval gate and exact owner-reviewed batch acknowledgement; do not reinterpret it as automatic historical publication.

### Expanded requirements recorded in this revision

Ship presentation must distinguish custom name, identifier and model label, with correct fitted-module class/rating/variant and witnessed engineering. The same resolved identity must reach every gameplay display. Builder checks must report current upstream catalog/algorithm revisions and native behavior coverage; using a current source pin alone is insufficient.

Spansh planning belongs inside ERIDION, with one-click current-system fill, native job handling, import/activation and active Spansh priority in the Route tab. Preserve the game's actual route and target as separate observations. Launcher → Key Controls configures typed game actions, including landing gear and an explicitly clicked next-target search/select/plot sequence. The former blanket no-game-command product limitation is superseded for these requested capabilities; no actual game input was authorized or performed during this audit.

Performance is a first-order acceptance requirement. Normal mode updates demanded views; optional **Low latency: keep views ready** maintains enabled view models. Both keep ingestion, essential live state, route/sample progress and action acknowledgements current. One host shares work across displays and respects provider cooldowns. Native helper integrations must be individually controllable in the launcher and use shared ingestion/storage/services.

### What was checked

The review covered architecture, project history and PR state, journal/storage/reconciliation, account recovery, owner/viewer boundaries, OAuth/vault/consent, signed atlas/spatial ingestion, map/navigation/FSD/window behavior, provider payloads and receipts, Commander/Fleet/Community Goals, BioScan, BGS/Spansh/statistics/module parity, dependencies, CI, packaging, documentation, and performance claims.

Three independent review tracks examined map/UI, integrations/parity, and native security/reliability. Findings were reconciled against the actual pinned source; high-value React/CSS/SQL reproductions were also run by the coordinating reviewer.

The follow-up inspected source authority/fleet, default-account/display behavior and Exosuit handling in three focused tracks. Actual React harness cases confirmed that the Commander page still renders journal rank 5 despite receiving effective Frontier rank 8, and that paired Commander/Fleet pages are blocked while terminal settings remain mounted. Other new findings below are source-backed, with native/live-user reproduction limits stated explicitly. The original broad test counts remain baseline results; they were not rerun just to amend the plan.

The expanded follow-up completed independent catalog/algorithm, performance, and Spansh/plugin/input tracks, plus a coordinating review of every currently registered native service against primary documentation or maintained source. It compared all current/upstream catalog IDs, parsed the pinned BioScan rule data without executing upstream code, reproduced a BioScan stale-response overwrite and an Inara-rank display mismatch in the actual React harness, and checked the native CSV bridge. Native CPU/RSS/latency, live provider acceptance and game-input execution remain unmeasured. A secondary source review corrected an initial EDAstro cadence hypothesis: the existing one-minute read gate is distinct from its five-second upload cadence.

This is a repository-wide audit at the requirements and subsystem level, with deeper review of important failure paths. It is not a line-by-line formal verification of every tracked file.

### Verification results

| Evidence | Result | What it establishes |
| --- | --- | --- |
| Local npm test | **178 passed, 0 failed/skipped** | Existing JS service/frontend regressions at the audited head |
| Local test:go-assets | **29 passed, 0 failed/skipped** | Asset, packaging, workflow and fixture checks |
| Local lint:security | Passed | Blocking scoped JS lint |
| Local lint:go-assets | Passed | Blocking build/test script lint |
| [Native CI 45007583577](https://github.com/Jakoczig/eridion/actions/runs/45007583577) | **2/2 jobs successful**, attempt 1, run associated with PR head 864c724… | Windows/Linux Go tests/vet, builds, native/browser smokes, Linux race checks and retained gates |
| [Legacy/security CI 45007583599](https://github.com/Jakoczig/eridion/actions/runs/45007583599) | **5/5 jobs successful**, attempt 1, same head | Existing legacy checks; global legacy lint is explicitly advisory |
| Map state harness | Defects reproduced | Actual component behavior with synthetic RPC responses |
| Imported CSS analysis | FSD block-layout override reproduced | Deterministic selector/cascade result; no rendered screenshot |
| Exact SQL/schema reproduction | Chronology and consent-reset defects reproduced | Actual SQLite query/state behavior using synthetic in-memory data |
| npm audit | **45 package alerts** | Dependency-maintenance input, not 45 demonstrated application exploits |

The native CI run was created at 10:26:48 CET and finished updating at 10:43:19 CET on 22 November (Europe/Budapest); the complete run took about 16½ minutes including scheduling. This supports keeping full CI as a final batch gate rather than repeating it for every small edit.

### Important evidence limits

There was no usable Go compiler in the audit container, so successful Go tests and native builds were **verified from GitHub CI runs associated with the audited PR head**, not rerun locally. The workflow uses the default PR checkout, which can be a synthetic merge revision. The run's head SHA alone does not independently establish the revision embedded in every packaged executable; build identity is a correction task below. Chromium could not launch in the audit environment; the UI findings use source, actual React harness behavior, and deterministic CSS analysis. No new Windows/Linux native screenshots or physical gameplay acceptance are claimed.

No real journals, production profile, credentials, live OAuth grant, provider upload, or publisher endpoint were exercised. Public specifications, primary upstream source, and the public npm advisory service were consulted. Prior documented owner acceptance remains historical evidence, not acceptance of the latest build.

## 2. What is already delivered, and what is still partial

| Area | Current assessment | Remaining work |
| --- | --- | --- |
| Go/Wails foundation | Implemented with cross-platform CI for the audited PR head | Real platform/runtime acceptance, packaged revision identity and performance envelope |
| Shared journal ingestion | Implemented replay/live/rotation/partial-line/provenance handling | Chronological current-view defects and directory capacity limit below |
| SQLite history | Implemented durable history, backups, migrations, recovery and cached startup | All-account restore consent and observation-order corrections |
| Pairing and secondary displays | Implemented expiring six-character pairing, TLS, read-only scopes, revocation | Replace reduced gameplay view with full selected-account display parity; physical reconnect/idle/touch acceptance |
| Launcher/accounts/settings | Substantial implementation, but single mode follows journal-active FID | Persist and auto-select a default; move remaining terminal settings to launcher; restore consistency |
| Launcher service labels | Earlier complaint corrected | It now says reviewing saved status; no automatic service health probe is claimed |
| Commander/Fleet/Community Goals | Substantial native and CAPI functionality | Apply new source hierarchy in backend and UI; complete resolved fleet/count/search and coherent rank/progress |
| Commander → Exosuit | Missing; suit-shaped LoadGame values can occupy ship identity | Separate known suit/loadout and real ship; retain suit evidence and add attributable live on-foot telemetry |
| G3C map data | Signed density overview, bounded spatial cache, offline Bubble and consent implemented | Whole-native measurements and real route/coverage acceptance |
| G3D | Unified map, synchronized projections, remaining-route framing and FSD policies implemented | FSD CSS and async-state issues found by this audit |
| G3E | Fixed exterior panel geometry, one search action, 16:9 first-window defaults and saved normal/maximized state implemented | Search completion/provenance and smaller-monitor aspect behavior |
| G3F | XZ camera separated from XYZ selection; Y no longer inflates face-on camera | Physical acceptance; explicit side-axis scale explanation |
| BioScan | Native pinned rules, candidates, sample spacing and reference values implemented | Stable genus ID fix; expanded validated constraints and language/cache tests |
| BGS Tally | Contextual native journal accounting, closed-tick/UTC provenance implemented | Full upstream reporting/inference/overlay parity remains out of scope |
| Spansh | Website destinations, notes and limited CSV itinerary following/history implemented | New required native planning, current-system fill, typed stops, active-plan precedence and later next-target game plotting |
| Ship statistics | Fitted power, stowed/deployed scenarios, priorities and final journal modifiers implemented | Validated full jump/defense/thruster/weapon models remain incomplete |
| EDSM | Reads, enrichment and selected uploads implemented | Per-event acknowledgements and missing-coordinate search |
| EDDN | Schema validation and dock/time/consent checks implemented | Public station inventory classification is too narrow |
| Frontier | Bounded account/fleet/station/CG integration implemented; prior live-read evidence exists | Current preview acceptance, refresh/revocation and carrier-owner 200 response |
| EDAstro/GEC | Bounded implemented subset | Keep rare-POI scope and live-upload acceptance limits explicit |
| Inara | Adapters exist; application approval gate remains | Fix cash/assets mapping before activation and finish controlled acceptance |
| Module toggles/credits | Implemented compiled native modules | These do not constitute a downloadable plugin ecosystem |
| Plugin installation/auto-update/full overlay | Deferred | Separate lifecycle, compatibility, update and permission design needed |
| Other original features | Parity is incomplete | Complete a route-by-route native/legacy/unsupported ledger before further ports |

The former Frontier audit is relevant: its repository handoff records successful owner-authorized reads of profile, market, shipyard and communitygoals, plus a normal fleetcarrier 204 response on 18 November. It does not establish refresh-token rotation, current-build UI acceptance, or a complete carrier-owner 200 contract. [Frontier live-audit handoff][frontier-audit]

The G3F geometry tradeoff is intentional: a fixed square face-on view and thin side strips cannot display arbitrary independent XYZ extents with one identical light-year-per-pixel scale in every direction while filling every panel. The current solution preserves X/Z isotropy and compresses Y in the side strips. Empty/unreported space remains empty. Acceptance should assess this agreed presentation honestly, rather than requiring data distortion. [G3F PR][pr28]

## 3. Findings and correction priorities

Priorities below are **development priorities**, not CVSS scores. **P1** means include in the next correctness batch because the issue affects an existing core promise or publication reliability. **P2** means the following maintenance/capacity batch or a prerequisite to activating that feature. A package advisory severity is separately identified.

### F01 — P1: importing older history can regress current facts

**Evidence:** exact SQL and schema reproduction, plus traced ingestion path.

Historical records are retained correctly, but some “current” queries choose by insertion ID instead of observation time. A later import of an older independent journal can therefore replace newer Commander facts, navigation evidence, and construction status.

The live reducer deliberately ignores older source records, but persistence still commits them into immutable history. The current-value queries then select the wrong row. In the synthetic reproduction:

- Newer credits 500 remained stored, but the effective query chose older credits 100.
- A complete construction project at progress 1.0 became an older 0.25/incomplete view.
- Newer fuel evidence 32 remained stored, but the query chose older evidence 4.

This is a derived-data correctness defect; no deletion, cross-account leak, or automatic upload was demonstrated. Existing timestamp guards on commander names and ship records show that the right pattern already exists elsewhere.

**Correction:** use deterministic observed-time precedence within each source, preserve unknown-time distinctions and stable equal-time tie-breaking, and repair affected derived views from retained history. Keep Rank levels and Progress percentages semantically separate.

**Acceptance:** newer→older and older→newer imports must yield identical current views, before/after restart and rebuild, with both commanders and all original rows preserved. Include future/unknown/equal timestamps and copied sources.

Sources: [EffectiveFacts][facts], [normalization][normalize], [navigation evidence][nav-evidence], [persistence][persistence].

### F02 — P1: restoring a backup does not reset every account’s provider consent

**Evidence:** exact restore SQL and account-state reconstruction.

Restore correctly sets global publication policies to Never, removes exceptions, pauses the active provider rows, and holds pending candidates. However, other accounts’ settings also exist in the provider-ledger JSON. Those account scopes are left unchanged and can become enabled again when selected.

The reproduction restored account A as paused/upload=false while account B retained read=true, upload=true, paused=false. **Both global publication policies remained Never.** Therefore, restoration alone did not prove an upload or privacy breach. The defect is that the effective per-account reset is incomplete despite the restore promise; later global re-enablement can expose the old provider flags.

**Correction:** reset all account namespaces atomically in the staged backup and update the ledger checksum. Preserve credentials in their existing OS-vault namespaces; require renewed effective consent for restored accounts.

**Acceptance:** restore with A, B and a separately pinned account selected; all remain paused across another restart, policies remain Never, exceptions are absent, candidates are held, and transports see no activity until the relevant consent is renewed.

Sources: [restore][restore], [consent persistence][consent], [provider reload][provider-persistence], [provider accounts][provider-accounts].

### F03 — P1: FSD still inherits a single-panel CSS layout

**Evidence:** actual imported CSS parsed and matched against the rendered component ancestry.

The older compact selector has four classes and sets display:block. It overrides the newer unified grid selector with two classes. An additional older FSD rule also sets block. The compact XZ rule sets height:100%, which conflicts with the three-panel geometry.

The normal map’s winning display rule is grid; the compact/FSD map’s winning rule is block. This explains a layout path that existing map-helper tests do not protect. The rendered device symptom was not observed in this environment.

**Correction:** scope the legacy compact rules to their legacy component or explicitly define the unified FSD geometry. Preserve the intentional three-panel layout, touch controls and return behavior.

**Acceptance:** assert computed display:grid and actual bounds/visibility of XZ, XY and YZ during charging, jumping and restore, at desktop and small-screen sizes. An “FSD overlay exists” assertion is insufficient.

Sources: [legacy compact CSS][compact-css], [unified CSS][g3d-css], [FSD component][terminal].

### F04 — P1: XYZ controls disappear while Local data are pending or unavailable

**Evidence:** actual-source React harness with fabricated complete/incomplete spatial responses.

All six axis buttons were present before an extent edit. Immediately afterward the renderer switched to Galaxy fallback and all six disappeared. After an over-budget/incomplete response, they remained absent. The backend’s specific reason was also discarded.

The controls depend on successful Local rendering, so the user loses the controls needed to shrink the slice that caused the failure. This is a state-model issue independent of the repaired G3F camera math.

**Correction:** separate intended mode and editable selection from loading/completeness/rendered-data state. Retain XYZ controls while Local is selected and disclose pending/missing/budget reasons. Do not display incomplete coordinates as complete.

**Acceptance:** pending, missing, over-budget and transport-error responses retain the controls; shrinking a rejected volume recovers Local data without changing camera scale or enabling downloads. Superseded responses must not overwrite the current selection.

Source: [unified map component][unified-map].

### F05 — P2: search can stop on an unusable hit and misrepresent result freshness

**Evidence:** component harness and backend source trace.

A named local point without XYZ causes search to return before authorized EDSM fallback. The refresh path also needs to fill missing coordinates for an existing point. Separately, stale cache metadata is discarded by the UI, and a failed replacement query leaves the previous result and View System action displayed.

**Correction:** distinguish a named hit from a usable coordinate result, resolve and merge missing coordinates only through existing consent, and retain per-result source/freshness/status. Associate result state with the query that produced it.

**Acceptance:** missing-XYZ local hit with EDSM enabled resolves once from either main or trusted paired display under the same account/provider policy; disabled, paused, unpaired or revoked sessions send no unauthorized request. A failed second query cannot look like a successful result for the second query; stale data remain explicitly stale; duplicate submissions are suppressed.

Sources: [search and component state][unified-map], [navigation context][navigation-context], [community lookup][community].

### F06 — P1, revised: apply Frontier-first authority consistently

**Policy correction:** the earlier recommendation to keep a plausible journal value ahead of a fresh Frontier value is superseded by the owner's explicit new order. The original goalCards reproduction—fresh CAPI total 100 selected over journal total 200—is now expected behavior for that eligible field, not a defect by itself.

**Confirmed mismatch elsewhere:** EffectiveFacts explicitly restores journal authority unless an external source proves a newer observation; CAPI deliberately has no such timestamp. Commander UI starts from journal ranks and adds external categories only when missing. Its ship identity/location also comes directly from journal reducers. Fleet separately preserves journal values and suppresses current Frontier rows against historical local IDs. Changing two source scores will not correct these bypasses.

An actual component reproduction supplied effective fresh Frontier Combat rank 8 and journal Combat rank 5; the page rendered Rank 5. Existing journal-first labels and tests also need to change. Preserve the distinction between field authority and chronological ordering within one source.

**Correction:** use a common field/group resolver across Commander, Fleet, Community Goals, Exosuit and overlapping provider reference domains. Fresh validated Frontier wins even with unknown `observedAt`; journal follows, then one equal tier containing Inara and all other providers under confirmed D0. Missing/expired fields fall back. Keep real fetch/observation metadata, immutable history and account binding; normalize equivalent field meanings before comparison. Equal priority does not merge unrelated station stock, owned ships or biological reference records.

**Acceptance:** fresh Frontier 100 beats journal 200; after expiry/missing data journal 200 wins; missing journal falls back to the eligible equal third-party tier, including Inara. Test an Inara-versus-other-provider tie without an automatic Inara preference, zero/false, nested omissions, exact IDs and incompatible rank/progress. Goal membership/contribution can use verified personal Frontier fields under this order, but a global-only goal must never invent participation, contribution or a received reward.

Sources: [EffectiveFacts][facts], [Commander UI][commander-ui], [account overview][accounts], [fleet reconciliation][frontier-fleet], [goalCards][goal-cards], [CAPI normalization][capi-snapshots].

### F07 — P1: BioScan uses localized text as the genus identity

**Evidence:** source/data reproduction using the checked-in rules.

The exploration reducer prefers Genus_Localised and drops the stable Genus symbol. BioScan then matches only a raw codex symbol or English genus name. A different translated display label yields no candidates.

The bacterial rules selected 13 candidates for the raw symbol or English name and zero for an alternate localized label. This is a deterministic language-dependence defect; no real localized game capture is claimed.

**Correction:** retain canonical genus IDs and localized labels separately. Infer from IDs; render labels. Rebuild cached projections from retained evidence where possible instead of adding a translation table.

**Acceptance:** changing only the localized label must not change candidate IDs/states. Cover absent labels, offline restart, cache upgrade and body/FID/session isolation. Keep unknown biological constraints explicit.

Sources: [genus reduction][genus], [BioScan assembly][field-bio], [rule matching][bio-rules].

### F08 — P1: EDSM can record rejected events as confirmed

**Evidence:** source plus primary EDMC response handling.

The EDSM acceptance function inspects only the outer msgnum. Dispatch then removes the batch and marks every candidate confirmed. It does not inspect per-event replies. An outer success with a rejected inner event therefore satisfies the success branch.

The positive fixture uses only an outer success response, so it does not test actual per-event receipt semantics. EDMC’s primary implementation explicitly checks the individual replies. No real upload was made by this audit.

**Correction:** distinguish account-read responses from journal-batch responses. Correlate each event receipt to its submitted record and store accepted/rejected/duplicate/pending/unknown outcomes accurately. Do not resend ambiguous deliveries automatically.

**Acceptance:** mixed results, missing/short/extra replies, all-success batches and known discard codes. Count only documented acceptance. Also resolve the separate final-429/503 terminal-state test described below.

Sources: [acceptance and dispatch][provider-manager], [batch construction][provider-batch], [primary EDMC implementation](https://github.com/EDCD/EDMarketConnector/blob/main/plugins/edsm.py), [official EDSM contract](https://www.edsm.net/en/api-journal-v1).

### F09 — P1 before trusting station publication: EDDN inventory is systematically incomplete

**Evidence:** payload filter source, bundled symbols and published schemas.

Outfitting accepts only eight narrow core-module families. Ordinary chaff, shields and fuel scoops already present in the local catalogue are omitted. Shipyard uses a fixed 20-ship list and omits ordinary types including Dolphin and Krait MkII.

A mixed list is published after filtering. The EDDN schema supports ordinary public symbols and separately identifies commander-dependent exclusions. Schema-valid JSON does not establish inventory completeness.

**Correction:** use reviewed public catalogue metadata and explicit exclusion rules. Preserve cosmetic/rank/purchase-dependent restrictions. If a complete truthful category cannot be produced, keep it disabled rather than representing a truncated list as ordinary stock.

**Acceptance:** representative public utility, weapon, armour, shield, scoop and ship symbols survive; restricted symbols remain excluded; exact market ID, dock identity, marker timing and consent gates remain intact.

This audit did not establish how every downstream consumer interprets missing entries; no remote deletion is claimed.

Sources: [station adapter][station], [EDDN outfitting schema](https://eddn.edcd.io/schemas/outfitting/2), [shipyard schema](https://eddn.edcd.io/schemas/shipyard/2).

### F10 — P2 before Inara activation: total wealth is mapped to cash

**Evidence:** current adapter and primary Inara/EDMC documentation.

Statistics.Bank_Account.Current_Wealth is sent as commanderCredits. The API distinguishes cash, total assets and debt, and EDMC maps Current_Wealth to commanderAssets. Omitted debt can also reset the remote loan field to zero.

The application approval gate currently blocks production Inara activation. This is a latent adapter defect, not evidence of current account damage.

**Correction:** send only a coherent witnessed snapshot with cash/assets/debt kept distinct; handle unknown values conservatively. Keep the gate until application approval and contract acceptance are complete.

**Acceptance:** cash100/assets500/debt25 maps to three correct values; missing cash or debt does not become wealth or an invented zero; timestamps/accounts remain bound; default configuration makes zero requests.

Sources: [Inara adapter][inara-adapter], [Inara documentation](https://inara.cz/elite/inara-api-docs/), [EDMC release mapping](https://github.com/EDCD/EDMarketConnector/discussions/1466).

### F11 — P2 capacity: the journal-directory limit can stop current ingestion

**Evidence:** direct bounded discovery control flow.

The reader rejects a directory with more than 4,096 entries before filtering journal filenames/types. Unrelated files and directories consume the same count. Crossing the cap stops even the known current journal from being processed on that poll.

The error is explicit, existing DB history is retained, and the cap is intentional. This is an operational ceiling for long-lived profiles, not silent pruning or an exploit.

**Correction:** separate bounded discovery work from total inventory and the active-tail path. Use paged discovery/known-source indexing and visible backlog state, retaining memory and work limits. Do not delete history or simply remove every cap.

**Acceptance:** more than 4,096 journals and mixed entries, live appends, rollover and restart with bounded memory and retained dedupe.

Source: [journal discovery][journal-reader].

### F12 — P2 maintenance: dependency debt needs exposure-based triage

The current lockfile scan returned **45 affected package entries: 7 critical, 26 high, 9 moderate, 3 low**. Of these, 38 occur only in the dev/build dependency graph, while 7 occur in legacy runtime or mixed graphs: axios, brace-expansion, follow-redirects, form-data, glob, minimatch and ws.

Representative locked versions are Next 12.3.7, axios 1.15.0 and ws 8.20.0. Next 12 is outside the publisher’s supported versions. The native application embeds a static React export served by Go; it does not run the Next server or the legacy Node provider backend. Consequently, server-side Next advisories cannot automatically be treated as native remote exploits.

The ws maintainer documents fragment-related memory exhaustion before 8.21.0. The legacy server has a 1 MiB payload cap and authenticated upgrades, which reduce and constrain exposure; they are not evidence that every underlying advisory is fixed.

**Correction:** classify each advisory by shipped runtime, development/build path, reachable input and mitigation; update small compatible legacy fixes first, and modernize the frontend build stack in a separate controlled task. Do not run npm audit fix --force blindly across the native/legacy shared frontend.

**Acceptance:** reproducible installs, scoped advisory disposition, unchanged export/routes, Windows/Linux native and legacy tests, and no unsupported runtime added.

Sources: [lockfile][lockfile], [native static handler][static-handler], [legacy socket boundary][legacy-socket], [Next support policy](https://nextjs.org/support-policy), [ws advisory](https://github.com/websockets/ws/security/advisories/GHSA-96hv-2xvq-fx4p).

### F13 — P2 delivery: documentation and build identity are stale

The audited README still describes G1A, says pairing/integrations are unimplemented, and contains legacy claims about non-persistence and privacy that do not describe the native history/provider implementation. Two differently named project summary files end at different old stages.

G3F packaging still selects the G3E README and names artifacts eridion-g3e. The PR candidly discloses this, but a downloaded archive lacks a clear milestone/head identity in its label. Runtime telemetry also contains historical stage labels.

**Correction:** create one authoritative capability/acceptance ledger and build identity derived from the commit/milestone. Use it in archive naming, included README, native diagnostics/About and test evidence. Record the actual checked-out/build revision and distinguish a preview head from a synthetic merge-test revision. Preserve upstream attribution and separate legacy requirements from native ones.

**Acceptance:** a user can identify the exact build without relying on a chat message; archive identity, executable metadata, README, commit and hashes agree. Package only intended files. The original three-file packaging boundary remains valuable; add metadata deliberately rather than sweeping a build directory.

Sources: [README][readme], [summary A][summary-a], [summary B][summary-b], [packager][packager], [desktop workflow][desktop-ci].

### F14 — P2 readiness: merge/release constraints are procedural, not fully enforced

At inspection, GitHub reported main as unprotected and the repository rulesets response was empty. Auto-merge was disabled, and all current PRs were draft. The manual legacy release workflow still creates a non-draft release and lacks an explicit approved-branch/test gate.

These facts do not mean a release occurred. They mean future operator mistakes are not comprehensively blocked by repository configuration.

**Plan:** before any separately authorized release effort, establish reviewed branch/check requirements and a deliberate release identity/provenance gate. Keep the existing draft/no-release policy now. Changes to repository settings or release workflows were not made during this audit.

Sources: [main branch metadata](https://api.github.com/repos/Jakoczig/eridion/branches/main), [rulesets metadata](https://api.github.com/repos/Jakoczig/eridion/rulesets), [release workflow][release-ci].

### F15 — P1: single mode has no persistent default commander

**Evidence:** account preference, selection and provider pinning source. AccountSettings contains only Multiple and Selected. Disabling multiple mode clears Selected, and displayFIDLocked follows the journal-active FID. Single mode already bypasses the repeated chooser; the missing concept is a stable configured default. Its launcher identity card is also hidden in the actual component harness.

**Correction:** persist defaultFID independently of multi-mode selection and resolve one FID for displays, provider pinning and remote account binding. Automatically use a valid default on cold start, offline restart and reconnect. Migrate a prior explicit verified selection or adopt a sole verified commander. If first-run cached accounts are truly ambiguous, perform one initial default setup in the launcher; do not guess from import order. Preserve a missing/corrupt preference for recovery rather than silently activating another account's permissions. A different journal-active commander must not replace the default implicitly.

**Acceptance:** default B with reader-active A always displays/pins B; offline cached B works; toggling multi mode preserves the independent default; older imports cannot alter it; first-run missing identity and corrupt/missing preferences remain honest. Show the selected identity in the launcher without an unnecessary Select step. Automatic selection does not require automatically opening an extra window.

Sources: [account state][accounts], [account RPC][accounts-rpc], [startup feed][gamefeed], [account UI][accounts-ui].

### F16 — P1: paired gameplay is reduced and settings remain in terminals

**Evidence:** frontend/backend role gates plus actual component reproduction. Paired Commander and Fleet are blocked; other owner-only reads include carrier/CG/operations, construction, BGS, saved itinerary information and biological positions. Several read RPCs share an owner-only branch with management mutations. Both terminal roles have a settings/status gear; the owner sidebar mounts full settings, provider diagnostics, history tools, module controls and pairing.

**Correction:** use the same selected-commander gameplay models, components and action permissions on every trusted display. Separate launcher management from common display operations; an old owner/viewer role must not create a lower-trust paired gameplay surface. Share identity, balances/ranks, ships/loadouts, Exosuit, missions/CG, carrier, construction, BGS, routes and biological context, including equivalent permitted workflow actions. Keep configuration in the native launcher and remove settings UI from every terminal. Preserve independent navigation, map controls, filters and pagination. Main and paired screens use the same account/provider policies and configured game controls without another device permission step.

Keep tokens, local paths, provider administration, raw recovery data and foreign accounts outside gameplay DTOs. Existing account-bound pairing, expiry and revocation remain. Sharing the requested gameplay data does not itself require another pairing/confirmation ceremony, a second journal reader or more provider polling.

**Acceptance:** compare main/paired normalized data, rendered content and operational action outcomes for the same selected FID, including Fleet, Exosuit, provider-backed search, planning and configured game controls. Neither gameplay surface mounts settings or management forms. Replace obsolete privacy/read-only gameplay assertions with equal-display assertions; retain launcher-only management, foreign-FID and unpaired/revoked-session checks. Reconnect, account switch and revocation must never leak another account's data or permit a stale action.

Sources: [terminal UI][terminal], [RPC gates][socket], [operations UI][operations-ui], [Commander UI][commander-ui], [field projections][field-bio], [native launcher][native-launcher].

### F17 — P1: suits can be presented as ships; Exosuit support is missing

**Evidence:** LoadGame unconditionally assigns its Ship and ShipID to the main-ship projection after resetting state. Character/length validation does not classify suits or other controlled entities. Commander, launcher account cards and the terminal header consume that projection as a ship. Primary EDMC handling specifically avoids this assumption. No real user journal/native reproduction was performed.

There is no dedicated native suit/loadout reducer or Commander Exosuit route. Suit event rows retain continuity metadata, type and hash, but their payload is saved as `{}` because these types are excluded from relevant-event retention. The existing database alone may therefore be unable to reconstruct prior equipment. Ordinary duplicate reimport also skips those rows. On-foot Status flags are recognized, but numeric extraction returns early outside the main ship.

**Correction:** distinguish the controlled entity, last verified real owned ship and known suit/loadout. Add Commander → Exosuit at `/cmdr/exosuit`, with the same content on all gameplay displays. Show known suit type, reported grade/modifications, loadout and weapon slots/attributes. Add current-session on-foot health, oxygen, temperature, gravity, selected weapon and supported state flags only when actually reported. Preserve last-known equipment offline; retire the live claim on embark, shutdown, source loss or account change. Missing values remain unknown. Do not invent numeric suit battery/shield percentages or infer upgrade grade solely from a stale type symbol.

Add bounded suit event retention and a versioned projection. For older metadata-only rows, enrich from available unchanged original journals using source/offset/hash verification and a separate versioned evidence record; preserve immutable original events. If original files or a trustworthy new snapshot are absent, show unknown until evidence arrives. Unknown genuine ship models remain visible; a finite catalogue is not an entity classifier.

**Acceptance:** on-foot LoadGame with FlightSuit or ExplorationSuit never becomes a ship; prior real ship remains last-known. Test SRV/fighter/taxi contexts, exact-ID suit/loadout/weapon lifecycle, translated labels, offline restart, 75% health/50% oxygen from valid Status, expiry/transitions, two accounts, and old-cache payload enrichment without duplicate events or publication.

Sources: [ship reducer][ship-model], [history retention][normalize], [Status parser][status-parser], [sidebar navigation][navigation-items], [primary EDMC monitor][edmc-monitor], [Frontier journal manual][journal-manual].

### F18 — P1: complete the resolved owned fleet, counts and search

**Evidence:** current Fleet/count/reconciliation source and existing paging tests. A journal row marked sold blocks a fresh positive Frontier record for that ID, including a possible later reuse. Top-level current `ship` is normalized separately but not unioned into Fleet's `ships` inventory; if the collection is incomplete and no local fleet row exists, a known current ship can be absent from the list/count. The UI loads 30 rows and searches only that page. CAPI-only rows appear after journal pages, so users can miss ships without any permanent paging failure.

**Counterevidence:** array and keyed Frontier collections already work, a stable 75-ship paging test exists, duplicate models keep distinct IDs, and unknown hull labels are not filtered out by the catalogue. The owner's precise missing ship IDs have not been inspected, so this is not a claimed diagnosis of every missing real vessel. Profile refresh during positional paging is an additional test lead, not a confirmed runtime result.

**Correction:** one account-scoped resolved fleet snapshot must drive current ship, list, Commander count and search on all displays. Include verified current ship plus owned inventory using exact IDs and entity kind. Apply fresh Frontier authority to current ownership/fields while retaining old sales as history; accommodate reused IDs without inventing transaction reversals. Record collection completeness: missing/partial/malformed lists cannot remove known ships or create sales. Absence from a validated complete snapshot means not reported in current inventory, not proof of a sale. Never add station shipyard stock to owned Fleet.

Search the whole known resolved fleet through bounded queries. Bind pages/counts to a snapshot revision, show loaded/total/coverage, and reset safely on account/filter/refresh changes. Stale or unavailable Frontier falls back under the revised priority rule.

**Acceptance:** current/stored/in-transit/Frontier-only ships; repeated model with distinct IDs; historical sale plus reused ID; current-only and partial profiles; no mass removal on malformed response; 0/1/30/31/75/512 ships; search for an item beyond page one; refresh on page two; Commander total equals resolved fleet total; suit/SRV/fighter entries do not inflate ownership. Retain exact account/instance boundaries.

Sources: [fleet reconciliation][frontier-fleet], [Frontier normalization][provider-reconcile], [fleet UI][fleet-ui], [operations UI][operations-ui], [CAPI contract][capi-contract].

### F19 — P1: ship names and model labels use inconsistent, incomplete paths

The bundled ship catalog has **38 models**, compared with **49** at inspected FDevIDs revision `c7c29f7a13df543d4bc8b947eab29f88e06100fa`. The 11 missing symbols include Python MkII, Type-8, Mandalay, Cobra MkV and newer hulls. Formatting `Explorer_NX` cannot produce its proper Caspian Explorer label. This is a confirmed coverage gap, independent of whether an owned ship is missing from the Fleet list. [FDevIDs shipyard][current-shipyard] [Catalog][catalog-source]

The common serialized ship state also excludes `ShipName`; only an owner-specific RPC repairs some historical names, while account cards/Commander/header/statistics use other paths. A later same-ship Loadout omitting `ShipName` clears the reducer name, and the historical-name timestamp filter can prevent recovery. Different screens can therefore disagree even when evidence exists. [Ship state][shipstate-source] [Identity handling][accounts]

**Correction:** one account/ship-scoped identity projection containing custom name, identifier, model symbol/label, source and freshness. Apply Frontier-first authority and distinguish omission from explicit clearing. Use it across local/paired gameplay, Fleet and planner ship selection. Keep unfamiliar hulls visible with their raw ID. A synthetic omission path is established; the owner's exact missing names still require a final profile comparison.

### F20 — P1: equipment data are incomplete; a simple refresh would misidentify variants

Current outfitting has **1,068 IDs** against **1,243** upstream: **175 missing IDs** and **five existing label/class/rating changes**. The module catalog discards numeric IDs and indexes one entry by normalized symbol. Current upstream has **22 repeated symbols** across distinct IDs: for example ordinary Beam Laser and Overloaded Beam Laser share a symbol. Blind replacement would let a special variant overwrite the ordinary item. Current bundled data do not yet have those collisions; this is a proven update hazard, not an assertion that those overwrites have already happened. [FDevIDs outfitting][current-outfitting] [Catalog][catalog-source]

CAPI already retains module numeric ID, symbol and slot, but the main module inspector uses the journal-only symbol path. Its journal-authority notice conflicts with the latest source policy. The separate selected Coriolis projection also lacks records for 223 current local symbols, so correct labels do not imply verified mass/power/performance. [CAPI normalization][provider-reconcile-source] [Module parsing][parity-parse-source]

**Correction:** index exact numeric ID plus symbol candidate sets; reconcile account, vessel and slot before joining evidence. Keep module size, slot size, rating, variant, stock values and observed engineering separate. A symbol-only ambiguous observation remains unresolved instead of acquiring guessed statistics. Update catalog provenance and coverage before importing current data; never apply engineering twice to a witnessed final value.

### F21 — P1 conformance, staged coverage: BioScan pin is current but the native algorithm is partial

The pinned BioScan `fc6b85c…` and ExploData `fd4e188…` are still the public default heads checked on 22 November. All **116 included literal species records** match that upstream source. The native converter omits **Bark Mound, Amphora Plant and Radicoida Unicus**, defined outside its 19 ruleset files: source total **119**. **74 of 116 included species** have no fully supported OR alternative in the native evaluator. [BioScan species][current-bio-species] [Native interpreter][bio-native-source]

Gravity/pressure conversions and the maximum-pressure boundary also differ from upstream. A small arithmetic probe at 1012.8 Pa reverses the maximum-pressure predicate for Tussock Stigmasis. This proves source-equivalence drift, not which formula is independently correct about the game. Missing regional/stellar/parent/composition/volcanism/variant capabilities must remain explicit. [BioScan algorithm][current-bio-algorithm]

**Correction:** add source-conformance fixtures, omitted records, explicit intended units/boundaries and a coverage manifest; then complete additional predicates with their required evidence. Preserve honest unknowns. A builder check must report both freshness and implemented behavior, rather than calling a current pin full parity. Keep F07 stable localized identity and biological sample persistence as separate regressions.

### F22 — P1: an older BioScan response can replace newer displayed data

The actual `BioScanPanel`/`useFieldRead` starts another request while one is pending. A deterministic React harness resolved request two first, then request one: the older result replaced the newer view. Interval and journal-triggered reads lack a shared in-flight/revision guard. Both BioScan and Operations clean up timers on unmount; Operations already has a working pending guard. [Field operations UI][field-ui-source]

**Correction:** one in-flight read per account/body/query, semantic invalidation, cancellation and response epochs. A reply from an older body, commander, request or rules revision must never become current. The probe establishes request counts and ordering; it is not a native latency benchmark.

### F23 — P1 performance foundation: display demand does not release backend work

The frontend already mounts the selected tab. Server request flags, however, stay true until the socket closes, and every display subscribes to full projections with unconditional navigation/exploration broadcasts. Single-domain getters construct the full projection; cloning/comparison happen before transport can choose a small Status delta. BioScan also repeats stable inference while holding the main service mutex. [Socket service][socket-source] [Projection service][projection-source]

**Correction:** explicit subscriptions/releases, domain revisions, shared immutable snapshots, finite caches and union of demand across displays. Split essential live state from optional view derivation. Normal mode computes demanded views; low-latency mode prewarms enabled view models without rendering hidden trees or bypassing provider limits. Preserve current bounded/latest-only channels, compact deltas, map query stability, durable journal history and existing ship-statistics caching. Full-process measurement is required before claiming speedups; see sections 6–7 and the expanded roadmap.

### F24 — P2 before affected feature acceptance: service contracts need targeted corrections

The complete implemented-service matrix is in the roadmap. Two additional concrete mismatches were found. Inara ranks are normalized as an array of `{name,value}`, while the Commander UI expects a map keyed by rank; an actual component case displayed neither Combat nor Rank 8. Progress is dropped, and uploader pilot-rank bounds allow 14 where the API documents 13 (Navy differs). Inara remains approval-gated, so these are preactivation defects. [Normalization][provider-reconcile-source] [Inara API][inara-current-docs]

EliteBGS `count=20` is passed on current-reference reads. For the provider's systems route it requests history, not page size, adding historical aggregation which ERIDION discards. Remove it from that capability. EDAstro's one-minute read gate/cache is already conservative; the five-second configured upload interval is not evidence of a read-quota bug. Raven's current stats schema does contain required `buildId`, so its existing identity check is supported. Record positive checks as well as mismatches. [EliteBGS provider source][elitebgs-systems-source] [Raven OpenAPI][raven-openapi]

**Correction:** maintain per-service endpoint/schema/units/time/pagination/acceptance/rate/retry/consent fixtures with source dates. Keep F08–F10 receipt/payload corrections. Documentation/source conformity and authenticated live acceptance remain separate evidence states.

### F25 — P1: native CSV export still rejects the rebranded filenames

`webapp/fieldops.go` emits `eridion-itinerary.csv` and `eridion-private-activity.csv`; the directly wired native `Controller.SaveFieldCSV` permits only `daedalus-itinerary.csv` and `daedalus-private-bgs.csv`. Both new exports therefore return `Invalid field export` before the save dialog. The current web test supplies a fake callback and misses this integration mismatch. [Export caller][field-rpc-source] [Native controller][desktop-controller-source]

**Correction:** centralize the bounded filename contract and test itinerary/activity exports through the real controller boundary, including cancellation/error reporting. This is deterministic source evidence; a native save dialog was not executed in this audit.

### F26 — P2 requested capability: full Spansh planning needs a richer route model

Current ERIDION opens websites, stores local notes and accepts a limited CSV. Route always presents GAME ROUTE first. The importer rejects repeated system names and cannot preserve body, station or trade details. It cannot represent A → B → A or multiple exobiology objectives in the same system. [Current Spansh UI][spansh-ui-source] [Route model][route-model-source]

**Correction:** native forms, one-click selected-commander current location, one shared asynchronous planner-job service, typed ordered stops, persistent progress and active Spansh display precedence. Keep actual NavRoute/target separately. EDMC-SpanshTools and EDMC-NeutronDancer provide concrete source references; no plugin attachment is needed for this work. Their planner endpoints are source-observed contracts, separate from Spansh's published four-resource read API. Validate permitted/current route behavior before claiming live support. [SpanshTools][spansh-tools] [NeutronDancer][neutron-dancer] [Published docs][spansh-docs]

### F27 — P2 requested capability: native controls and docking helper are new work

The native runtime has clipboard support but no game-input executor; legacy binding inspection is not an input backend. Add launcher-only Key Controls, typed actions, one bounded host operation, explicit focus handoff, cancellation/key release and game-state acknowledgement. Landing gear sends one configured toggle; next-target plotting is a calibrated search/select/plot sequence with verified outcome or clearly unverified status. Do not treat an OS input return value as in-game success. [Native controller][desktop-controller-source] [Windows input contract][windows-input]

The owner selected **LandingPad** for native assigned-pad assistance and confirmed that all paired displays have the same trust and configured game actions as the main display. No separate per-device input opt-in is required. Settings remain launcher-only. Windows and Linux input backends require separate acceptance; clipboard remains a useful fallback. [LandingPad][landing-pad] [Wayland portal][wayland-input]

## 4. Targeted open investigations

These are narrow source-backed risks, not promoted to demonstrated native application vulnerabilities:

1. **Null spatial root:** an unchecked map assertion in spatial_format.go can plausibly panic on roots:[null]. The live path requires publisher-signed/hash-matching data. Add a bounded parser no-panic test; malformed trusted input should return an error.
2. **Final 429/503:** after the maximum explicit-refusal retries, dispatch appears able to remove the in-memory job without a terminal durable result. Test four refusal responses and require held/failed with a reason; ambiguous transport results remain unknown.
3. **Failed restore staging:** an interrupted/failed exclusive staging-file creation may block retry. Inject cancellation/disk failure and verify a deliberate recovery path without losing originals.
4. **Smaller-monitor restoration:** saved width/height are clamped independently, so the previous aspect may change. Decide whether the handoff promise means preserving aspect or merely fitting the new work area; keep manual resizing unrestricted.
5. **FSD View System:** its navigation action does not match the normal manual-dismissal selector and may remain covered by the automatic overlay. Add an actual navigation/overlay test.
6. **Distant search inspection:** search selection does not automatically center the camera. Make the interaction explicit, with a deliberate inspect/return-to-follow action if needed, while preserving manual context.
7. **BioScan interrupted cycles:** compare live and historical sample assembly on interrupted/restarted cycles before expanding the algorithm.

The companion plan assigns these to their owning tasks, rather than proposing an unrelated broad rewrite.

## 5. Security and architecture to preserve

The source contains meaningful defenses:

- An ephemeral loopback-only owner service with peer, Host and Origin checks; owner credentials are not viewer credentials.
- Optional HTTPS secondary displays, remote default off, five-minute one-use pairing, global attempt limits, hashed independent viewer grants, expiry, revocation and account binding.
- Launcher-only account, provider, history and other configuration mutations. Main and paired gameplay displays share the same trusted operational capability set under D1. Passive rendering does not schedule downloads or duplicate provider work; explicit gameplay requests use the same configured account/provider policy on every display.
- S256 PKCE, state/expiry/generation validation, fixed endpoints, backend-only tokens, Windows DPAPI and Linux Secret Service storage.
- Per-provider/account/category consent, bounded workers/queues, persisted dedupe, and no silent retry after uncertain transport delivery.
- Pinned Ed25519 trust, strict path/JSON/hash/size validation, bounded archive/decompression/query work and verified caches.
- One journal pipeline, one serialized SQLite writer, WAL/FULL synchronization, atomic checkpoints/state, migration backups, source quarantine and preserved recovery evidence.

Preserve pairing/authentication, account binding, revocation and backend-held credentials while replacing the old lower-trust viewer gameplay restrictions with the confirmed equal-display contract. Settings remain in the launcher for every display. Rendering another screen does not by itself cause extra provider calls; explicit actions are checked against the same policy regardless of the initiating display.

## 6. Performance assessment

**Revision 5 follow-up:** the subsequent code review is now recorded as PERF-01–10 in the [native performance implementation plan](../superpowers/plans/2026-11-22-native-performance.md). It adds disabled-profiler and repeated socket parsing, unnecessary React updates, unchanged-sidecar decoding, duplicate station persistence, shared Go revisions, visibility-aware view scheduling, spatial transfer/prefetch/cache work, canvas painting and evidence-gated persistence changes. These refine F23; they do not introduce measured speedup claims or mark the runtime fixed.

There is evidence of deliberate bounded work and some measured improvements, but the original lightweight goal is not yet supported by a complete current-build, whole-native resource baseline.

The G3C handoff includes real signed data queries, small renderer samples, and synthetic Go-host measurements, and explicitly excludes full Wails/WebView overhead and statistical speedup claims. G3D’s short renderer heap/main-thread sample is likewise not total desktop memory or sustained CPU. The earlier controlled Windows G2L investigation found a smaller material-workload overhead than the noisy CI sample, with mixed-workload costs reflecting new functionality. These are useful measurements with limited applicability. [G3C PR][pr25] [Windows benchmark handoff][g2l-benchmark]

The expanded source audit confirms sticky backend subscriptions, whole-projection work before filtering, repeated local reads, BioScan inference under the service mutex, double parsing of socket responses and avoidable unchanged activity updates. It also confirms existing protections: only the selected page mounts, timers clean up, Operations prevents overlapping reads, ship calculations are cached and map query keys are stable. No percentage saving is claimed from source inspection.

**Required mode contract:** normal mode derives/sends active views and their dependencies; **Low latency: keep views ready** additionally prewarms the finite enabled view models. Journal ingestion, durable events, source/account identity, core Status/FSD, organic recording, active route progress and game-action acknowledgement continue in both. Hidden animation stays suspended. The host computes the union of all displays' demand, sharing one calculation for identical account/query/revision. Closing one screen cannot pause another. Provider cooldown and consent are unchanged.

Next measurements should compare baseline, efficient mode and prewarm mode for the **entire native process tree**, on the same OS/hardware and identical synthetic profiles, for:

- Launcher only and dashboard idle.
- Galaxy/Local map with a stationary view and with interaction.
- 100k-event cold import and 1k-event incremental restart.
- One and two paired displays, reconnect and revocation.
- Longer real gameplay after owner acceptance.

Report CPU, RSS/working set, renderer heap, startup/catch-up latency, request counts, per-topic work, socket bytes, mutex hold time and retained-cache size separately. Include median/p95 source-to-visible latency, tab first paint and tab-to-fresh-content; a stale cached first paint is not a fresh-data result. Establish a measured baseline before setting absolute memory/FPS promises. Require no stale overwrite, zero heavy view-only work after final demand release, shared identical queries, stable map queries and bounded growth. Persistence optimization must preserve durable source history and crash/replay guarantees.

## 7. Ordered action plan

### Batch A — proposed G3G correctness and recovery

Implement F01–F10, F15–F20, F22 and F25 in cohesive data, map, account/display, Exosuit/fleet, catalog and provider workstreams. Include small F21 conformance/coverage corrections and F24 contract fixes while touching their owners. F06 now means the explicitly requested Frontier-first resolver. Keep Inara gated. Record the native performance baseline and domain/subscription interface in this batch; use it to guide Batch B. Include focused retry/parser/restore investigations and correct build identity.

Use targeted failing regression cases during edits, one consolidated final CI gate covering both existing workflows on the final head, one independent whole-branch review, and one consolidated owner acceptance pass. Do not require a complete historical milestone pipeline after every small edit.

### Batch B — performance, capacity and service acceptance

Complete F23's subscription release/domain snapshots/shared computation and launcher normal/low-latency modes before heavier planner/helpers. Address F11–F14, dependency exposure classification, reliable manifests and F24's full service contract ledger. Compare whole-native baseline/candidate on Windows/Linux and one/two displays. Keep release-control configuration as a future explicitly authorized readiness action.

### Batch C — native route planning and deliberate game actions

Implement F26's typed route schema, native Spansh forms/jobs/current-location fill and Spansh-first Route presentation. Then add F27's launcher bindings and single-key actions, followed by the bounded next-target galaxy-map sequence. Every trusted display has the same operational access; no extra paired-device input opt-in is needed. Keep planner job latency distinct from ERIDION rendering/input latency. This replaces the old website-only product scope.

### Batch D — native helper coverage and maintained upstream versions

Implement the selected docking helper and remaining F21 BioScan predicates/variants with source-conformance fixtures. Maintain the broader native capability catalogue, shared event/service interfaces, enable/disable behavior, provenance and builder freshness checks. Validated ship models and other exploration/BGS/colonisation/engineering helpers follow their dependencies and evidence. Downloadable arbitrary-plugin distribution and broad overlays are distinct later scope; the requested native integrations do not need a Python runtime or a general plugin installer.

These are ordered deliverables, not commitments to merge or ship them. Every earlier PR remains draft and unmerged.

## 8. Consolidated owner acceptance checklist

Use the final corrected preview alongside the preserved original installation, with uploads disabled initially.

- [ ] Identify the exact commit/milestone from the app and included README.
- [ ] Start with game closed in single mode: the saved default is selected automatically from cache; start a different journal account and confirm it does not silently replace the default.
- [ ] Import older synthetic/sanitized history after newer history; verify current facts do not move backwards and old rows remain accessible.
- [ ] Restore a disposable two-account backup; both accounts remain paused with publication Never.
- [ ] Inspect all three map panels at 1920×1080 and 1366×768, then a small/portrait paired view; exercise normal resize/maximize/restore.
- [ ] Change XYZ repeatedly through complete, missing and over-budget Local data; controls remain usable and camera scale remains stable.
- [ ] Test a real short/long/near-complete route, clear/replot, incomplete coordinates and manual pan/inspection.
- [ ] Charge/cancel/jump/arrive with and without a route; inspect all FSD projections, View System, manual dismissal and restoration.
- [ ] Search local, missing-coordinate, stale and failed cases; existing EDSM consent remains the only authorization for fallback.
- [ ] Compare conflicting eligible sources: fresh Frontier wins, then journal after Frontier expiry/missing fields, then Inara and all other providers equally under confirmed D0 policy. Check Commander ranks/progress, Fleet and CG personal/global provenance.
- [ ] Scan/sample biology, restart offline and compare a translated-label fixture; candidate identity and sampling isolation remain correct.
- [ ] Compare main and paired gameplay content/navigation, including balances/ranks, Fleet, Exosuit and biological positions; neither display has settings. Reconnect/revoke without foreign-account data or management access.
- [ ] Enter on foot and inspect Commander → Exosuit; the suit never appears as a ship. Verify known equipment offline and live telemetry only during attributable on-foot activity.
- [ ] Compare all known owned ship IDs/counts across current/stored/in-transit states, search beyond the first page, and refresh during pagination; partial source responses do not erase known ships.
- [ ] Reuse existing Frontier authorization for explicitly requested reads; do not revoke a working grant just to complete the checklist.
- [ ] Validate provider publication separately with deliberate recipient/category consent only after receipt/payload fixes; fixtures alone are not live acceptance.
- [ ] Record whole-native resource measurements and label any remaining platform/device limitations.
- [ ] Compare custom/model names and fitted parts across all displays; test the current hull catalog, standard/special variants, omitted name fields and unknown future IDs.
- [ ] Verify older BioScan replies cannot overwrite current body/account data; confirm omitted species and documented numeric boundaries against the chosen rule revision.
- [ ] Export both itinerary and private activity through the actual native save bridge.
- [ ] Switch normal/low-latency modes with two displays on same/different tabs; core route/sample/FSD state remains current, and optional work releases when no display needs it.
- [ ] In the planner batch, use current location, calculate/cancel/resume a plan, preserve A → B → A and body/trade stops, activate Spansh priority and return explicitly to the game route.
- [ ] In the controls batch, use launcher-configured gear and next-target actions from main and paired displays with identical access; verify focus handoff, cancellation, game-state confirmation, concurrent-request handling, session revocation and unsupported-platform fallback. No extra per-device control switch or re-pairing is required.

## 9. Audit conclusion

The immediate gap is dependable behavior across state transitions and source boundaries. The architecture already supports the intended product, but passing isolated tests has not guaranteed chronology, source authority, complete provider receipts, or the actual FSD/Local UI contract.

A focused correctness batch, an early measured performance pass, native Spansh/game-control work and source-verified helper expansion are the ordered next steps. Builder freshness and algorithm conformance are separate requirements. D0–D2 are resolved: Inara shares the equal third-party tier; all paired displays share main-display trust and gameplay actions; LandingPad is the selected docking helper. No clarification remains for these decisions.

No implementation, merge, release, deployment or production upload was performed as part of this audit.

## Source index

All repository source links below are pinned to the audited commit unless they explicitly identify a historical handoff, PR, workflow run or live metadata endpoint.

[pr28]: https://github.com/Jakoczig/eridion/pull/28
[pr25]: https://github.com/Jakoczig/eridion/pull/25
[frontier-audit]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/docs/handoffs/g2m-a-frontier-live-audit.md
[g2l-benchmark]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/docs/handoffs/g2l-fleet-economy-commander-operations.md
[accounts]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/gamestate/accounts.go
[accounts-rpc]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/webapp/accounts.go
[gamefeed]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/webapp/gamefeed.go
[accounts-ui]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-accounts.js
[commander-ui]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-commander-overview.js
[frontier-fleet]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/history/frontier_fleet.go
[provider-reconcile]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/reconcile.go
[operations-ui]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-operations.js
[fleet-ui]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-fleet.js
[socket]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/webapp/socket.go
[native-launcher]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-companion.js
[ship-model]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/shipstate/model.go
[status-parser]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/paritystate/parse.go
[navigation-items]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/lib/navigation-items.js
[capi-contract]: https://github.com/EDCD/FDevIDs/blob/master/Frontier%20API/FrontierDevelopments-CAPI-endpoints.md
[edmc-monitor]: https://github.com/EDCD/EDMarketConnector/blob/main/monitor.py
[journal-manual]: https://hosting.zaonce.net/community/journal/v37/Journal_Manual_v37.pdf
[facts]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/history/reconcile.go#L104
[normalize]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/history/normalize.go#L217
[nav-evidence]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/history/navigation_evidence.go#L21
[persistence]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/gamestate/persistence.go#L108
[restore]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/history/restore.go#L65
[consent]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/history/consent.go#L49
[provider-persistence]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/persistence.go#L48
[provider-accounts]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/accounts.go#L115
[compact-css]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/css/panels/go-navigation-launcher.css#L49
[g3d-css]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/css/go-g3d.css#L1
[terminal]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-terminal.js#L89
[unified-map]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-unified-galaxy.js
[navigation-context]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/webapp/navigation_context.go
[community]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/community/community.go
[goal-cards]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-community-goals.js#L14
[capi-snapshots]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/capi_snapshots.go#L19
[genus]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/explorestate/model.go#L353
[field-bio]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/gamestate/fieldops.go#L105
[bio-rules]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/fieldops/bio.go#L55
[provider-manager]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/manager.go#L574
[provider-batch]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/batch.go
[station]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/station.go#L126
[inara-adapter]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/adapters.go#L306
[journal-reader]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/journal/reader.go#L127
[lockfile]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/package-lock.json
[static-handler]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/webapp/static.go
[legacy-socket]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/service/lib/socket-server.js#L5
[readme]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/README.md
[summary-a]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/docs/PROJECT-SUMMARY.md
[summary-b]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/docs/PROJECT_SUMMARY.md
[packager]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/scripts/build/package-go-preview.cjs#L31
[desktop-ci]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/.github/workflows/go-desktop.yml
[release-ci]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/.github/workflows/release.yml
[current-shipyard]: https://github.com/EDCD/FDevIDs/blob/c7c29f7a13df543d4bc8b947eab29f88e06100fa/shipyard.csv
[current-outfitting]: https://github.com/EDCD/FDevIDs/blob/c7c29f7a13df543d4bc8b947eab29f88e06100fa/outfitting.csv
[catalog-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/catalog/catalog.go
[shipstate-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/shipstate/model.go
[provider-reconcile-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/reconcile.go
[parity-parse-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/paritystate/parse.go
[current-bio-species]: https://github.com/Silarn/EDMC-BioScan/blob/fc6b85c3de48c8fb79fa9e4f79ad780e48f56629/src/bio_scan/bio_data/species.py
[current-bio-algorithm]: https://github.com/Silarn/EDMC-BioScan/blob/fc6b85c3de48c8fb79fa9e4f79ad780e48f56629/src/load.py
[bio-native-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/fieldops/bio.go
[field-ui-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-field-operations.js
[socket-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/webapp/socket.go
[projection-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/gamestate/g2b.go
[inara-current-docs]: https://inara.cz/elite/inara-api-docs/
[elitebgs-systems-source]: https://github.com/elite-kode/elitebgs/blob/d458e6fc2709e178b0d05bf89169de6d8f71717d/backend/routes/elite_bgs_api/v5/systems.js
[raven-openapi]: https://ravencolonial100-awcbdvabgze4c5cq.canadacentral-01.azurewebsites.net/openapi/v1.json
[field-rpc-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/webapp/fieldops.go
[desktop-controller-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/desktop/controller.go
[spansh-ui-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-spansh.js
[route-model-source]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/fieldops/route.go
[spansh-tools]: https://github.com/wuuthradd/EDMC-SpanshTools
[neutron-dancer]: https://github.com/dwomble/EDMC-NeutronDancer
[spansh-docs]: https://docs.spansh.co.uk/
[windows-input]: https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput
[wayland-input]: https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.RemoteDesktop.html
[landing-pad]: https://github.com/bgol/LandingPad
