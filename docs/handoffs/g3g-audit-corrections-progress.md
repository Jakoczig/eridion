# G3G audit corrections — implementation progress ledger

Statuses: `planned`, `implemented`, `targeted checks passed`, `final CI passed`, `owner acceptance pending`, `owner acceptance confirmed`.

## Pins (verified 22 November 2026)

| Item | Value |
| --- | --- |
| Native base | PR #28, `go/g3f-galaxy-viewport-slices`, `864c724057a7fe28d9d42f5985fa5e1828d3b030` (open draft) |
| G3E parent | PR #27, `64ddd186f692e48b2e2c1f5a342be20de5d1ac12` (ancestor of G3F, verified) |
| G3D ancestor | PR #26, `79a034d6a65f9d83272f7655f35ed68f70eeecdf` (ancestor of G3E, verified) |
| Documentation publication (`ERIDION_PLAN_COMMIT`) | `ec89a14dfd52eca55b4653a73f561528d33c8eb6` on `main`; the owner-supplied handoff file is byte-identical to the copy at this commit, and it is the only commit touching that path |
| Implementation branch | `go/g3g-audit-corrections`, worktree created from the native base commit |

Open PR manifest at start: #1–#28 all open drafts; #26/#27/#28 heads match the pins above. No earlier PR was modified.

Only the six planning-packet files were restored from the documentation commit; root README, package and workflow files were not touched.

## Execution environment notes

- Local Go: the session image had Go 1.24.7/1.25.1 only and proxy.golang.org was unreachable. Go `1.26.10` was built from the official `golang/go` tag `go1.26.10` with 1.25.1 as bootstrap. Modules were fetched with `GOPROXY=direct` through a temporary modfile (`golang.org/x/sys` replaced by its GitHub mirror at the same version, `-modfile` outside the repository); `go.mod`/`go.sum` in the repository are unchanged.
- Local Go tests cannot cover `internal/desktop` (GTK4/WebKitGTK headers absent) or `internal/frontend` (needs the real static export). Both run in CI.
- The npm registry is unreachable from this session, so the static export, Node test suite and lints run only in CI.
- Baseline at `864c724`: all other `go test ./internal/...` packages passed locally.

## Slice 1 of project Batch A

| ID | Status | Notes |
| --- | --- | --- |
| Task 0 | implemented | Pins/stack verified, planning packet committed, ledger created |
| Task 1 / F01 | planned | |
| Task 2 / F02 | planned | |
| Task 12 / F15 | planned | |
| PERF-01 | planned | |
| PERF-02 (activity identity only) | planned | |
| PERF-03 | planned | |
| PERF-04 | planned | |
| PERF-05 (narrow Status getter, exploration relevance ordering) | planned | |

Remaining Batch A scope (Tasks 3–8, 13–19, immediate parts of 9–11 and 23, performance baseline measurements) stays planned for following slices.

## Measurements

None recorded yet. No performance percentage is claimed.
