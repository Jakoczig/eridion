# ERIDION development plans

Updated **22 November 2026**. This is the canonical repository planning set for the next native correction work and its handoff to Claude.

**Status:** planning and handoff complete; native corrections and performance optimizations remain pending. Findings describe native G3F commit `864c724057a7fe28d9d42f5985fa5e1828d3b030`. The documentation is published on `main`, where the inherited application remains separate from the native development stack.

## Read in this order

| Document | Use |
| --- | --- |
| [Claude coding handoff](../handoffs/2026-11-22-claude-coding-handoff.md) | Start here: source pins, branch setup, scope, settled decisions, verification and delivery instructions. |
| [Correction implementation plan](ERIDION_Correction_Plan_2026-11-22.md) | Tasks 0–24 and project batches A–D; the overall implementation order. |
| [Integration and performance roadmap](ERIDION_Integration_Performance_Roadmap_2026-11-22.md) | Confirmed product contracts, source authority, display behavior, native helpers and update modes. |
| [Detailed native performance plan](../superpowers/plans/2026-11-22-native-performance.md) | PERF-01–10, implementation steps, targeted checks and acceptance criteria refining F23 / Task 11. |
| [Full source audit](ERIDION_Audit_2026-11-22.md) | Findings F01–F27, source evidence, qualifications and historical investigation results. Consult the relevant finding for each task. |

The performance plan's phases A–D are subdivisions of performance work, not replacements for the project batches A–D. Small independent performance fixes and baseline measurements can accompany project Batch A. Shared source/account/display interfaces must settle before the shared-update and mode changes. Profile persistence before deciding whether to redesign it.

## Source and publication boundaries

- **Native coding base:** [G3F / PR #28](https://github.com/Jakoczig/eridion/pull/28), branch `go/g3f-galaxy-viewport-slices`, pinned commit [`864c724`](https://github.com/Jakoczig/eridion/tree/864c724057a7fe28d9d42f5985fa5e1828d3b030).
- **Main before this documentation publication:** [`ffc5f66`](https://github.com/Jakoczig/eridion/commit/ffc5f663c0cfafff1acd41b1af65cc5b4ff056db). README, inherited documentation and legacy build configuration were already corrected there; that does not complete native feature or packaging work.
- **This publication:** documents and a README link only. It does not create an implementation branch, modify native runtime code, merge previous PRs, dispatch an installer workflow or deploy anything.

Use the immutable documentation commit supplied in the handoff message. Bring only the six planning files listed in the handoff into the new native checkout. Do not merge main or replace native source, package scripts or workflows with main's legacy equivalents.

## Decisions already settled

Frontier live data has the highest valid source authority, followed by attributable journal/Status evidence, then Inara and every other third-party provider at equal priority. All paired gameplay displays have the same trust and capabilities as the main display. Settings are launcher-only; LandingPad is the selected docking helper. The complete contracts and fallbacks are in the roadmap and correction plan. These decisions do not need another confirmation round.

Track completed implementation against source commits, targeted checks, final CI and actual acceptance evidence. An unchecked planned task, a historical green build, and a source-reviewed optimization are distinct statuses.
