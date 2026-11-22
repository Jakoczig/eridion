# Native Performance Implementation Plan

> **For agentic workers:** Use `superpowers:executing-plans` when available; otherwise follow these checkboxes in Claude Code directly. Implement coherent batches, with targeted checks during development and one consolidated review/CI gate per batch. This document records proposed work; it does not authorize implementation in the documentation session.

**Goal:** Reduce repeated native-host and WebView work while preserving source correctness, continuous gameplay tracking, durable history, and equal main/paired gameplay.

**Architecture:** Keep the existing Go backend and static React/WebView frontend. Share immutable, revision-scoped semantic models; calculate effective demand before expensive projection, copying, serialization, or painting. Keep JavaScript responsible for local interaction and drawing; a complete rewrite is not justified by the present evidence.

**Tech stack:** Go ([module version 1.26.0][go-module]), SQLite, WebSockets, React 17, static Next export, platform WebViews. Node/Next are build/test tools for the native application; the native runtime does not require a Next server. [Build configuration][build] [Native runtime notes][runtime]

**Spec:** [Correction plan, Task 11 / F23 and Tasks 5, 12, 13, 18](../../plans/ERIDION_Correction_Plan_2026-11-22.md); [integration/performance roadmap, section 4](../../plans/ERIDION_Integration_Performance_Roadmap_2026-11-22.md).

## Starting point and global constraints

- Native source baseline: PR **#28**, `go/g3f-galaxy-viewport-slices`, **`864c724057a7fe28d9d42f5985fa5e1828d3b030`**. All source links below pin that commit. Start implementation on a new stacked branch from the explicitly chosen native/correction head; preserve earlier PRs pinned, draft, and unmerged.
- Recorded main head **`ffc5f663c0cfafff1acd41b1af65cc5b4ff056db`** contains legacy application code plus documentation/configuration. Documentation on main is not the native implementation base. Recheck the chosen branch and `HEAD` before coding; do not transplant native changes onto legacy main.
- Authority is **fresh eligible Frontier live data > attributable journal/Status evidence > Inara and every other third-party provider at one equal tier**. Preserve provenance, eligibility, same-source chronology, coherent field groups, exact identifiers, `json.Number`, null, false, zero, and unknown distinctions.
- Main and every valid paired gameplay display have the same trust, data, and gameplay capabilities. Settings, credentials, consent, pairing, module configuration, and the performance toggle remain launcher-only. Do not restore obsolete owner/viewer gameplay restrictions through cache keys or subscriptions.
- Retain one ingestion pipeline, durable writer, and shared provider budget. Hidden tabs never suspend core state, immutable event history, route progress, organic sample capture, Status/FSD tracking, or action acknowledgement.
- A visible but unfocused second screen stays live. Use visibility/minimization, not keyboard focus, to release display demand. Reject old account, connection, subscription, body, route, and query completions.
- Preserve bounded queues, ordering, atomic commit/rollback, reconnect behavior, revocation checks, and existing cache/download limits. No numerical performance claim precedes comparable measurements.

## Review focus

1. A stale bootstrap or old account reply must not replace a newer broadcast: PERF-02/06.
2. Same-size rewritten, partially written, or replaced files must retain verified last-good state and detect real changes: PERF-03.
3. An irrelevant journal event may still break source continuity; detect that before skipping projection invalidation: PERF-05.
4. A minimized screen and an unfocused visible screen have different demand, while hidden route/sample/FSD tracking remains continuous: PERF-06.
5. Spatial cache reuse or persistence failure must not publish corrupt, truncated, foreign-account, or uncommitted state: PERF-07/08/10.

## Execution order and measurements

| Phase | Scope | Dependency / relationship to correction batches |
|---|---|---|
| A: baseline and local savings | Baseline; PERF-01, PERF-02 activity identity, PERF-03/04, PERF-05 narrow getters/relevance | May accompany correction Batch A without displacing correctness fixes |
| B: shared updates | PERF-02 shared store; PERF-05 domain revisions; PERF-06 subscriptions/modes | Follow Tasks 5/12/13/18: source authority, account identity, equal gameplay, stale-response correctness |
| C: maps | PERF-07/08/09 | Use B's epochs/demand; preserve corrected search and G3F geometry |
| D: evidence-led work | PERF-10 and measured residual hotspots | Separate durability proof before any checkpoint redesign |

- [ ] Record baseline and candidate identity, hardware, OS/WebView/compiler versions, fixture hash, display dimensions, installed map data, cache state, and mode in proposed `docs/testing/g3g-native-performance.md`. Store raw measurements under proposed `docs/benchmarks/native-performance/`.
- [ ] On the same Windows/Linux machines, capture three comparable runs after warm-up: five-minute launcher/dashboard idle; 100k-event cold import; 1k-event incremental restart; live append; large Fleet/loadouts; biology/FSD/route transitions; stationary and interactive maps; one and two paired devices, both identical and different views; minimize/resume/reconnect/revoke.
- [ ] Measure the Go process **and every WebView child**, plus remote browser processes separately: CPU, RSS/working set and growth, allocations, lock hold time, query/decode/build counts, socket frames/bytes, cache hits, checkpoint writes, p50/p95 source-to-visible latency, first paint and time to fresh content. Separate provider job/network time. Compare baseline, normal, and low-latency modes.
- [ ] Treat [G2L controlled Windows results][historical] as historical measurements of their recorded G2K/G2L commits, not current G3F measurements or proof of complete application efficiency. Set numerical budgets only from the new baseline and run variance.

## File responsibilities

Existing paths below describe the pinned native tree. Paths labelled **Create** are proposals, not files already available on main. Keep tests in the existing Go packages and Node test runner; add no test framework solely for this plan.

| Boundary | Existing implementation | Proposed additions |
|---|---|---|
| Client transport and semantic cache | `src/client/lib/socket.js`, `performance-profiler.js`; `components/go-projection.js`, `go-terminal.js` | `src/client/lib/go-view-store.js`; `tests/service/native-performance.test.js`; `tests/helpers/client-transport-harness.cjs` |
| Projection ownership and demand | `go/internal/gamestate/g2b.go`; `go/internal/webapp/socket.go` | `gamestate/view_revisions.go`, `webapp/gameplay_demand.go`, corresponding `_test.go` files |
| Existing caches and providers | `sidecar/snapshots.go`, `providers/manager.go`, `providers/station.go`, `paritystate/model.go`, `explorestate/model.go` | Extend their existing package tests |
| Map data and drawing | `atlas/spatial_view.go`, `spatial_manager.go`; `webapp/spatial.go`; client `go-unified-galaxy.js`, `go-spatial.js`, `lib/go-galaxy-policy.js` | `atlas/spatial_cache.go`, `atlas/spatial_performance_test.go` |
| Mode and evidence | `webapp/companion.go`, client `go-companion.js`; `gamestate/service.go`, `persistence.go` | Measurement document/raw artifacts above; extend existing tests |

Paths abbreviated by Go package are under `go/internal/`; client components are under `src/client/components/`.

## PERF-01 — Parse socket messages once; make disabled profiling cheap

**Evidence:** [`_app.js:7`][app] imports the profiler globally. Its wrappers parse outgoing/incoming JSON at lines 84–105 even when recording is disabled; the guard is inside `record`. Socket dispatch and RPC callbacks parse the same response again. [Profiler][profiler] [Dispatch][client-socket] [Callback][client-callback]

**Files:** Modify `src/client/lib/performance-profiler.js`, `src/client/lib/socket.js`; create the transport harness and `tests/service/native-performance.test.js` above.

**Interface:** Keep public `sendEvent(name, message, timeout)` behavior. Pass the already decoded envelope to the internal callback handler; profiler observations consume decoded metadata and return immediately while disabled.

- [ ] Add a fake-WebSocket/Babel harness using the installed Next compiler and Node assertions. Test one incoming parse, zero profiler-only parsing while stopped, normal success/error replies, malformed frames, request timeouts, cancellation, and reconnect ignoring an old connection.
- [ ] Remove redundant parsing and gate profiler work before inspecting payloads; preserve receiver binding and exactly-once callback cleanup. Keep profiling opt-in and bounded.
- [ ] Run the focused Node file. Acceptance: valid responses resolve/reject as before, malformed input cannot leak pending callbacks, and disabling profiling adds no JSON pass. Measure any runtime effect separately.

## PERF-02 — Stabilize activity context and share view bootstrap

**Evidence:** Activity updates allocate new context objects even when `active` is unchanged. Each `useGoProjection` consumer sends its own initial request; terminal and route components both request navigation. The hook already protects a newer broadcast from a delayed bootstrap. [Transport][client-socket] [Callback][client-callback] [Hook][projection] [Terminal][terminal]

**Files:** Modify socket, projection, terminal, and `go-navigation.js`; create `src/client/lib/go-view-store.js`; extend `tests/service/g2b-panels.test.js` and the proposed performance tests.

**Proposed interface:** `createGoViewStore(transport)` exposes `replaceEpoch({ connection, account })`, `subscribe(key, listener)`, and `read(key)`. Keys contain RPC/arguments and epoch; snapshot identity changes only for a changed semantic revision. Store listeners and single-flight promises belong to the connection, not individual components.

- [ ] In phase A, return `prevState` when the computed activity boolean is unchanged; test unchanged identity and real transitions.
- [ ] After account/display corrections, share one bootstrap and immutable snapshot per identical key. Bound entries to enabled semantic views/current pages; release listeners, pending ownership, and stale snapshots on epoch changes. Keep hooks compatible with the installed React 17 APIs.
- [ ] Preserve live-broadcast-over-initial-RPC protection, including deltas before initial resolution. Test two consumers/one RPC, two distinct queries, unmount/remount, old connection/account replies, failures and retries. Unrelated socket activity must not rerender all consumers.

## PERF-03 — Avoid decoding unchanged verified sidecars

**Evidence:** `Poll` deduplicates only after `read` performs safe reading, hashing, JSON decoding, and Raw copying. [Sidecars][sidecars]

**Files/interface:** Modify `go/internal/sidecar/snapshots.go`; extend `snapshots_test.go`. Preserve `Poll(ctx)` and detached `LastGood(kind)` results.

- [ ] Add `TestPerfSidecarVerifiedReuse` and same-size/same-mtime rewrite, replacement, malformed-to-valid, missing/reappearing, stale timestamp, and partial-write cases; extend current safe-file fixtures.
- [ ] Reuse verified hash/envelope classification only **after** current bounded reads and file-identity/stability checks. Skip repeated decode/copy for identical verified content; still evaluate state/freshness transitions against last-good state. Size/mtime alone cannot establish unchanged content.
- [ ] Run sidecar tests; acceptance is reduced decode/copy counts on unchanged polls, changed bytes still detected, detached results, and no loss of last-good data. Reading/hashing may remain necessary and must not be described as eliminated I/O.

## PERF-04 — Decode provider observations once; save station evidence once

**Evidence:** `Observe` decodes `e.Raw`, then `stationEventLocked` decodes it again. Eligible station publication can call `SaveSidecar` at both lines 71 and 112. [Observer][observer] [Station publication][station]

**Files/interface:** Modify `go/internal/providers/manager.go`, `station.go`; extend `station_test.go`. Change internal `stationEventLocked(e journal.Event, phase journal.Phase, decoded map[string]any)` to consume the existing `UseNumber` result.

- [ ] Test `TestPerfStationSingleDecodeAndSave`, exact large MarketID, consent off/paused, wrong commander/dock/timestamp, duplicate marker/sidecar, failed persistence, queue saturation, and cancellation.
- [ ] Remove redundant event decode and the second save. Retain the first successful persistence point before upload eligibility so consent-off local evidence remains retained. Preserve candidate outcome ordering, recipient/category consent, source continuity and queue idempotence.
- [ ] Run station/provider fixtures with synthetic transports. Acceptance: one save per accepted publication attempt, no queue after failed source persistence, unchanged eligibility and receipts. This is not authority to add provider calls.

## PERF-05 — Narrow getters, then revisions and shared immutable delivery

**Evidence:** Every event marks projections dirty; publication builds/deep-compares all views and clones per subscriber. Sockets compare/encode again. A Status-only read calls full `parity.View`; exploration clears its cache before checking relevance. [Projections][host-projection] [Socket fanout][host-socket] [Parity][parity] [Exploration][exploration]

**Files:** Modify those files plus `paritystate/clone.go`; create `gamestate/view_revisions.go` and `view_revisions_test.go`; extend existing clone/exploration/G2B tests.

**Interfaces:** First add `(*paritystate.Model).Status(key string) StatusView` with detached pointer fields. Later define one revision per semantic domain; cache identity is `(authorized FID, account epoch, domain, canonical query, revision, gameplay schema)`.

- [ ] Add Status mutation-isolation/null tests; use the narrow getter in `sidecarProjectionLocked`. Move exploration relevance invalidation **after** continuity/session checks: irrelevant continuous events reuse cache, sequence gaps/Fileheader/LoadGame/account/system changes still invalidate it.
- [ ] After correction prerequisites, replace global rebuild decisions with dependency revisions. Domain getters must read their own models instead of constructing a full projection to select one field. Filter effective demand before building or copying; share one immutable result and encoded gameplay payload for equivalent keys. Preserve per-principal validity/revocation checks immediately before sending.
- [ ] Capture bounded immutable inputs under the service lock, compute outside it, and revalidate epoch/revision before publication. Cache stable BioScan candidates by account/body/evidence/rules revision; keep live spacing and sample capture separate. Test that telemetry-only updates retain candidates and changed biological evidence invalidates them.
- [ ] Test `TestPerfDomainRevisionDemand`: two matching consumers build/encode once, differing queries remain independent, slow clients retain bounded latest-only queues, and mutations never cross subscribers. Test source/freshness/account changes even when visible values happen to match. Do not expose launcher/admin data through a shared gameplay cache.

## PERF-06 — Replace redundant timers and implement both update modes

**Evidence:** Every socket has a one-second context/provider-status loop; navigation context rebuilds route/history under the service lock. UI timers include [launcher 5s][launcher-timer], [Operations 3s with a pending guard][operations-timer], [Commander 5s][commander-timer], [BioScan 1.5s plus 200ms journal debounce without an in-flight guard][timers], and [Community 15s without visibility gating][community-timer]. Provider `Status()` is a local read, not an external API call. [Host loop][host-socket] [Navigation context][nav-context]

**Files:** Create `webapp/gameplay_demand.go`/`gameplay_demand_test.go`; modify `socket.go`, `companion.go`, `gamestate/navigation_context.go`, and client `go-companion.js`, `go-operations.js`, `go-commander-overview.js`, `go-field-operations.js`, `go-community.js`. Extend `g2b_test.go`, `g2mb_test.go`, and `tests/service/native-performance.test.js`.

**Protocol:** Add explicit replace/release gameplay demand with `connectionEpoch`, server-authorized `accountEpoch`, `generation`, `visible`, and bounded `topics/query` descriptors. Produce atomic snapshot revision R followed only by newer deltas. Encode exact counters as validated decimal strings or opaque tokens, never lossy JavaScript numbers. The client cannot choose a foreign FID or grant itself launcher authority.

- [ ] Centralize domain/provider-status change notifications; schedule explicit freshness/FSD expiry deadlines even without new events. Keep transport liveness timers. Replace view polling only where reliable invalidations exist; otherwise retain bounded visibility-aware single-flight refresh and Operations' guard. Apply Task 18's ordering fix to BioScan first.
- [ ] Persist proposed `displayConfig.LowLatency bool` / JSON `lowLatency`, default false, in launcher configuration. Label **Low latency: keep views ready**. Configuration save failure must leave the previous effective mode unchanged.
- [ ] Define **effective demand = visible-display demand + bounded warm demand**. Off: no warm demand. On: prewarm each finite enabled semantic model once, then update for relevant revisions; reuse host models across screens. Bound pages and the relevant existing map viewport. All-hidden screens may leave the configured warm set active; disabled/no-longer-warmed modules release caches. Warming performs no extra provider fetch, map download, hidden mounting, or painting.
- [ ] Test `TestPerfDemandLifecycle`: identical/different views, one release versus last effective release, repeated replace/release, disabled modules, all-hidden normal/low-latency, visible unfocused second screen, revocation and atomic account switch. Hidden route progress, biological sample capture and FSD start/cancel/expiry must continue in both modes. Resume receives current atomic state; old deltas never replay over it.

## PERF-07 — Separate ready viewport data from prefetch progress

**Evidence:** The galaxy client repeats full spatial RPCs every 600ms while `pending`. Go can return complete cached points while the global pending flag describes adjacent/route prefetch, causing identical payloads and redraws. [Client spatial requests][galaxy] [Spatial view][spatial]

**Files:** Modify `atlas/spatial_view.go`, `webapp/spatial.go`, `go-unified-galaxy.js`; create `atlas/spatial_performance_test.go`; extend G3E/G3D Node fixtures.

**Proposed wire contract:** Extend the strict RPC request with optional `since`; return opaque `viewToken`, `unchanged`, `viewportReady`, and separate `prefetchPending`. An unchanged response omits point replacement; it is not an empty point set. Token identity includes release/content revision, query, and relevant epoch.

- [ ] Test complete viewport with ongoing adjacent prefetch: stable token, no repeated coordinates, preserved client points, no extra paint. Test missing/corrupt viewport assets, query changes, release activation/rollback, reconnect and account epoch invalidation.
- [ ] Use invalidation/single-flight refresh and explicit progress; stop the full-payload polling chain once viewport data is ready. Preserve truthful missing/unknown notices and pending cancellation.
- [ ] Run spatial and map tests. Acceptance: identical ready views neither rebuild/retransmit coordinates nor repaint because unrelated prefetch progresses.

## PERF-08 — Plan missing spatial assets without materializing unused points

**Evidence:** `schedule` invokes the full `query(w)` for prefetch windows then discards coordinates. Only one exact completed query is cached. `ioMu` protects ZIP access, eviction, activation and scratch memory. [Spatial implementation][spatial]

**Files:** Modify `atlas/spatial_view.go`, `spatial_manager.go`; create `atlas/spatial_cache.go`; extend proposed performance tests plus existing spatial security/reliability fixtures.

**Interface:** Add `planMissing(q SpatialQuery) ([]spatialNeed, map[string]bool, error)` under the existing I/O ownership contract; it validates required indexes/assets without constructing point output. Cache parsed indexes/decoded blocks by immutable release/hash/format identity.

- [ ] Test that prefetch planning constructs zero output point vectors, overlapping viewport queries reuse valid decode work, and a hash/release change cannot reuse old entries.
- [ ] Add a byte-bounded LRU with explicit accounting for decoded memory and bounded query scratch; choose and record its cap from phase-A RSS evidence. Preserve current query/download/disk limits and cancellation. Never hold `mu` while acquiring `ioMu`.
- [ ] Run corruption, signed-manifest, malicious ZIP/path, interrupted activation/rollback, eviction and point-budget fixtures. Cached success cannot bypass signed activation, byte/hash validation, coordinate bounds, or whole-result completeness. Publish no truncated point set as complete.

## PERF-09 — Coalesce spatial paints and reuse buffers

**Evidence:** Each `SpatialCanvas` effect resizes the canvas, allocates ImageData and per-point projected arrays, then repeats on ResizeObserver callbacks. Route lookup repeatedly searches context points. There is no current perpetual canvas animation loop to remove. [Canvas][canvas] [Route lookup][route-policy]

**Files:** Modify `go-spatial.js`, `go-unified-galaxy.js`, `lib/go-galaxy-policy.js`; extend `tests/service/g3e-map.test.js`, `g3d-policy.test.js`, and proposed performance tests.

- [ ] Introduce one pending `requestAnimationFrame` paint per dirty canvas, stable dimensions/ImageData, and allocation-free pixel indexing. Cancel pending work on release/hide/unmount; redraw on real data/frame/size/DPR changes. Preserve the 2048 dimension and DPR bounds.
- [ ] Memoize route/context lookup by semantic revision using a name map with deterministic duplicate handling; retain unknown-coordinate behavior.
- [ ] Assert scheduling/buffer reuse with a small fake canvas/ResizeObserver, then inspect actual native rendering in the final visual gate. Preserve G3F fixed panel geometry, equal X/Z scale, XYZ selection versus camera separation, clipping and marker positions. Defer workers/OffscreenCanvas unless main-thread measurements identify remaining blocking work.

## PERF-10 — Profile durable checkpoints before redesigning them

**Evidence:** Sidecar batches serialize a full gob cache before/after changes and call SQL `SaveCache` under the game mutex. Journal commits also retain before/after rollback state. Local provider preparation runs after committed observations under locks; HTTP waits are already outside the provider lock. [Sidecar commit][service] [Durable commit][persistence]

**Files:** Initially extend `gamestate/persistence_test.go` measurements and the performance report only. Any later change to `service.go`/`persistence.go` is a separate durability-scoped batch.

- [ ] Measure serialization, transaction, lock wait/hold, snapshot size and provider-local preparation separately. Use existing `BenchmarkG2JLifecycle` and `BenchmarkProjectionReplay` as supporting microbenchmarks, not whole-app results.
- [ ] If checkpoint work dominates, document a concrete atomic checkpoint/replay contract before changing code. Fault-inject before/after transaction boundaries; prove offsets, source events, derived cache, restart state, consent and publication roll back together. Preserve sidecar-only accepted NavRoute across offline restart.
- [ ] If measured provider preparation dominates, design a bounded queue of **committed** observations with ordering/account/generation revalidation; do not move publication ahead of persistence. Shared decoded journal payloads and native-only frontend pruning are later measured options, not blanket rewrites.
- [ ] Acceptance requires crash/restart equivalence and unchanged durability; otherwise retain the existing implementation. Reduced writes or lock time alone is insufficient.

## Targeted verification and one batch gate

These are future execution commands, not claims that tests ran during planning. Run Go commands from `go/` with the real supported compiler. Run Node commands from the repository root with existing dependencies; do not install a new DOM/test framework. New test paths/selectors below become runnable only after their owning task creates them.

- Transport/cache/demand UI: `node --require ./tests/helpers/no-external-network.cjs --test --test-timeout=15000 tests/service/native-performance.test.js tests/service/g2b-panels.test.js`
- Sidecars/providers: `go test ./internal/sidecar ./internal/providers -run 'TestPerf|TestSidecar|TestEDDNStation'`
- Getters/revisions/demand: `go test ./internal/paritystate ./internal/explorestate ./internal/gamestate ./internal/webapp -run 'TestPerf|TestClone|TestExploration|TestG2B|TestG2MB'`
- Spatial data: `go test ./internal/atlas -run 'TestPerf|TestSpatial'`
- Map semantics: `node --require ./tests/helpers/no-external-network.cjs --test --test-timeout=15000 tests/service/g3e-map.test.js tests/service/g3d-policy.test.js tests/service/g3c-spatial.test.js`
- Durability only when changed: `go test ./internal/gamestate -run 'TestG2J|TestPerfCheckpoint'`
- Supporting benchmarks: `go test ./internal/gamestate -run '^$' -bench 'BenchmarkG2JLifecycle|BenchmarkProjectionReplay' -benchmem -count=3`

Existing tests containing obsolete paired-read-only or fixed-account restrictions must be corrected by Tasks 12/13, not preserved as product requirements or silently deleted. Extend the current React harness deliberately for required fake clocks/listeners; its existing static renderer does not prove real browser scheduling or pixel output.

- [ ] Before each scoped change, add only the regression that proves its behavior; run it failing where appropriate, implement, then rerun the owning checks. Use targeted race checks for changed shared ownership. Do not repeat full CI per checkbox or per measurement run.
- [ ] At the end of each coherent batch, run its required existing lint/build/test gates once, obtain one consolidated review, and record the exact final SHA and actual results. An unavailable compiler/browser/platform is an explicit remaining validation gap.
- [ ] Accept only if identical effective demand shares computation; empty effective demand stops expensive view-only work; memory/queues remain bounded; no stale overwrite, lost history, sample, route or FSD transition occurs; equal displays retain identical source/capability behavior; normal and low-latency measurements are comparable. Low-latency warm demand intentionally survives all-hidden screens; it never permits hidden paints or additional network fetches.
- [ ] Record remaining findings and measured tradeoffs. Preserve draft/unmerged stacked PRs; merging and release are separate owner decisions. This plan itself changes no application behavior.

## Immutable source references

All linked code and historical evidence above use the pinned native commit, so line anchors remain reviewable after new branches advance.

[go-module]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/go.mod#L1-L3
[app]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/pages/_app.js#L1-L7
[build]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/package.json#L5-L28
[runtime]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/docs/testing/g2l-fleet-economy-preview.md#L3-L7
[historical]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/docs/benchmarks/g2l-controlled-windows-review.json#L1-L25
[profiler]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/lib/performance-profiler.js#L75-L115
[client-socket]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/lib/socket.js#L148-L198
[client-callback]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/lib/socket.js#L514-L540
[projection]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-projection.js#L3-L16
[terminal]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-terminal.js#L20-L41
[sidecars]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/sidecar/snapshots.go#L58-L160
[observer]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/manager.go#L382-L416
[station]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/providers/station.go#L19-L119
[host-projection]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/gamestate/g2b.go#L116-L199
[host-socket]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/webapp/socket.go#L110-L251
[parity]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/paritystate/model.go#L70-L85
[exploration]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/explorestate/model.go#L122-L166
[nav-context]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/gamestate/navigation_context.go#L10-L58
[timers]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-field-operations.js#L1-L16
[launcher-timer]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-companion.js#L21-L25
[operations-timer]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-operations.js#L23-L43
[commander-timer]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-commander-overview.js#L1-L8
[community-timer]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-community.js#L1-L30
[galaxy]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-unified-galaxy.js#L27-L75
[spatial]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/atlas/spatial_view.go#L214-L325
[canvas]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-spatial.js#L5-L32
[route-policy]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/lib/go-galaxy-policy.js#L1-L16
[service]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/gamestate/service.go#L431-L462
[persistence]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/gamestate/persistence.go#L48-L168
