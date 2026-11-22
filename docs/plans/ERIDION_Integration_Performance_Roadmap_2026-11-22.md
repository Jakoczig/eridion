# ERIDION — Integration, Performance and Native Capability Roadmap

**Revision 5 — performance actions and Claude handoff recorded. Date:** 22 November 2026 (Europe/Budapest). **Baseline:** [G3F / PR #28](https://github.com/Jakoczig/eridion/pull/28), commit `864c724057a7fe28d9d42f5985fa5e1828d3b030`. **Status:** audited requirements and implementation design; no application correction, release, provider job or game input has been performed.

This document extends **ERIDION_Audit_2026-11-22.md, Revision 5** and its **Correction Plan, Revision 5**. The audit contains evidence/findings F01–F27; the correction plan contains Tasks 0–24 and repository paths. D0–D2 are confirmed: the equal third-party tier includes Inara; all paired displays have the same trust and gameplay operations as the main display; LandingPad is selected. Settings remain launcher-only. These decisions require no further clarification.

## Documentation status and Claude handoff — Revision 5

The owner requested that the additional performance findings be recorded and the coding work prepared for Claude. This revision publishes the complete planning set in the repository and links the [detailed native performance plan](../superpowers/plans/2026-11-22-native-performance.md) and [Claude coding handoff](../handoffs/2026-11-22-claude-coding-handoff.md). Start with the [planning index](README.md). **Runtime corrections and PERF-01–10 remain unimplemented in this handoff.** Performance opportunities were confirmed by source inspection; no new runtime speedup or G3F benchmark result is claimed.

Keep two baselines distinct:

- **Native implementation source:** G3F `864c724057a7fe28d9d42f5985fa5e1828d3b030`, PR #28, `go/g3f-galaxy-viewport-slices`; still open, draft and unmerged when rechecked on 22 November 2026.
- **Main documentation/configuration baseline:** `ffc5f663c0cfafff1acd41b1af65cc5b4ff056db`. README rewrite `eff5fe7`, inherited-file consolidation `40d07eb`, and package/legacy-workflow correction `ffc5f66` are already published there. Main still lacks the native `go/` tree. These commits do not mark native findings or feature tasks complete.

The original audit observations below describe their pinned source and inspection time. In particular, main's old stable-release workflow has since become an artifact-only legacy packaging workflow; historical native branches retain their own copies. Read the [package/workflow maintenance record](https://github.com/Jakoczig/eridion/blob/ffc5f663c0cfafff1acd41b1af65cc5b4ff056db/docs/maintenance/package-and-workflows.md) before adapting metadata or workflows. Preserve native test scripts and gates; do not merge main into the native stack or replace native workflows wholesale.

## 1. Confirmed requirements

| Requirement | Recorded behavior |
|---|---|
| Information authority — D0 confirmed | Fresh validated Frontier first; journal/attributable Status second; Inara and all other third-party providers equally third. Apply per field/coherent group, preserving provenance and unknowns. The earlier separate Inara tier is superseded. |
| Single commander | Automatically select the persisted default commander, including offline startup. Multi-mode temporary selection does not overwrite it. |
| Gameplay displays — D1 confirmed | Main and paired displays have equal trust, selected-account information, navigation and operational actions. Pairing establishes display trust without an extra per-device game-control switch. Each may view a different tab/map. |
| Settings | All configuration, credentials, consent, pairing, modules, performance mode and Key Controls belong in the native launcher. Gameplay screens retain operational actions and view controls. |
| Suit and ship | Keep controlled entity, actual ship and exosuit separate. Commander → Exosuit shows known suit/loadout and attributable live on-foot values. |
| Fleet and equipment | Show all known owned vessels with explicit coverage; correct custom/model names, fitted-module identity and supported statistics. Unknown variants/statistics remain identified as unknown. |
| Upstream algorithms | Builder checks current primary source versions; native behavior must also be tested for equivalence/completeness. A current pin is not proof of parity. |
| Spansh | Plan in app, fill current location with one click, review/import/activate results and prioritize the active Spansh plan in Route. |
| Game actions | Launcher-configured keys drive deliberate actions such as landing gear and next-target map/search/select/plot. Display a truthful result based on game evidence. |
| Native modules — D2 confirmed | LandingPad is the selected docking helper. Relevant EDMC capabilities become individually controllable native integrations sharing existing ingestion, storage and provider services. |
| Performance | Normal mode performs optional display work for active views. Low-latency mode keeps enabled view models ready. Core game/history processing remains continuous in both. |

The confirmed scope supersedes earlier website-only Spansh, blanket no-game-command and lower-trust paired gameplay limitations. Every trusted display uses the same configured gameplay capabilities and account/provider policy. Settings remain in the launcher for all displays; credentials remain backend-held. This records planned product behavior and does not perform provider publication or game input during the audit. Confirmed D0–D2 are recorded in section 10.

## 2. Identity and algorithm corrections that must precede expansion

### Ship names and fitted modules

The catalog comparison found **38 bundled hull models versus 49 upstream**, and **1,068 module IDs versus 1,243**, at FDevIDs revision `c7c29f7a13df543d4bc8b947eab29f88e06100fa`. This is a dated source comparison, not a count of the user's owned ships. The missing hull labels are Python MkII, Type-8 Transporter, Mandalay, Cobra MkV, Corsair, Panther Clipper MkII, Type-11 Prospector, Caspian Explorer, Kestrel Mk II, Lynx Highliner and Anaconda Mk II. [Source data][hulls] [Source equipment][equipment]

Catalog replacement alone is unsafe: **22 upstream module symbols map to multiple numeric IDs**, including standard/special variants. The current symbol-only map would overwrite earlier rows. The required model has an exact-ID index and a symbol-to-candidate-set index. Resolve CAPI ID against the same commander, vessel and slot before supplementing a journal observation. Never choose performance data by nearest name or size. [Current importer][catalog]

The shared gameplay identity should carry custom name, ship identifier, hull symbol/label, instance identity, source/freshness and missing-data reason. Missing `ShipName` in a partial same-ship observation must preserve known evidence; explicit clearing is separate. Keep module size, slot capacity, rating, variant, stock attributes and observed engineering separate. These changes address naming paths as well as the distinct incomplete-Fleet problem. [Ship state][shipstate] [Current normalization][reconcile]

### BioScan and numerical helpers

The current BioScan and ExploData pins still match their checked public default heads. The actual gap is native coverage: 116 included species records match upstream, but three records defined outside the converted ruleset directory are absent; 74 of the 116 have no fully supported OR branch. The native pressure/gravity conversions and pressure boundary also differ from source. These are source-parity findings, not independent verification of the game's biological model. [BioScan species][bio-species] [Algorithm][bio-algorithm] [Native evaluator][bio-native]

Correct the missing records and stable identities, declare intended numerical semantics, and add conformance fixtures before extending predicates. Regional/stellar/parent/nebula/composition/volcanism/variant support requires its own evidence dependencies. Preserve possible/excluded/unknown states. Keep known scans, predictions, sample spacing, first-footfall eligibility and sale estimates distinct. Cache stable inference by body evidence and rules revision; update live spacing independently.

## 3. Audit of every implemented native service

“Reviewed” below means current source was checked against primary documentation or the provider's maintained implementation. It does not mean an authenticated live transaction succeeded. All source checks are dated 22 November 2026; future builders must recheck changed contracts. The separately audited signed atlas/spatial publisher remains its own trust and delivery system.

| Service and current surface | Evidence and assessment | Required action / acceptance |
|---|---|---|
| **Frontier:** OAuth PKCE; CAPI profile, market, shipyard, fleetcarrier, communitygoals | Existing integration has bounded reads and approved callback. CAPI is a queried snapshot; station shipyard stock is not owned inventory. [CAPI][capi] [Journal/Status][journal] | Apply common authority, exact account/entity matching and coherent normalized fields. Preserve current authorization/cooldown. Correct ship/suit, Fleet, module and current/unknown semantics. |
| **EDSM publication:** journal endpoint | Top-level batch acceptance cannot establish every event's outcome. Primary docs and EDMC client were examined. [Journal API][edsm-journal] [Client][edsm-client] | F08 per-event receipts, actual game version/build and correct transient context. Dynamic discard information may narrow a consented event set; it must not silently broaden publication. |
| **EDSM reads:** system/sphere, stations/market/shipyard/outfitting, private flightlog | Endpoint families and principal methods align. Coordinates may be absent. [Systems API][edsm-systems] [System details][edsm-details] | Correct missing-coordinate fallback and stale query state; retain exact IDs, bounded paging, shared cache and consent. Separate public lookup from account history. |
| **EDDN:** journal/1, commodity/3, shipyard/2, outfitting/2 | Existing station classifier omits valid ordinary inventory. [Schemas][eddn] | F09 complete-or-suppressed snapshots, schema-specific payloads, exact identifiers and game metadata. Preserve live-only/consent restrictions and source history. |
| **Inara:** profile/CG reads; travel/loadout/materials/ranks/credits publication | Wealth is mapped to cash; rank array conflicts with Commander DTO; progress drops; pilot rank bounds need correction. Per-event 200/202 and exact commander matching already help. [API documentation][inara] | Fix cash/assets/debt, common rank/progress DTO and rank-family validation; distinguish 204 no-result. Protect absent-loadout reset semantics. Preserve application approval and explicit publication gates. |
| **EDAstro:** system/GEC reads; ScanOrganic/CarrierStats/CarrierJumpRequest subset | Read path has one-minute persisted provider gate, GEC cache and 429 cooldown; the five-second capability interval applies to uploads. Supported publication is a deliberate subset. [Reads][edastro-read] [Submission][edastro-send] | Preserve rate enforcement. Recheck interested-event feed, payload/context and distinct JSON no-new-observation outcome before extending support. No current retrieval-budget violation was demonstrated. |
| **EliteBGS:** v5 named systems/factions/stations and ticks | Page envelopes align with provider source. `count=20` on systems asks for history, which the native view ignores. [Provider route source][elitebgs] | Remove unsolicited history from current-state lookup; check each route's parameters. Preserve named identity, pagination, error-envelope rejection and stale cache on outage. |
| **Canonn:** four Strapi site collections | Array responses and `_limit`/`_start` are documented. Current site subset is not all Canonn/Odyssey biological coverage. [Documentation][canonn] [OpenAPI source][canonn-schema] | Pin nested system identity/filter fixtures and expose collection scope. Keep reference sites distinct from predictions. Broader Canonn services require separate capability records. |
| **RavenColonial:** system/project/last/stats reads | Current schema confirms stats `buildId` and string date-time `/last`, matching existing identity handling. [About][raven] [OpenAPI][raven-schema] | Use last-change invalidation; retain exact project/system identity. Contributions/completion are different capabilities and remain outside current read scope. |
| **Spansh:** four ID-resource reads plus website/CSV UI | Published read API and plugin-observed planning endpoints are different evidence classes. Current ERIDION has no native planner-job service. [Published docs][spansh-docs] [Planner source][spansh-plotters] | Implement a versioned, bounded job adapter with source/fixture/current-use checks. Do not claim planner quotas or a stable route OpenAPI that was not verified. |

Each service record must include endpoint/method/encoding, source revision/date, account/entity identifiers, units/timestamps, nullable/partial fields, pagination, response-level acceptance, error handling, retry/cancellation, rate budget, caching, consent and evidence status. Preserve proven controls; do not manufacture findings merely because a service has more endpoints than ERIDION implements.

Official documentation is incomplete or difficult to retrieve for some surfaces. EliteBGS provider source supplied parameter semantics where the rendered documentation was unreadable. EDSM official indexed/localized pages plus its maintained EDMC client supplied details where direct English pages returned access errors. Spansh planner behavior is source-observed and needs a current permitted contract check before shipping. These are explicit evidence limits, not successful live tests.

## 4. Performance design

### Recorded implementation actions — 22 November follow-up

The [detailed performance plan](../superpowers/plans/2026-11-22-native-performance.md) expands correction Task 11/F23 into PERF-01–10. It is the implementation checklist; this roadmap keeps the product-level direction. All entries below are pending. Code inspection establishes repeated work, not a measured percentage saving.

| Order | Recorded action | Intended result |
| --- | --- | --- |
| A | Gate disabled profiling and parse each socket frame once. | Eliminate duplicate decoding on every display. |
| A | Avoid unchanged socket-activity state updates; share session-scoped projection subscriptions in B. | Unrelated transport traffic does not refresh expensive views; bootstrap is shared. |
| A | Reuse validated unchanged sidecars after safe-read/hash checks. | Stable Cargo/Materials/route/station files avoid repeated JSON decode/copy without losing replacement or corruption detection. |
| A | Reuse decoded provider events and persist each eligible station sidecar once. | Remove redundant parse/transaction work while retaining consent, deduplication and durable evidence. |
| A/B | Use narrow Status access, relevant-domain invalidation, Go revisions and shared immutable publications. | A flight update does not rebuild unrelated inventories or surveys; paired displays share calculations. |
| B | Replace per-display local status/context polling with shared change notifications; use visibility-aware single-flight view refresh. | Live core state remains current; normal mode works for visible demand and low latency keeps enabled view models ready. |
| C | Separate visible spatial readiness from background prefetch and return unchanged responses by view revision. | A complete map is not repeatedly transferred and repainted while neighboring regions download. |
| C | Plan missing spatial assets without constructing discarded point arrays; reuse bounded validated index/block caches. | Reduce repeated Go decompression and query work across nearby volumes. |
| C | Reuse canvas buffers, coalesce paints, memoize route lookups. | Reduce browser allocation and redundant drawing while preserving exact XYZ/camera semantics. |
| D | Profile full-cache serialization, SQLite work and lock hold times in the real persistent path. | Make any persistence redesign evidence-led, preserving atomic checkpoints, rollback and source history. |

Keep shared source resolution, game-state derivation, heavy inference, provider jobs and future helpers in Go. Keep layout, local interaction and canvas/DOM painting in the shared browser interface. Native G3F already embeds static frontend assets and runs a Go service; Node/Next are build dependencies rather than a native Node server to remove. A full interface rewrite, a binary transport conversion, additional workers and a native-only frontend build are later options only if measured evidence justifies them.

Effective demand is visible-display subscriptions plus the explicitly configured, finite low-latency warm set. Normal mode releases view-only work after the last visible consumer. Low latency retains its enabled warm set even with all gameplay windows hidden/disconnected while the host runs; disabling the mode or module releases that warm demand.

Normal mode and low latency must never stop journal/history collection, flight/FSD transitions, organic sample recording, active route progress or action acknowledgments. Keep hidden animation paused in both modes. A visible but unfocused second-screen dashboard stays live. Extra displays and low latency must not multiply external API requests or provider limits. Full paired-display parity remains governed by the confirmed trust contract and correction Tasks 5/12/13/18.


### What the audit established

Only the selected React page is mounted. The main missing lifecycle is on the host: requested topics stay subscribed until socket closure; every display receives a full projection subscription; navigation/exploration broadcasts are unconditional. Whole-projection construction/cloning occurs before a compact Status message is selected. BioScan repeats stable inference under the main service lock. The actual component harness also reproduced an older BioScan response replacing newer data. Operations has a working in-flight guard, and both panels clear their timers on unmount. [Socket][socket] [Projection][projection] [Field reads][field-ui]

Existing optimizations must survive the change: a single journal reader/writer, bounded latest-only derived channels, unchanged-state suppression, compact flight deltas, stable/debounced map keys, ship-statistics caching, shared provider cooldowns and bounded socket queues. There is no evidence that every hidden tab currently renders or that ship statistics recalculate on every Status event.

### Launcher setting

**Low latency: keep views ready — Off by default.** This name describes the intended faster tab activation; native measurements must establish its actual benefit.

| Work | Off: efficient updates | On: keep enabled views ready |
|---|---|---|
| Journal, durable history, account/source evidence | Continuous | Continuous |
| Core Status/FSD, active route progress, organic sample recording, action acknowledgement | Continuous | Continuous |
| Optional view calculations | Visible views plus dependencies | Finite enabled view models maintained at bounded cadences |
| Rendering | Visible view; hidden animation paused | Same; warming data does not require hidden DOM rendering |
| Tab activation | One current snapshot followed by deltas | Reuse/validate warm snapshot, then deltas |
| API policy | Shared cache, consent and provider cooldowns | Identical; no per-tab or per-display polling expansion |

“All views” means enabled view models with bounded pages and a relevant viewport. It does not mean every history page, every star or every plugin. Disabled modules remain disabled. A minimized screen drops optional demand; another visible display continues receiving its own data.

### Host architecture

Each connection advertises active route, visibility, topic interest and subscription generation. The host calculates the union of demand per commander/account epoch and permitted read scope. Two displays on the same Fleet view share one calculation. Fleet on one and BioScan on another keep both current. Closing one display releases only its demand.

A legitimate commander selection in the launcher changes the host-authorized account context for all trusted displays atomically. Invalidate old snapshots/actions and update any legacy FID-bound grants through that switch; it must not demand a second device permission or re-pairing. Arbitrary client-chosen foreign-account queries remain invalid on both main and paired displays.

Use explicit subscribe/replace/release operations and monotonic domain revisions. Capture immutable inputs under the service lock; do bounded work outside it; revalidate account/revision before publishing. Domain getters must avoid constructing unrelated projections. Filter demand before cloning/serializing; share serialized bytes only for equivalent account/read scopes. Caches remain finite and keyed by actual input revisions.

On activation/reconnect, supply an atomic snapshot and then later deltas. Discard responses from old accounts, bodies, routes, queries, connections or subscription generations. Cancellation helps but cannot replace revision checks. Replace repeated unchanged view queries with invalidation where available; otherwise retain in-flight guards and visibility-aware polling. Parse each WebSocket frame once and preserve state identity for unchanged activity flags.

Profile status checkpoint serialization separately. A possible optimization is a distinct volatile telemetry cache or bounded checkpoint coalescing, but only after proving offsets, journal durability, restart state and crash recovery. Dropping immutable source events is not a performance option.

### Measurement and gates

Measure baseline, candidate normal and candidate low-latency on the same Windows/Linux hardware, WebView version, fixture and display sizes. Include the host and complete WebView process tree. Use launcher idle, one view, two identical views, two different views, minimized/reconnected displays, map interaction, 100k-event import, 1k-event restart, large Fleet/loadouts, biology and FSD/route transitions. Record three comparable runs after warm-up.

Report CPU, working set/RSS and growth, allocations/lock hold time, query/candidate calculation counts, socket frames/bytes, cache hits, checkpoint writes, median/p95 source-to-visible latency, and tab first paint versus time to fresh content. Provider route-computation time is a separate metric.

Immediate pass conditions are concrete: no stale response overwrite; no lost history/sample/route transitions; zero expensive view-only work when effective demand is empty (visible demand plus the finite low-latency warm set); one shared computation for identical account/query/revision; no repeated stable-map query; bounded queues/memory; and no hidden animation. Set numerical CPU/memory/latency budgets from the actual baseline. This audit does not claim measured native speedups.

## 5. Spansh planning and route behavior

The identified references are **EDMC-SpanshTools v1.1.2** (latest release checked, 16 July 2026) and **EDMC-NeutronDancer v3.0.0** (9 November 2026). They provide real in-app planning and route workflows. SpanshTools source inspected at `c64ee2b8480b521e0c633b212133b81188d8b8ff` is a default-branch source pin, not asserted byte-identical to its release tag. [SpanshTools][spansh-tools] [NeutronDancer][neutron]

### User flow

Select planner mode → **Use current system** or enter origin → choose destination/options and a known ship profile → **Calculate** → review progress/result → **Activate route**. The current-system button uses the existing selected-commander location and displays freshness. It does not continually overwrite manual input or cause a provider request just to fill a form. Carrier origin is explicitly labelled separately.

Provide real validated controls per supported mode: neutron/galaxy first, riches/exobiology/carrier next, trade/tourist as their schemas are verified. Unknown jump-range/equipment assumptions require visible input. Preserve useful manual website/import/export fallbacks. Do not display a form as supported while its adapter silently drops options.

### Planner adapter

| Observed family | Source-observed route |
|---|---|
| Neutron | POST `/api/route` |
| Galaxy / exact ship route | POST `/api/generic/route` |
| Riches / exploration | POST `/api/riches/route` |
| Exobiology | POST `/api/exobiology/route` |
| Fleet carrier | POST `/api/fleetcarrier/route` |
| Asynchronous result | GET `/api/results/{job}` |

The exact host, query-versus-form encoding and body/result fields vary by family and must be pinned in fixtures. These are upstream-client observations, not a published quota/SLA guarantee. Current plugin polling choices are examples, not ERIDION's automatic rate policy. Validate permitted use/current behavior before shipping. [Planner functions][spansh-plotters] [Job client][spansh-web]

One host job records commander, planner kind, input hash, request revision and provider job ID. Any trusted main/paired display can initiate the same planning, import, activation, progress, export and copy operations under the same account/provider policy. Handle direct result, queued/running, completed, rejected, failed, expired and locally cancelled states. Bound requests, polling/backoff and result size; honor rate-limit responses. Do not blindly retry a POST after uncertain submission. Local cancellation stops local polling and result adoption; it does not claim remote deletion without a server contract. Account/input changes invalidate late results. Additional displays viewing a job never resubmit it.

### Route data and precedence

Route identity is routeID/kind/revision; stop identity is stable stopID plus ordinal. Store exact system address/name and optional body, station, market, trade or carrier fields. Preserve repeated visits A→B→A and multiple biological bodies within A. A ship jump confirms travel, not every body scan or trade objective. Matching objective events or explicit manual completion advance those objectives. Selecting or copying a target never completes it.

The shared Route projection selects **active usable Spansh plan → otherwise game NavRoute → otherwise no route**. A completed active plan stays visibly completed until ended/suspended. Header, next stop, progress and destination all use that selection. Display the actual game target separately when different; an intended plan must not masquerade as an already plotted game route. Preserve game observations for FSD/arrival logic and suspend/end explicitly to return to game-route presentation.

## 6. Launcher Key Controls and host game actions

### Configuration

Add Launcher → **Key Controls** with supported binding detection, read-only game preset import, explicit overrides, missing/conflicting binding states, platform capability, calibration and emergency cancel. Do not modify the game's bindings automatically. Parse bounded XML without external resources. Joystick-only bindings require a configured supported alternative; Hungarian and other keyboard layouts need explicit key-code versus text handling.

Gameplay screens contain action buttons, not settings: **Toggle landing gear** on Ship; **Copy to game / Plot next target** beside the active next stop. Keep a separate plain host-clipboard copy action for manual use.

### Action execution

Use typed actions resolved on the host, with selected account/context, expected route/stop revision and operation ID. No arbitrary shell or unrestricted key stream is exposed to a display. One operation runs at a time. A double click or stale page cannot toggle twice or plot a changed stop.

A local click moves focus away from the game, so the operation needs an intentional focus handoff or calibrated countdown, followed by validation of the actual game window. Check context before each step and cancel on focus loss, game exit, account change, route change, permission revocation, timeout or manual stop. Release only keys held by the operation and account for user-held modifiers. Keep diagnostics to operation/state/result rather than clipboard history.

Landing gear sends one configured toggle and waits for a new matching Status flag. Timeout means unconfirmed; it must not send a speculative second toggle. The route action reserves the exact next stop, focuses game, opens Galaxy Map, enters/searches exact system, selects and plots, then requires fresh post-action NavRoute evidence with the intended destination before reporting a successful plot. A changed FSD target alone proves selection; it may be an intermediate hop and does not prove plotting. `GuiFocus` helps with map context, but journal data do not expose every search/result UI step. Those transitions need bounded calibration. Report **target selected; plot unverified** only when fresh matching selection evidence exists. Otherwise report **input sent; selection/plot unverified**, or the actual failure/cancellation state, and offer manual completion. No automatic jump or flight movement follows.

Windows `SendInput` has integrity and existing-key-state constraints; success reports inserted events, not game outcome. Linux Wayland requires a supported compositor/portal session and separate platform acceptance. Keep capability detection and manual fallback instead of promising all desktop/Proton combinations. [Windows documentation][sendinput] [Wayland portal][wayland]

**Confirmed D1 — equal display trust:** every successfully paired display can use the same globally configured gameplay actions as the main display, including landing gear and next-target plotting. There is no extra per-device **Allow game controls** switch, approval or mandatory re-pairing. Existing valid paired sessions inherit the common trusted-display capability set. The host uses one action executor and the same focus/context/revision checks for all initiators; simultaneous requests receive coherent busy/cancel/outcome handling. Pairing revocation or global control disable stops applicable actions. Settings remain launcher-only on every gameplay screen.

## 7. Native capability catalogue

Use the current [EDCD plugin registry][registry] as a discovery source and inspect each actual repository/license/version before porting. Registry metadata is not proof that a listed version is the latest upstream release. SpanshTools, NeutronDancer and LandingPad releases were independently checked; broader entries below are candidates and behavior references, not installed functionality.

| Capability | Reference | Native direction |
|---|---|---|
| Spansh planning/progress | SpanshTools; NeutronDancer | Shared native planner/route service, not separate competing route engines. |
| Assigned docking pad | [LandingPad][landing] | Event-driven pad/layout view. Latest checked release v2.6.0, 11 October 2026. |
| Docking refusal explanation | [Docking Denied Reason][docking-denied] | Separate small capability if wanted; exact source still needs inspection. |
| BioScan | [EDMC-BioScan][bioscan] | Complete declarative native inference and sample support with uncertainty. |
| Exploration data/value helpers | [Pioneer][pioneer], [ExploData][explodata], [ExplorerLite][explorerlite] | Compare overlapping coverage; share one survey model and validated estimates. |
| Route display details | [EDMC-NavRoute][navroute] | Merge useful fields into the one navigation model. |
| BGS/Powerplay activity | [BGS-Tally][bgstally] | Extend existing ledger; outbound reporting remains separately controlled. |
| Colonisation materials | [Architect Tracker][architect] | Compare against existing native construction requirements. |
| Surface navigation | [EDPlanetNavigator][planetnav] | Bounded event-derived heading/distance and optional view. |
| Carrier navigation | [GalaxyGPS][galaxygps] | Carrier-specific position/route assumptions, shared CAPI service. |
| Engineering planning | [EDMC-Engineering][engineering] | Evaluate capability and external service contract before enabling. |
| Wider tool comparison | [ED-Multitool][multitool] | Avoid duplicate market/material/route engines. |
| Overlay presentation | [EDMCModernOverlay][overlay] | Later native presentation work after core correctness/performance. |

Each capability has available/enabled/deferred/unavailable state, source/version, dependencies and an independent launcher toggle. Disabling it stops its private calculations/network work while preserving saved user data and common ingestion needed elsewhere. All modules share account/source identity, journal/Status reader, SQLite writer, provider scheduler, route model and display subscriptions.

**Confirmed D2 — LandingPad:** implement the native assigned-pad diagram based on the identified LandingPad source. It reacts to assigned docking pad and station/carrier type and clears on docking/cancel/timeout/jump/session changes. Unknown layouts remain unknown. Docking Denied Reason remains a separate catalogue candidate; it is not part of the selected LandingPad scope. No additional plugin identification is needed. [LandingPad source][landing-source]

## 8. Builder freshness and reproducibility

| Source | Public default head inspected on audit date | ERIDION relationship |
|---|---|---|
| BioScan | `fc6b85c3de48c8fb79fa9e4f79ad780e48f56629` — 23 Sep 2026 | Existing pin current; behavior partial. Latest published release v2.12.2 is older than this head. |
| ExploData | `fd4e188defd6405e46398fb0a54ac8669c9e2189` — 31 Aug 2026 | Existing pin current; inspected source v1.5.1/database12. No latest GitHub release returned. |
| FDevIDs | `c7c29f7a13df543d4bc8b947eab29f88e06100fa` — 30 Oct 2026 | Bundled catalog behind. |
| Coriolis-data | `0db9234b5b9ce8c939ea84133d7ce336eea88e27` — 24 Apr 2026 | Current selected local projection lacks exact upstream revision manifest. |
| EDSY | `bee2ebab24fd563b637c56a0a192a8b450887686` — 21 Nov 2026 | Reference comparison candidate; not imported. |

These revisions were checked through primary GitHub APIs and exact-revision contents. “Current” means the default heads observed then, not every fork or unpublished change. A builder must recheck at implementation time. [BioScan][bioscan] [ExploData][explodata] [FDevIDs][fdev] [Coriolis][coriolis] [EDSY][edsy]

Create `upstream-sources.lock.json` recording repository/path, selected immutable revision, observed release/default head, raw byte hash and any normalized-content hash, check date, license/notice, schema/converter version, and implemented/omitted behavior. Preserve line-ending provenance when acquisition normalizes CRLF; a changed transport hash alone does not prove changed algorithm data.

The builder checks official sources with timeouts, conditional requests and rate handling and reports **current / update available / unable to check**. It generates reviewable changes for added/removed IDs, symbol collisions, label/variant changes, new rule keys/species, algorithm outputs and licenses. Local/offline builds reproduce locked inputs. Release preparation needs a dated freshness check or documented offline exception; runtime never downloads unreviewed code merely because a newer release exists.

Keep licenses distinct. BioScan/ExploData carry GPL-2.0-or-later notices; Coriolis distinguishes Frontier-owned data from MIT-covered code; EDSY has noncommercial/source conditions and separately described game data; FDevIDs metadata does not itself establish a license. Record per-source terms and attribution during refresh. This is an implementation provenance task, not a reason to delay the unrelated corrections. [Coriolis terms][coriolis-license] [EDSY source][edsy-source]

## 9. Delivery order and acceptance

| Batch | Outcome | Depends on |
|---|---|---|
| A — correctness | Source/default/display/Exosuit/Fleet fixes; catalog/identity; receipts; BioScan ordering/conformance essentials; CSV export; performance baseline | Current G3F head; proposed `go/g3g-audit-corrections`, preserving old draft PRs |
| B — performance and contracts | Explicit demand release, shared domain models, normal/low-latency modes, measured full-process comparison, complete service contract ledger | Shared identity/projection interfaces from A |
| C — planning and actions | Typed route model, native Spansh jobs and precedence; launcher Key Controls; landing gear then next-target plot sequence | Correct ship data, shared provider scheduler and B's lifecycle |
| D — helper parity | Chosen docking helper, full sourced BioScan predicates, additional native capability coverage and maintained builder checks | Shared ingestion/projections; relevant source/license/contract fixtures |

Per implemented batch, use targeted meaningful tests during changes, then one whole-branch review, one consolidated final CI run and one practical owner session. Preserve the previous PR stack; no merge/release is part of this audit. Do not rerun all historical milestone gates after every small edit.

Acceptance must include two displays on same/different tabs; account switch and offline restart; older-history import; restore consent; current/unknown hulls and variants; suits and on-foot telemetry; stale response ordering; BioScan language/boundaries; export through native bridge; provider mixed receipts; and full process resource measurements. Planner acceptance adds repeated/body/trade stops, queued/cancelled/offline jobs, active-plan precedence and actual game target mismatch. Input acceptance adds focus handoff/loss, keyboard layout/modifiers, double click, timeout, cancellation and matching observed gear/route result on each claimed platform. Include target-selected-but-plot-failed and multi-hop routes whose first hop differs from the intended destination.

## 10. Confirmed owner decisions — 22 November 2026

1. **D0 — Confirmed:** **Frontier → journal/Status → Inara and all other providers equally**. No hidden preference for Inara within the third-party tier.
2. **D1 — Confirmed:** all paired displays are trusted at the same level as the main display. They receive the same gameplay information and operational actions, including configured game controls, with no additional per-device opt-in. Configuration remains in the launcher for every display; existing pairing/session and account-context checks apply equally.
3. **D2 — Confirmed:** **LandingPad** is the native docking-helper reference and implementation target.

All three clarification points are resolved and recorded in the audit, plan and roadmap. Native performance, authenticated provider success, live route jobs, physical display behavior and game-input outcomes remain implementation acceptance work. Baseline CI/local results are documented in the main audit and do not prove the planned changes already work. The next implementation batch remains Batch A.

## Sources

[hulls]: https://github.com/EDCD/FDevIDs/blob/c7c29f7a13df543d4bc8b947eab29f88e06100fa/shipyard.csv
[equipment]: https://github.com/EDCD/FDevIDs/blob/c7c29f7a13df543d4bc8b947eab29f88e06100fa/outfitting.csv
[catalog]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/catalog/catalog.go
[shipstate]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/shipstate/model.go
[reconcile]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/reconcile.go
[bio-species]: https://github.com/Silarn/EDMC-BioScan/blob/fc6b85c3de48c8fb79fa9e4f79ad780e48f56629/src/bio_scan/bio_data/species.py
[bio-algorithm]: https://github.com/Silarn/EDMC-BioScan/blob/fc6b85c3de48c8fb79fa9e4f79ad780e48f56629/src/load.py
[bio-native]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/fieldops/bio.go
[capi]: https://github.com/EDCD/FDevIDs/blob/master/Frontier%20API/FrontierDevelopments-CAPI-endpoints.md
[journal]: https://hosting.zaonce.net/community/journal/v37/Journal_Manual_v37.pdf
[edsm-journal]: https://www.edsm.net/pl/api-journal-v1
[edsm-client]: https://github.com/EDCD/EDMarketConnector/blob/main/plugins/edsm.py
[edsm-systems]: https://www.edsm.net/en/api-v1
[edsm-details]: https://www.edsm.net/de/api-system-v1
[eddn]: https://eddn.edcd.io/schemas/
[inara]: https://inara.cz/elite/inara-api-docs/
[edastro-read]: https://edastro.com/api-details.html
[edastro-send]: https://edastro.com/api-info.html
[elitebgs]: https://github.com/elite-kode/elitebgs/blob/d458e6fc2709e178b0d05bf89169de6d8f71717d/backend/routes/elite_bgs_api/v5/systems.js
[canonn]: https://docs.canonn.tech/home/about.html
[canonn-schema]: https://github.com/canonn-science/CAPIv2-Strapi/blob/4204b7ade0a960b2a4471a9eb7eae16cab335ec7/api/grsite/documentation/2.4.0/grsite.json
[raven]: https://ravencolonial100-awcbdvabgze4c5cq.canadacentral-01.azurewebsites.net/About
[raven-schema]: https://ravencolonial100-awcbdvabgze4c5cq.canadacentral-01.azurewebsites.net/openapi/v1.json
[spansh-docs]: https://docs.spansh.co.uk/
[socket]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/webapp/socket.go
[projection]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/gamestate/g2b.go
[field-ui]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-field-operations.js
[spansh-tools]: https://github.com/wuuthradd/EDMC-SpanshTools
[neutron]: https://github.com/dwomble/EDMC-NeutronDancer
[spansh-plotters]: https://github.com/wuuthradd/EDMC-SpanshTools/blob/c64ee2b8480b521e0c633b212133b81188d8b8ff/SpanshTools/plotters.py
[spansh-web]: https://github.com/wuuthradd/EDMC-SpanshTools/blob/c64ee2b8480b521e0c633b212133b81188d8b8ff/SpanshTools/web_utils.py
[sendinput]: https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput
[wayland]: https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.RemoteDesktop.html
[registry]: https://github.com/EDCD/EDMC-Plugin-Registry/blob/main/master_plugin_list.json
[landing]: https://github.com/bgol/LandingPad
[landing-source]: https://github.com/bgol/LandingPad/blob/v2.6.0/load.py
[docking-denied]: https://gitlab.com/inutt/edmc-docking-denied-reason
[bioscan]: https://github.com/Silarn/EDMC-BioScan
[pioneer]: https://github.com/Silarn/EDMC-Pioneer
[explodata]: https://github.com/Silarn/EDMC-ExploData
[explorerlite]: https://github.com/dwomble/EDMC-ExplorerLite
[navroute]: https://github.com/Silarn/EDMC-NavRoute
[bgstally]: https://github.com/aussig/BGS-Tally
[architect]: https://github.com/kfpopeye/EliteDangerous
[planetnav]: https://github.com/mcjohnso/EDPlanetNavigator
[galaxygps]: https://github.com/Fenris159/EDMC_GalaxyGPS
[engineering]: https://github.com/majorxp00/EDMC-Engineering
[multitool]: https://github.com/Martin8O/ED-Multitool
[overlay]: https://github.com/SweetJonnySauce/EDMCModernOverlay
[fdev]: https://github.com/EDCD/FDevIDs
[coriolis]: https://github.com/EDCD/coriolis-data
[edsy]: https://github.com/taleden/EDSY
[coriolis-license]: https://github.com/EDCD/coriolis-data/blob/0db9234b5b9ce8c939ea84133d7ce336eea88e27/LICENSE.md
[edsy-source]: https://github.com/taleden/EDSY/blob/bee2ebab24fd563b637c56a0a192a8b450887686/index.html
