# G3L — Reliability, Event Delivery & Build Identity: progress ledger

- **Branch:** `go/g3l-reliability-build-identity`, stacked on `go/g3k-native-performance` at `01639056c59bc5418d0b508bb9166ce0d19e0c26` (exact base; main was not used).
- **Draft PR:** targets `go/g3k-native-performance`.
- **Earlier PRs:** #29–#33 are unchanged and still drafts.
- **Environment:** Linux x86_64 with 8 cores, Go 1.26.10 (CGO, SQLite), Node 24.22.1, headless Chromium 131. GTK4/WebKitGTK and Windows are not available here: the native window and `cmd/daedalus` are built and tested only in CI (Windows-vetted locally with mingw).

## Status

| Area | Status | Commits |
| --- | --- | --- |
| Task 9: journal discovery beyond 4,096 entries | done | `8293d24`, `44b2c6e` |
| Task 9: spatial `roots:[null]` and fuzzing | done | `e95b75a` |
| Task 9: database restore cancellation and retry | done | `4db99e3`, `44b2c6e` |
| Event-driven navigation and FSD delivery | done | `9636a9b`, `44b2c6e`, `65ef7bc` |
| Task 10: build identity | done | `e989ad7` |
| Dependency and security maintenance | done (compatible updates; Next migration deferred) | `e989ad7` |
| Independent review and fixes | done | `44b2c6e`, `65ef7bc` |
| CI on the final source commit, previews | see [CI](#ci-and-previews) | — |

## Task 9: journal discovery and recovery

### Large journal directories (F11)

**Reproduced.** On the G3K base, a directory of 4,200 journals plus other files went to `source_error` with "journal file count exceeds limit" and **0 events** ingested. Nothing was read, including the active journal.

**Change** (`internal/journal/reader.go`):
- Discovery is bounded: at most 65,536 directory entries, with an explicit error beyond that.
- Directories with up to 512 journals are read completely, as before.
- Larger directories are read in batches on each poll:
  1. the two newest journals (active and rolled over) first;
  2. then up to 256 unread journals in chronological order;
  3. then a round-robin slice of settled journals, so changes to old journals are still detected.
- `ReaderStats.Backlog` reports progress.
- Cursors and durable checkpoints are unchanged, so nothing is read twice or lost.

When older journals commit records behind the active one, the service rebuilds derived state once from durable SQLite in chronological order, using the existing maintenance rebuild. Restart neither duplicates records nor rebuilds again. Historical events stay Replay under the timestamp rule.

Review follow-up (`44b2c6e`):
- While the backlog drains, and until the rebuild ran, the summary reports an import in progress. G3K showed "live" from the first poll: a probe saw events 774–1782 of 1800 presented as finished.
- A journal that cannot be opened is retried but does not hold the import open.

**Tests:**
- `TestAuditDiscoveryBacklog`: 4,200 journals plus 50 other files.
  - The active journal is read on the first poll.
  - An append and a rollover are read while the backlog remains.
  - The backlog drained in 17 polls (94 ms) with about 6 MiB heap in use.
  - Every journal is read exactly once, and a late write to an old journal is detected once.
- `TestAuditDiscoveryHardBound`.
- `TestAuditDiscoveryUnreadableJournalNotBacklog`.
- `TestAuditDiscoveryBacklogChronologicalState` (service, 600 journals):
  - survey evidence known only from the oldest journal appears once the backlog drains;
  - every published summary is observed, and incomplete history is never shown as a finished import;
  - restart keeps the same stored event count.
  - It is red without the rebuild and red without the import flag.

**Disproved:** "Older journals overwrite the current system while the backlog drains." Older files never replaced the active projection, because the active-source rules held. Only the import flag was wrong.

### Spatial `roots: [null]`

**Reproduced.** A correctly shaped manifest with a `null` root **panicked** `parseSpatial`, through an unchecked interface conversion. Through a correctly signed pointer/distribution chain the panic came out of `SpatialManager.Update`.

**Change:** the raw roots array and each element are type-checked and rejected with `errSpatial`, which selects the density fallback. Signature, digest and strict-key rules are unchanged.

**Tests:**
- `TestAuditSpatialNullRoot`: null alone, before and after valid roots.
- `TestAuditSpatialNullRootSignedInstall`: signed chain; nothing is installed.
- `FuzzAuditSpatialNoPanic`: bounded, 10 s, about 283k executions, no crash.

A sweep of the atlas parsers for other unchecked assertions found one more. It is safe: strict decoding has already required an object with the right format, and a `null` fuzz seed covers it.

### Database restore and retry

**Reproduced.** `StageRecovery` created `restore.pending.sqlite` with exclusive create and then copied into it. Any failure after creation (cancellation, VACUUM error, consent pause, verification) left the file behind, and **every later retry failed with "file exists"** until it was deleted by hand. The original database was never touched, so there was no data loss.

**Change:**
- Each attempt stages into its own private `restore.staging-<nonce>` file, verifies it, and then promotes it before writing the intent.
- On failure, only that attempt's own files are removed.
- A complete stage that matches its intent is refused ("restart to apply").
- Any other staged file is unexplained. It is moved byte-for-byte to `backups/unexplained-stage-<time>/`, never overwritten or deleted. Orphan `-wal`/`-shm` files are treated the same way (`44b2c6e`).
- Staging is serialised.

**Tests** (all red before, except as noted):
- cancellation, then retry;
- injected filesystem errors after the copy, consent, verify and promote steps, then retry;
- an unexplained stage is preserved and a complete stage is not replaced;
- killed staging;
- orphan sidecars;
- concurrent staging (timing-dependent on the old code).

**Disproved:** restart interrupted after promotion. `consumeRestore` already completes from the intent; the test passed on the base.

## Event-driven navigation and FSD

**Before (G3K):** the navigation context (current system, charge/jump/arrival mode, route points, display policy) was pushed by a per-connection 1 s sampler with a 900 ms shared memo. A charge, jump and arrival within one second could be merged or skipped, and they reached displays up to about 1 s late.

**After:**
- A **navigation-context signal** in `gamestate` advances only when an input changes:
  - journal events;
  - a Status flight-mode change (`navcontext.Status` now reports changes);
  - NavRoute;
  - source reset or restore;
  - the display commander;
  - the source, cached and import flags.

  It is checked at the single projection publication point.
- The only scheduled update left is time-dependent: the 60 s mode window (`navcontext.Deadline`).
- Each connection's context goroutine wakes on:
  - the signal;
  - a display-policy change;
  - a demand or visibility change;
  - the deadline.

  It sends the latest value only when it differs from what the connection last received; an explicit read counts as received (`65ef7bc`). It sends nothing while the display is hidden, and the current state on resume.
- The shared memo is keyed by the signal sequence, so one build serves every display for each change, replacing the TTL.
- Account epochs, authorization, latest-only bounded sends, demand and G3K revisions are unchanged. Core FSD tracking stays in `gamestate`, whatever the visibility or mode. The other 1 s ticker items (spatial revision, community consent invalidation, provider status) are unchanged and outside the FSD scope.

**Tests:**
- `TestG3LEventDrivenFSDTransitions`, with a paired and a local display:
  - charge, jump and arrival are each delivered, in under 1 s in total;
  - a fuel-only Status and a delayed, older charge publish nothing;
  - rapid events while hidden deliver nothing, and resume delivers only the current state;
  - reconnect reads the current state.
  - It is **red on G3K**: no push within 900 ms.
- `TestG3LPolicyChangePublishesContext`.
- `TestG3LContextBuildSharedAcrossDisplays`: 2 changes × 4 displays gave 2 builds and 8 pushes.
- `TestG3LModeDeadlineTimerDelivers`: the timer path, delivered within 300 ms of the deadline.
- `TestG3LExplicitReadIsNotRepushed`.
- `TestG3LNavigationContextSignal`, `TestG3LNavigationSignalKeyInputs`, `TestG3LStatusChangeAndDeadline` and `TestG3LExpiredModeReopenedIsChange`.
- Commander switch: the existing `TestAuditCommanderSwitchInvalidatesMainDisplay`. Missing route data is part of the delivery fixture.

**Browser regressions on the event-driven code:**
- G3D, G3E, G3C, G2G and G2H passed. These cover FSD dismissal, no-route FSD, charge/jump framing, manual override with restore of the prior screen, and Galaxy/Local.
- G3D also passed with 600 ms delayed context messages (`G3D_CONTEXT_DELAY`).

### Measured cost against the G3K baseline

`scripts/testing/g3k-perf-run.sh` was run with the same 10k fixture, alternating G3K (`0163905`) and G3L hosts, two repetitions each. Host CPU is in CPU-seconds per case; latency is journal-write-to-visible.

| Case | G3K rep 1 / rep 2 | G3L rep 1 / rep 2 |
| --- | --- | --- |
| live-latency host CPU | 0.97 / 0.96 | 0.93 / 0.80 |
| live-latency p95 (max) | 416 (537) / 416 (558) ms | 350 (524) / 331 (441) ms |
| fsd-transitions host CPU | 1.37 / 1.30 | 1.26 / 1.06 |
| fsd-transitions frames, bytes | 250, 365 KB / 249, 365 KB | 256, 511 KB / 251, 507 KB |
| paired-2-same host CPU | 0.70 / 0.73 | 0.65 / 0.68 |
| paired-2-same frames per display | 182 | 183 |
| hidden display frames (resume case) | 15 / 15 | 17 / 21 |

- In run A, host CPU was lower for G3L in every case. Across both runs it is equal within noise (see the conclusion below). Builds are one per change, shared.
- In run A, p95 latency was lower; run B (below) found it equal.
- **Socket bytes rise in the FSD case** (about +40%). Each context payload is about 24 KB, and G3L delivers the transitions that the G3K sampler merged away. Frame counts are nearly unchanged.
- The **hidden-display extra frames** were traced by recording frame names. Each display open got a duplicate push of its own explicit read. This was fixed in `65ef7bc` and re-measured below.

**Re-measurement after `65ef7bc`** (run B, same method, G3L host rebuilt):

| Case | G3K rep 1 / rep 2 | G3L rep 1 / rep 2 |
| --- | --- | --- |
| live-latency host CPU | 0.98 / 0.86 | 1.14 / 0.90 |
| live-latency p95 (max) | 301 (568) / 301 (507) ms | 301 (706) / 301 (506) ms |
| fsd-transitions host CPU | 1.11 / 1.31 | 1.27 / 1.19 |
| fsd-transitions frames, bytes | 225, 343 KB / 225, 343 KB | 274, 528 KB / 249, 499 KB |
| hidden display frames (resume case) | 15 / 17 | **15 / 15** |

**Conclusion across all eight repetitions:**
- **Host CPU:** live-latency means 0.94 (G3K) and 0.94 (G3L); fsd-transitions means 1.27 and 1.20; paired displays 0.72 and 0.67. Event-driven delivery adds no measurable host CPU. Single repetitions vary by about ±0.15 CPU-s in both builds.
- **Builds:** context builds are one per change, shared by all displays. G3K built one per second while any display wanted context.
- **Latency:** p95 is equal (run B) or better (run A). Most of the latency comes from journal polling, not delivery.
- **Socket traffic:**
  - The FSD case carries about 45% more bytes (about 340–365 KB → 500–530 KB) and up to about 20% more frames. This is the deliberate cost of delivering every charge, jump and arrival instead of sampling, at about 24 KB per context payload. It is bounded by actual transitions, not by time.
  - Steady live updates carry about 13% more bytes.
  - Paired displays are unchanged (+1 frame in 2 minutes).
  - A hidden display gets no context frames once its demand is declared. The frame count is now equal to G3K's.
- **Possible follow-up:** reducing the context payload (sending route points only when they change) would remove most of the byte increase. That is a protocol change, deferred.

## Task 10: build identity

- `scripts/build/go-build-identity.cjs` generates one record (schema `eridion-build-identity/1`) from the exact checked-out commit:
  - full and short revision;
  - version `2.2.0-g3l+<12-hex>`, following the policy `<package version>-<milestone>+<revision>` (correction plan F13; the old G3G/G3E labels are no longer used);
  - milestone, target OS/arch, the commit date as source date;
  - executable and artifact names;
  - CI provenance.

  It refuses a tree with any tracked or untracked change. For pull requests it records the **PR head (built)** and **GitHub's synthetic merge commit (not built)** separately. The desktop workflow checks out the PR head.
- `internal/buildinfo` holds the linker values. `eridion --build-info` prints them together with Go's own `vcs.revision`/`vcs.modified`. It opens no window, profile, assets or network, and it must be the only argument.
- `packageIdentified` fails unless all of these agree:
  - the executable's report (linker and VCS);
  - `BUILD-IDENTITY.json`;
  - the packaged and artifact names;
  - the README milestone;
  - `SHA256SUMS`, which covers the executable, README and manifest.

  The package is re-verified after writing.
- In `go-desktop.yml`, the steps run in this order:
  1. generate the identity;
  2. build with its ldflags and `-buildvcs=true`;
  3. verify `--build-info`;
  4. package;
  5. upload under the generated artifact name, replacing `eridion-g3e-*`.
- **Tests:**
  - identity generation, refusal on a dirty tree, PR provenance, and tampered records;
  - packaging fails on a mismatched artifact or file name, revision, version, VCS revision, modified tree, OS, manifest or README;
  - an altered package fails verification;
  - the workflow-structure test now requires the identity-derived name.

**Local end-to-end example.** A plain clone of `e989ad7`, built with the generated ldflags through a stand-in main that imports `buildinfo` (`cmd/daedalus` needs GTK here), produced:

```json
{ "schema": "eridion-build-identity/1", "product": "ERIDION",
  "version": "2.2.0-g3l+e989ad77937d", "baseVersion": "2.2.0", "milestone": "g3l",
  "revision": "e989ad77937d3ab255e35fa8f091958b330069cf", "shortRevision": "e989ad77937d",
  "sourceDate": "2026-11-25T05:45:31Z", "target": { "os": "linux", "arch": "amd64" },
  "executable": "eridion",
  "artifact": "eridion-2.2.0-g3l-e989ad77937d-linux-amd64-experimental-unsigned-preview",
  "ci": null }
```

The executable reported `vcs.revision = e989ad77937d…` with `modified=false`, and the package contained `eridion`, `README.md`, `BUILD-IDENTITY.json` and `SHA256SUMS`.

**Found while testing:** Go does **not** record VCS information when building from a linked git worktree, which is how this environment works. `packageIdentified` correctly refused that binary (fail-closed). CI uses a normal `actions/checkout` clone.

## Dependencies and release readiness

See [`dependency-triage-2026-11.md`](../maintenance/dependency-triage-2026-11.md) and [`release-readiness.md`](../maintenance/release-readiness.md).

**Dependencies:**
- 45 packages before (7 critical/26 high/9 moderate/3 low), 42 after.
- 38 were development or build only. None reach the native runtime, which has no Node.js at runtime and embeds only the static client.
- Compatible updates for the legacy runtime, each at least 14 days old: axios 1.21.1, ws 8.22.2, follow-redirects 1.16.3, form-data 4.0.6, brace-expansion 5.0.12.
- The Next 12 → 16 static-export migration is deferred to its own validated change. All 17 Next advisories concern its server, which is not used.
- `npm audit fix --force` was not used.

**Release readiness:**
- Main's configuration commits (`eff5fe7`, `40d07eb`, `ffc5f66`) were compared read-only and not merged.
- **Blocker:** the native stack still carries the inherited `release.yml` (manual dispatch, `contents: write`, publishes a stable `v2.2.0` GitHub release). It was not changed or dispatched.
- `main` is not protected and there are no rulesets; repository settings were not changed.

## Independent review

A separate reviewer read the cumulative diff. Its findings and dispositions:

| # | Finding | Disposition |
| --- | --- | --- |
| 1 | The backlog is presented as finished or live, and older batches overwrite state | Fixed (import flag and pending rebuild). The overwrite part was **disproved** by a probe. |
| 2 | An unopenable journal holds the backlog forever | Fixed |
| 3 | `StageRecovery` is not serialised | Fixed |
| 4 | Orphan `-wal`/`-shm` with no stage | Fixed |
| 5 | Re-opening an expired mode window is not signalled | Fixed |
| 6 | The signal relies on a single publication point | Documented |
| T | The timer path and signal-key inputs are untested; some windows are loose | Added `TestG3LModeDeadlineTimerDelivers` and `TestG3LNavigationSignalKeyInputs` |

The perf run found one more issue: a duplicate push after an explicit read. It was fixed in `65ef7bc`.

## Verification

Final source (`65ef7bc` plus this ledger):
- `gofmt` clean.
- `go vet` and `go test -race -count=1` pass on every package except `internal/desktop` and `cmd/daedalus`, which need GTK here. The plain `go test -count=1` run also passes.
- `CC=x86_64-w64-mingw32-gcc GOOS=windows go vet ./...` (the whole module, including `cmd/daedalus` and `internal/desktop`) is clean.
- `npm test`: 219/219. `lint:security` and `lint:go-assets` are clean. `test:go-assets`: 37/37.
- Red→green evidence for every Task 9, event-driven and review fix is recorded with each test above.
- Browser and visual gates on the final source: see the table below.

All 17 browser and visual gates CI runs passed on `65ef7bca9aa4fd3cc977f7b7d32ed0366e216587`, with assets and visual fixture rebuilt from that source and headless Chromium 131. Each exited 0 with `"passed":true`:
- G2D, G2E, G2G, G2H, G2J;
- G2K, G2L, G2M-A, each against its exact rebuilt baseline (all baselines built);
- G2M-B, G2M-C, G2N, G2O, G2P, G2Q;
- G3C, G3D, G3E.

G3D also passed earlier with 600 ms delayed context messages.

## CI and previews

**Final source commit:** `358b91f30dfe0e5ea09f490105a3d27f0c58c051`. The draft PR is [#34](https://github.com/Jakoczig/eridion/pull/34), targeting `go/g3k-native-performance`. Both workflows triggered for this target, because the `pull_request.branches` filter now includes it.

| Workflow | Run | Result |
| --- | --- | --- |
| ERIDION Go desktop | [45566787212](https://github.com/Jakoczig/eridion/actions/runs/45566787212) | success on `windows-latest` and `ubuntu-24.04`: all Go tests, vet, race, stress, native smokes, every browser and visual gate, identity verification and packaging |
| CI (legacy) | [45566787280](https://github.com/Jakoczig/eridion/actions/runs/45566787280) | success |

**Earlier run on `667399a`** ([45564718884](https://github.com/Jakoczig/eridion/actions/runs/45564718884)): Linux passed and Windows failed.
- **Cause:** the 600-journal service test committed only about 8 journals per second on the Windows runner (one full-sync SQLite transaction per journal). It timed out, and because it never stopped its service, the leftover I/O starved three timing tests that ran alongside it.
- **Fix:** `358b91f` lowers the discovery limits for that test and stops the service on every exit path. It is still red without either fix.
- No test was skipped or weakened.

**Unsigned previews** (14-day Actions artifacts from run 45566787212, downloaded and verified):

| Artifact | Contents | Identity |
| --- | --- | --- |
| `eridion-2.2.0-g3l-358b91f30dfe-windows-amd64-experimental-unsigned-preview` | `eridion.exe`, `README.md`, `BUILD-IDENTITY.json`, `SHA256SUMS` (all OK) | `2.2.0-g3l+358b91f30dfe`, windows/amd64 |
| `eridion-2.2.0-g3l-358b91f30dfe-linux-amd64-experimental-unsigned-preview` | `eridion`, `README.md`, `BUILD-IDENTITY.json`, `SHA256SUMS` (all OK) | `2.2.0-g3l+358b91f30dfe`, linux/amd64 |

- Both manifests record `ci.pullRequestHead = builtRevision = 358b91f…`.
- They record `syntheticMergeCommit = c33441c986cb643c7b699a07585feb2ecac87ab6` with `syntheticMergeBuilt: false`.
- On each runner, "Verify embedded build identity" printed the executable's own report: `revision` 358b91f…, `vcs.revision` 358b91f… and `vcs.modified` false. Packaging then cross-checked it against the manifest.
- No release, tag, signing, deployment or upload outside Actions artifacts was performed.

This ledger is committed after the verified source commit. It is documentation only, and its own CI run is linked from PR #34.

## Remaining physical and real-game gaps

- No real game session, and no native Windows WebView2 or Linux WebKitGTK window, were exercised here. Event-driven FSD should be checked on selected owner and viewer screens: no-route and routed charge, jump, cancel and arrival; rapid consecutive jumps; manual dismissal; hide and resume of a paired tablet.
- No real journal folder with more than 4,096 files was used (synthetic fixtures only). A real large folder should show catching-up and then live state, with live play continuing.
- On the Windows runner, durable import ran at about 8 journals per second (one full-sync SQLite transaction per journal). A real folder of 10,000 journals on a similar disk would take about 20 minutes to catch up. Live play continues meanwhile, and history arrives in the background. The per-journal commit cost already existed in G3K and is not a G3L regression. Batching several small journals per transaction would be a separate, evidence-gated change.
- Restore was tested against synthetic databases. A Windows run should check cancel and retry with antivirus locking files.
- Next milestones: Task 20 Spansh, Task 21 game controls, Task 22 LandingPad.
