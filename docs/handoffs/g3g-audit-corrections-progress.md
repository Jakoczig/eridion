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
| PERF-01 | targeted checks passed | `01ef87f` | One `JSON.parse` per incoming frame (baseline: two); zero profiler parses while disabled. |
| PERF-02 (phase A: activity identity) | targeted checks passed | `01ef87f` | Socket state identity kept when the activity flag is unchanged. |

### Open items carried from slice 1

- **Task 12:** while no default is resolved (several cached commanders, or an unreadable preference file), the provider namespace stays unpinned and follows the journal-active commander, as before this change. The plan asks for provider activity to stay blocked until the launcher choice; that needs an explicit blocked state in the provider manager (a sentinel pin would create a stray account namespace). Remote account binding across a host-authorized switch for paired sessions is unchanged and belongs with Task 13.
- **Task 12:** when `g2e` persistence is disabled, the game service has no profile directory while providers still read the preference file (pre-existing asymmetry, unchanged).
- **Task 1:** the Commander component still shows Frontier-labelled ranks from `facts.ranks`; rank/progress presentation and cross-source authority follow in Task 5. `EffectiveFacts` now makes a small bounded set of indexed lookups per call (fields × sources) instead of one grouped query.
- **Review:** this slice had a self-review of the full diff against the plan and the account/privacy boundaries. The independent whole-branch review in Task 24 is still pending.

Remaining Batch A scope (Tasks 3–8, 13–19, immediate parts of 9–11 and 23, and the whole-app performance baseline) stays planned for following slices.

## Measurements

No whole-application measurements were taken: the session has no Windows/Linux desktop, WebView or game environment. The PERF figures above are operation counts from targeted tests, not runtime or percentage improvements.
