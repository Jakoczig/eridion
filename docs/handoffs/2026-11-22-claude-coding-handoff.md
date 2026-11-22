# ERIDION coding handoff to Claude

Prepared **22 November 2026** for the owner's requested coding handoff. The planning documents are complete; application implementation has not been performed by this handoff.

## 1. Objective and first delivery

Continue ERIDION's native Go/Wails development from the pinned G3F source. Implement the recorded correctness fixes and performance work in coherent batches, then the planned native integrations. Use the source audit as evidence and the plans as the task ledger; do not restart a broad audit or reconstruct prior conversations.

**First coding delivery:** correction Task 0 and project **Batch A**, including its native performance baseline and the small independent fixes from performance phase A. Follow the existing correctness order for chronology/restore/default commander, source resolution, Exosuit/Fleet/catalog and equal displays. The larger shared-update architecture depends on those interfaces. Complete one reviewable batch through its final verification and report before expanding its scope; future work remains listed in the ledger.

This is a portable handoff. If a document mentions a Superpowers skill that is unavailable in Claude's environment, follow the written task steps directly. Installing a plugin or requesting the owner's execution-method choice again is unnecessary. The owner selected Claude for coding and prefers targeted intermediate checks plus one consolidated final review/CI cycle per batch.

## 2. Exact source and documentation baselines

| Item | Required value |
| --- | --- |
| Repository | `https://github.com/Jakoczig/eridion` |
| Native base PR | [#28](https://github.com/Jakoczig/eridion/pull/28), open draft and unmerged when rechecked |
| Native base branch | `go/g3f-galaxy-viewport-slices` |
| Native base commit | `864c724057a7fe28d9d42f5985fa5e1828d3b030` |
| G3E parent | PR #27, `go/g3e-galaxy-geometry-search`, `64ddd186f692e48b2e2c1f5a342be20de5d1ac12` |
| G3D ancestor | PR #26, `go/g3d-unified-galaxy-map`, `79a034d6a65f9d83272f7655f35ed68f70eeecdf` |
| Proposed new coding branch | `go/g3g-audit-corrections`, new worktree based on the native base commit |
| Main before the plan publication | `ffc5f663c0cfafff1acd41b1af65cc5b4ff056db` |
| Documentation revision | Use the immutable publication commit/link supplied with this handoff, and record its full SHA in the implementation ledger. |

**Main is not the native implementation base.** It contains the inherited application plus recent documentation/configuration maintenance; it lacks the native `go/` tree and native scripts. Do not merge main into the native stack or cherry-pick its complete workflow/package replacement. Those files differ by design.

Completed on main only:

- [`eff5fe7`](https://github.com/Jakoczig/eridion/commit/eff5fe719b0a82611255cdf846d166a24a7cc228): ERIDION README.
- [`40d07eb`](https://github.com/Jakoczig/eridion/commit/40d07eb38a1614b6dd63cd1ab3e55556a7f6da15): inherited-file consolidation and build/contribution documentation.
- [`ffc5f66`](https://github.com/Jakoczig/eridion/commit/ffc5f663c0cfafff1acd41b1af65cc5b4ff056db): package/lock root identity, Node range and legacy CI; manual legacy packaging became artifact-only. Dependency resolutions were preserved.

Native artifact identity, dependency maintenance and native release readiness remain open. Read the [configuration maintenance note at its exact commit](https://github.com/Jakoczig/eridion/blob/ffc5f663c0cfafff1acd41b1af65cc5b4ff056db/docs/maintenance/package-and-workflows.md) before adapting overlapping changes. Native/history branches still contain older workflow definitions; do not dispatch them.

## 3. Planning packet and reading order

Bring only these six files from the pinned documentation commit into the new native worktree:

1. `docs/plans/README.md`
2. `docs/plans/ERIDION_Audit_2026-11-22.md`
3. `docs/plans/ERIDION_Correction_Plan_2026-11-22.md`
4. `docs/plans/ERIDION_Integration_Performance_Roadmap_2026-11-22.md`
5. `docs/superpowers/plans/2026-11-22-native-performance.md`
6. `docs/handoffs/2026-11-22-claude-coding-handoff.md`

Read this handoff, the correction plan's constraints/order and selected tasks, the roadmap's matching requirements, and the detailed performance tasks being implemented. Consult the full audit by finding ID. Immutable source links establish the original finding; confirm the relevant code still matches before editing it. A proposed new file or test in a plan is not an existing implementation.

All confirmed owner decisions and needed technical context are included. Prior ChatGPT conversations, private credentials and Library access are not prerequisites.

## 4. Safe checkout setup

- Inspect the checkout, working tree and remote. Preserve any existing user edits. Refresh PR #26–28 metadata and the open draft stack; do not assume that the historical count of 28 PRs is current.
- Verify the three native head SHAs and their expected ancestry. If a head has moved, inspect the changes and report the exact mismatch before selecting another baseline; do not reset a remote branch or silently substitute main.
- Read the **documentation publication SHA from the owner's prompt/link** and assign that exact value to `ERIDION_PLAN_COMMIT`. Verify it contains all six planning paths. Do not use `ffc5f66` as the documentation publication SHA: it predates these files.
- Create the new worktree from the pinned native source. If the proposed branch/worktree already exists, inspect and resume it only if it belongs to this work and has the expected ancestry; do not overwrite it.

The following shell examples assume a repository whose `origin` is the verified ERIDION repository and that `ERIDION_PLAN_COMMIT` has been assigned from the supplied immutable link. Windows users can use Git Bash or equivalent PowerShell commands; do not run a POSIX block unchanged in PowerShell.

```bash
git fetch origin refs/heads/go/g3f-galaxy-viewport-slices
test "$(git rev-parse FETCH_HEAD)" = 864c724057a7fe28d9d42f5985fa5e1828d3b030
git fetch origin "$ERIDION_PLAN_COMMIT"
git cat-file -e "${ERIDION_PLAN_COMMIT}:docs/plans/README.md"
git worktree add -b go/g3g-audit-corrections ../eridion-g3g 864c724057a7fe28d9d42f5985fa5e1828d3b030
```

Enter the new worktree. Copy the six enumerated planning files using `git restore --source="$ERIDION_PLAN_COMMIT" --worktree -- <the six explicit paths>` or an equivalent file/API operation. Do not restore the whole tree or root README/package/workflows. Inspect `git status` and the copied documents, then record the native and documentation SHAs. Keep this preparation in its own documentation commit when coding begins.

Unavailable GitHub write access does not prevent local implementation or creation of a reviewable patch. Report the concrete missing capability and preserve the work. Do not invent a PR, CI result or uploaded artifact.

## 5. Settled product contracts

1. **Source authority:** fresh validated Frontier first; valid journal and attributable Status second; Inara and all other third parties share the third tier. Resolve by field/coherent group after checking account/entity/units; preserve provenance, unknowns and freshness. Frontier CAPI is a queried snapshot, not a continuous telemetry stream. Missing/invalid/expired fields fall back; a valid zero or false does not.
2. **Commander:** persist a default commander and auto-select it in single-commander mode. Keep temporary multi-mode selection separate, including offline startup and account changes.
3. **Displays:** every successfully paired gameplay display has the same selected-account information, navigation and configured operational actions as the main display. No secondary-only private view or extra per-device game-control approval. Pairing, revocation and host account boundaries remain enforced.
4. **Launcher:** all settings, provider/credential/consent controls, pairing, module enables, performance mode and key bindings belong in the launcher. Gameplay screens retain useful view controls and actions.
5. **Identity:** Exosuit is a separate Commander tab. Keep main ship, controlled SRV/fighter and suit distinct; show known suit/loadout offline and attributable live on-foot data. Complete known fleet coverage and resolve custom names, hulls and outfitted modules consistently.
6. **Helpers:** implement the selected **LandingPad** docking helper natively. Spansh planning belongs in the application with one-click current location, shared jobs and active-plan route precedence. Native bioscanning and other helpers use reviewed current upstream algorithms with credits, exact pins and parity tests.
7. **Key controls:** launcher-managed bindings drive deliberate typed actions: landing gear first, then next-target map/search/select/plot. Distinguish input delivery, target selection and confirmed route plotting; never infer success from a sent key alone.

The roadmap specifies edge cases and acceptance. D0–D2 are already resolved; do not reopen them as routine clarification questions.

## 6. Performance work and language boundaries

The [performance plan](../superpowers/plans/2026-11-22-native-performance.md) records PERF-01–10 with source evidence, proposed changes and targeted tests. Its phases are:

| Performance phase | Scope | Relationship to the project plan |
| --- | --- | --- |
| A — remove repeated work | Disabled-profiler/socket decoding, redundant React activity updates, unchanged-sidecar parsing, repeated station decode/save, and narrowly proven Status/relevance fixes. | Small independent work and measurement fit project Batch A; preserve correctness ordering. |
| B — shared updates | Shared projection subscriptions, Go domain revisions, change-driven navigation/provider state, visibility and normal/low-latency modes. | Project Batch B; depends on Tasks 5/12/13/18 for source, account, equal-display and stale-response contracts. |
| C — map efficiency | Stable viewport revisions, unchanged responses, prefetch planning, bounded validated caches and coalesced canvas painting. | After the relevant correctness fixes and before heavier integrations; preserve G3F geometry/camera requirements. |
| D — measured persistence | Profile gob/cache writes, SQLite work and mutex hold times on the actual durable path. Implement a redesign only when evidence supports it and recovery is proven. | A separate evidence-led task; no speculative durability or checkpoint changes. |

The native runtime already has a Go backend and an embedded React/Next static interface. No legacy Node/Next server is launched by that native path. Keep shared data resolution, inference, jobs and state in Go; retain local interaction, DOM/canvas painting and camera gestures in the browser. A full UI rewrite is not the requested first step.

**Demand contract:** effective expensive-view demand is the union of visible-display subscriptions and explicitly configured, finite low-latency warm views. Normal mode is the default and releases view-only work when the last visible consumer releases it. Low latency keeps the enabled bounded warm set current even if all gameplay windows are hidden or disconnected while the host remains running. Turning the mode/module off releases its warm demand. Neither mode paints hidden views or starts extra external fetches. A visible but unfocused paired dashboard remains live; game focus is not document invisibility.

Journal ingestion, immutable history, commander/source state, Status/FSD, organic recording, active route progress and game-action acknowledgments always continue. One host shares calculations and provider budgets; each display can independently choose its tab.

## 7. Implementation and verification discipline

- Use synthetic journals/profiles/transports. Preserve original user profiles, credentials, publisher infrastructure and live installations.
- Preserve exact IDs/decimal values, absent/null/unknown distinctions, field provenance, replay ordering and account/connection epochs. A callback from an old account, query or connection must not update the current view.
- Retain one journal reader, one SQLite writer, bounded queues/caches, latest-state backpressure and safe file replacement checks. Do not weaken database durability, advance checkpoints ahead of their matching durable state, or mutate checked-in migration history.
- Reproduce the behavior being changed with a focused regression or representative measurement, implement the smallest coherent correction, then rerun affected checks. Do not repeatedly run every historical suite after each edit or add tests that merely mirror trivial implementation details.
- For performance, compare baseline and candidate under the same workload/hardware. Count actual reads, parses, encodes, bytes, render commits and paints; record CPU/RSS including WebView children, event-to-visible and tab-to-fresh-content latency. Use three repetitions of representative whole-app cases as specified in Task 11; separate cold import, warm startup, live updates and one/two paired displays. Historical G2L results and microbenchmarks are not current whole-app gains.
- Use the existing native toolchain and workflow: Node `24.22.1`, Go `1.26.10` (module floor `1.26.0`), CGO for SQLite and platform WebView libraries. Verify the actual execution environment. Node commands run from repository root; native Go commands run from `go/`.
- Avoid generic `npm run lint`, which includes mutating inherited formatters. The native branch's intended blocking checks are `npm run lint:security` and `npm run lint:go-assets`; formatting edits must remain scoped.

At the finished batch, run the existing applicable native verification once: unit/security tests, asset checks, shared frontend build, Go tests/vet, platform/race/smoke/browser checks from `.github/workflows/go-desktop.yml` and the native branch's `.github/workflows/ci.yml`. Both workflows accept a PR targeting G3F. A push to the new G3G branch alone does not establish those checks ran.

Prepare a **new draft PR targeting `go/g3f-galaxy-viewport-slices`** when the reviewed batch is ready and the user's coding session has repository write access. Preserve the existing stack; do not force-push, retarget, rebase or merge earlier PRs. Do not dispatch any release/publishing workflow. Do not remove required gates to obtain green CI. Investigate concrete failures and rerun only what the correction requires.

## 8. Completion report and continuation

Maintain a new `docs/handoffs/g3g-audit-corrections-progress.md` in the implementation branch with task IDs, status, source commits, measurements and limitations. Use explicit statuses: planned, implemented, targeted checks passed, final CI passed, and owner acceptance pending/confirmed. Update the existing checkboxes only when their full criteria are met.

For each delivered batch report:

- New branch, base, final commit and draft PR link if actually created.
- Completed correction/PERF IDs and resulting behavior.
- Tests and exact-head CI links, with failed/advisory/skipped checks identified.
- Before/after measurements and their environment; no unsupported percentage claims.
- Preview artifacts, hashes and embedded build identity if successfully built and uploaded.
- A short consolidated real-game/paired-display checklist and the next pending batch.

Physical gameplay, real provider access and key injection are separate acceptance activities. No merge, release, deployment, publisher change, production upload or private-account modification is included in this handoff. Ask the owner only for a genuinely new product decision, baseline drift that cannot be reconciled safely, missing access, or real acceptance input; routine reversible coding and already confirmed decisions should proceed.

## 9. Ready-to-use instruction

The handing-off assistant should supply the immutable URL of this document after publishing it. The owner can then give Claude:

> Continue ERIDION development using the linked coding handoff and its six-file planning packet. Use the document URL's commit as the documentation pin. The native coding base is `864c724057a7fe28d9d42f5985fa5e1828d3b030`, PR #28, branch `go/g3f-galaxy-viewport-slices`. Verify the pins, create or safely resume `go/g3g-audit-corrections`, copy only the planning packet, and begin Task 0 and project Batch A with the small independent performance fixes. Follow the recorded priorities and acceptance criteria. Work in coherent batches, use targeted intermediate checks and one final CI/review cycle, keep the progress ledger current, and deliver a draft PR/build handoff when available. Preserve all previous PRs and the no-merge/no-release/no-deploy constraints. Do not ask again about the confirmed source priority, equal display trust, launcher-only settings or LandingPad choice.
