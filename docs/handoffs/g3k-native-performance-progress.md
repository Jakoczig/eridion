# G3K native performance and shared update architecture: progress ledger

Statuses: `planned`, `implemented`, `targeted checks passed`, `final CI passed`, `owner acceptance pending`, `owner acceptance confirmed`.

## Pins and setup

| Item | Value |
| --- | --- |
| Base | PR #32, `go/g3j-bioscan-correctness`, `aeb3e9707deb18f5bf0b7dafed3aa55dc2ec2ebf` (draft; both workflows passed) |
| Branch | `go/g3k-native-performance`, isolated worktree created from `aeb3e97` |
| Measurement baseline | `2dc51ee`: `aeb3e97` plus measurement-only instrumentation (no functional change) |
| Draft PR | targets `go/g3j-bioscan-correctness` (see section 7) |

Checks at the start:

- The G3J head matched its pin, and G3I `0f465ad` is an ancestor.
- The G3J worktree was clean, no G3K branch existed, and the draft PR stack #22–#32 was intact.

Main was not used and earlier PRs were not modified.

Environment: Linux x86_64 (AMD Ryzen 5 9600X, 13 GB), Go 1.26.10, CGO with SQLite, Node 24.22.1. GTK4/WebKitGTK and Windows are not available here, so the native WebView window could not be run. Whole-application measurements use the real Go service with headless Chromium 131 as the display (see the performance report). `internal/desktop` is type-checked with `GOOS=windows go vet` and runs only in CI.

## 1. Scope and status

| PERF | Status | Commits | Summary |
| --- | --- | --- | --- |
| Measurement infrastructure | done | `2dc51ee`, `84a66c5` | `perfstats` counters (builds, frames and bytes, context builds, BioScan inference, spatial cache, checkpoint serialize and commit, service-mutex wait and hold); `go/cmd/perf-host`; `scripts/testing/g3k-perf.cjs`, `-run.sh` and `-report.py`. |
| PERF-02 shared frontend store | targeted checks passed | `eb14218` | `lib/go-view-store.js`: one entry per projection and transport (one initial request, one listener, frozen snapshot), stale-reply, account-change and reconnect protection, unchanged values do not re-render, different projections stay independent. |
| PERF-05 Go domain revisions | targeted checks passed | `eb14218`, `435eb43` | Per-domain builders (navigation, exploration, ship, workspace) extracted unchanged. Dirty bits: Status and other sidecars touch only the workspace; NavRoute touches only navigation. Semantic revisions are kept per domain, with flight separate. Only changed, demanded domains are rebuilt. Getters reuse the published value; one JSON encoding per published value is shared across connections, and never for unpublished initial snapshots. BioScan keeps G3J's single-flight cache (no second cache). |
| PERF-06 demand and update modes | targeted checks passed | `eb14218`, `84a66c5` | `setGoDemand` replace/release with increasing generations and validated topics. The host builds the union of visible demand plus the launcher-only **Low latency: keep views ready** warm set (default off). Hidden displays receive no view, context, provider, spatial or summary pushes and get the current state on resume. Consent invalidations always reach every display. Undeclared (older) clients keep the legacy behaviour. One shared navigation-context build per second. UI polls skip hidden windows; BioScan polls only during an active sampling cycle. |
| PERF-07 spatial revisions | targeted checks passed | `0a7fea1` | `viewToken`, `unchanged` (no coordinates; not an empty map), `viewportReady` separate from `prefetchPending`. The client keeps its points on unchanged replies and polls only until the viewport is ready. |
| PERF-08 asset planning and caching | targeted checks passed | `0a7fea1` | `planMissing` validates assets without building point vectors. A 24 MiB byte-bounded LRU holds parsed indexes and decoded blocks, keyed by verified content hash; it resets on activation, rollback and Bubble removal. Signed-manifest, limit, corruption and coverage behaviour is unchanged; the 64 MiB disk budget is untouched. |
| PERF-09 canvas | targeted checks passed | `0a7fea1`, `84a66c5` | Paints are coalesced into one animation frame; `ImageData` is reused; pixel math allocates nothing per point and matches `spatialPixel` exactly; hidden documents are not painted; unmount cancels pending paints; the route lookup is memoized. No OffscreenCanvas or workers (no measured main-thread blocking justified them). |
| PERF-10 persistence profiling | evidence recorded, implementation retained | — | See the performance report, section PERF-10. |
| CI filters | done | `b9ac835` | Both workflows run for PRs targeting `go/g3j-bioscan-correctness`; no gate changes. |

## 2. Regressions and corrections found during the work

| Finding | How found | Correction |
| --- | --- | --- |
| A connection's initial projection (revision 0) could have its encoding shared with a later connection, which would then receive an older snapshot. | Code review of the shared payload cache; `TestPerfInitialSnapshotNotShared` was red. | `435eb43`: unpublished snapshots are encoded per connection. |
| With visibility gating, an EDSM consent change no longer reached a hidden display, which could then briefly show revoked community data on resume. | The g2e browser gate (its map tab is a hidden background target). | `84a66c5`: consent invalidations are always delivered. `TestPerfConsentChangeReachesHiddenDisplays` was red before. |
| A per-subscriber clone was briefly replaced by a single shared clone, which weakened the existing snapshot-independence contract. | `TestG2BBroadcastSnapshotsAreIndependent`. | Reverted before commit: each subscriber still gets its own snapshot. |
| The g3c canvas gate read canvases synchronously after effects. | The g3c browser gate. | The gate now waits for the coalesced paint, then applies the same exact-position assertions. |

### G3D map timing flake (recurred; investigated and fixed)

The Windows-only G3D gate timeout seen in G3J recurred once on the docs-only G3K head `d6f28c2` (run 45489282846); the same code had passed on `74fa4ee`.

- **Evidence.** The failure artifact showed the page still in Galaxy mode after the **Local** click, and no spatial request ever sent. The synthetic response was never involved.
- **Root cause.** The Local control is disabled until the selected commander's centre is known from the navigation context. The gate clicked it right after the Galaxy layout step, so on a slow runner whose context arrived late the click hit a disabled button and did nothing.
- **Deterministic reproduction.** A delayed-response fixture holds navigation-context messages for 6 s. With it, the unmodified gate failed twice at the CI failure point. CPU throttling (6×) alone did not reproduce the failure in six runs.
- **Fix.** The gate waits, within the same 20 s bound, for the Local control to be enabled before clicking, and its failure diagnostics now include that state. A control that never enables still fails. The fixture is kept as an opt-in mode (`G3D_CONTEXT_DELAY=<ms>`); CI runs without it. With the fixture the fixed gate passed twice; without it, it passed as before.
- **Product behaviour is unchanged:** Local stays unavailable until a centre is known.

## 3. Independent review

A separate reviewer examined `aeb3e97..84a66c5` (all slices). It compared the domain builders line by line with the old combined projection, ran targeted Go and Node tests, and confirmed R1 with a probe in a scratch copy. No high-severity findings. All findings are fixed in `041e87b`.

| ID | Severity | Finding | Resolution |
| --- | --- | --- | --- |
| R1 | medium (confirmed) | Demand was computed under the registry lock but applied to the game service after unlocking, so concurrent recomputes could apply out of order and leave the host on a stale mask. A visible display's domain would then stay deferred. | Compute and apply are serialized (`demand.apply`). Test: `TestPerfDemandConcurrentApplyConsistent`. |
| R2 | low/medium | Counters were keyed by client-chosen RPC names, so the counter set could grow without bound. | Only known RPC names are counted; everything else is `rpc.other`. Test: `TestPerfFrameCounterNamesBounded`. |
| R3 | low | Planning trusted a verified record after the file disappeared, so no re-download was scheduled. | A miss in the decode path forgets the record. Test: `TestPerfSpatialStaleVerifiedRecord`, red before. |
| R4 | low | The published workspace could keep Frontier module IDs past the evidence TTL while offline. | Freshness respects the TTL, and a Frontier disconnect drops the evidence. |
| R5 | low | A failed demand declaration was never retried. | Bounded-backoff retry. Store test. |
| R6 | low | The context RPC could return a copy up to 900 ms old after a jump. | The RPC builds a fresh context; only the per-second ticker uses the memo. |
| R7 | low | Declared connections received ship/workspace pushes they had not requested. | Declared demand narrows the requested pushes and never widens them. |
| R8 | low | Building lock-series names on every lock. | The names are precomputed. |
| R9 | low | A reconnect with unchanged topics might not redeclare demand. | The first view after the store empties always declares. Store test. |

Checked and found sound by the reviewer:

- **Domain builders:** they match the old combined projection on every presentation rule.
- **Dirty bits and publication:** the dirty-bit and key coverage, publication and deferral bookkeeping, per-subscriber snapshots and getter clones are all correct.
- **Shared encodings:** identity and privacy hold (the same values went to every principal before).
- **Legacy and protocol rules:** legacy connections are unchanged; `setGoDemand` validation is correct; `setGoCompanion` stays launcher-only; the lock order is sound.
- **Spatial cache:** entries are created only from verified bytes, keys cannot collide, resets and eviction are correct, `unchanged` is never treated as empty, and polling stops once the viewport is ready.
- **Client:** painter equivalence, rAF lifecycle, route memoization, BioScan cycle gating, `visibleInterval` cleanup, and the view store's stale-reply handling.
- **Measurement code:** nothing shipped imports the measurement-only command or scripts.
- **CI:** the change is limited to the branch filter.

## 4. Verification

Local, on `041e87b` (Linux):

- `lint:security`, `lint:go-assets`, `build:go-assets`;
- `npm test` 219/219 (210 before G3K, plus store, canvas, BioScan-gating and review tests); `test:go-assets` 29/29;
- gofmt clean; `go vet` and `go test` on all packages except `internal/desktop` and `cmd/daedalus`; `GOOS=windows go vet ./internal/desktop` clean;
- `-race` on gamestate, atlas, perfstats, explorestate, paritystate and webapp.

Browser gates: all 17 passed on the slice-2 candidate after two corrections:

- **g2e:** found the consent regression described in section 2, which was fixed.
- **g3c:** now waits for the coalesced paint.

After the review fixes, g2d, g2e, g2g, g2k, g2l, g2ma, g2n, g3c and g3e were re-run and passed. The G3D map timing flake from G3J did not recur in any run.

## 5. Measured results

See `docs/testing/g3k-native-performance.md`.

## 6. Limitations and owner acceptance

- **Platform:** measurements are Linux with a headless Chromium display. The native WebView2 (Windows) and WebKitGTK windows were not measured, and paired displays ran on the same host.
- **Not measured:** map interaction with installed signed data, and provider-local preparation.
- **FSD context sampling:** the context is pushed on the existing one-second cadence, so a sub-second synthetic jump can be missed in both builds. Event-driven context pushes are recommended follow-up work.
- **Owner acceptance on real hardware:**
  - idle CPU and memory of the native app with the dashboard open (normal and low-latency);
  - main plus paired tablet on the same and different pages;
  - minimize, restore and reconnect of each display;
  - Galaxy/Local map interaction with installed data;
  - FSD charge, jump and arrival with the automatic display switching;
  - BioScan sampling while the tab is hidden;
  - a 100k-event profile import.

## 7. CI and draft PR

Draft PR: [#33](https://github.com/Jakoczig/eridion/pull/33), `go/g3k-native-performance` → `go/g3j-bioscan-correctness`. Both workflows trigger for it (`b9ac835`).

**Final source commit `41b7c4bb7d46956426e6e4bdfc10fb8138898785`** (all code, the gate fix, the report and raw data), attempt 1, no reruns:

- [Go desktop foundation 45492858676](https://github.com/Jakoczig/eridion/actions/runs/45492858676): success on `ubuntu-24.04` and `windows-latest` (Go tests and vet, race detector, native smoke, every browser and visual gate including G3D).
- [ERIDION legacy compatibility CI 45492859138](https://github.com/Jakoczig/eridion/actions/runs/45492859138): success on all 5 jobs.

Unsigned preview artifacts from run 45492858676:

- `eridion-g3e-Windows-x64-experimental-unsigned-preview` (id 14968466138)
- `eridion-g3e-Linux-x64-experimental-unsigned-preview` (id 14967433372)

They still carry the G3E name because build identity is Task 10.

Earlier runs, for the record:

- `74fa4ee` passed both workflows (runs 45486177740 and 45486177986).
- The docs-only `d6f28c2` hit the G3D gate race on Windows (run 45489282846). It was investigated and fixed in `41b7c4b` (see section 2), with no blind rerun.

The docs-only commit adding this section is verified in the PR description.

## 8. Next batch

Follow-ups from G3K:

- event-driven navigation-context pushes (removes the 1 s sampling limit);
- native WebView2/WebKitGTK and installed-map measurements on the owner's machine;
- a durability-scoped import-throughput batch, if wanted (contract first).

Remaining correction-plan tasks: Task 9, Task 10 (build identity), Tasks 20–22, and the Task 24 acceptance gate.
