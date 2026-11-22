# Mining: native scope and retained requirements

**Status:** consolidated roadmap and specification, reviewed 22 November 2026. This document preserves useful inherited mining requirements and identifies native gaps; it does not implement or certify new functionality.

**Baselines:** [inherited application `06a6963`][legacy-source] and [native G3F `864c724`][native-source]. These are distinct implementations. The old [Varzeki mining plan][original-plan] describes a Node architecture and initially empty pages that had already been implemented in the inherited application. Its completion checkmarks are not evidence of native parity.

## Current coverage

| Capability | Native G3F evidence | Remaining work |
| --- | --- | --- |
| Prospector observations | Materials, reported proportions, motherlode, content and remaining percentage appear in Status/Log. | Target-quality classification, filters and historical browsing need a defined native contract. |
| Observed mining counts | Prospected records, cracked cores, launched limpets and refined commodity counts are projected from attributed journal events. | Configurable quality and limpet assistance; explicit session-rate definitions. |
| Cargo and ship context | Remaining limpets come from known cargo; weapon pips come from known Status evidence. | Mining-capability checks, low/no-limpet warnings and reconciliation of additional supported events. |
| Rings | The Rings view shows current-system scan-derived names, classes and radii. | DSS hotspot association, mapping-state labels and target highlighting. |
| Durable records | SQLite stores witnessed refined tonnes and conservatively attributable mining-sale receipts. | Extend existing queries where required; do not create another accounting engine. |
| Display access | Native pages and paired transport exist, with some current role restrictions. | Apply ERIDION's confirmed equal gameplay access for all trusted displays. |

The [activity projection][activity], [native mining views][mining-ui] and [historical operations][operations-ui] define this baseline. The native terminal routes Mining Statistics to historical operations before the older parity component; the latter's statistics branch should not be mistaken for the current entry point.

### Recent observations and durable history

The current activity projection retains at most **16 recent prospects**, **64 refined commodity entries** and **64 recent refinement records**, and can report truncation. These are bounded working sets, not a complete mining log. Current-location prospects clear on travel; attributed historical records remain separate.

Historical mining-sale attribution requires a witnessed empty ship-cargo baseline and exclusively refined input in the same verified session. Mixed or unknown cargo does not acquire inferred mining provenance. Refined tonnes, cargo stock, sale receipts and profit are different quantities. Preserve the [existing conservative accounting rules][accounting].

## Retained requirements and corrections

### 1. Ring and hotspot assistance

Join scan and DSS observations to the correct commander, system, body and ring. Preserve canonical commodity identifiers separately from translated display names. Distinguish **not mapped**, **mapped with no relevant hotspots**, and **mapped with reported hotspots**; absence of an observation is not a confirmed zero.

Allow target commodities to highlight relevant results without discarding other observed rings. Any density calculation must identify its inputs and units; it must not become an unsupported prediction of yield. External ring search, if added, belongs to the shared provider service and existing consent controls.

**Acceptance:** scan and DSS events work in either order; repeated observations do not multiply counts; travel, ship changes and commander changes cannot expose unrelated context; cached local facts remain usable without network access.

### 2. Target preferences and limpet assistance

Provide launcher-managed commodity targets, quality thresholds and low/no-limpet warning settings. Derive mining capability from supported fitted-module evidence. Reconcile limpet information with attributable cargo snapshots and documented events rather than blindly accumulating independent counters.

Apply ERIDION's source policy to valid, matching supplied facts: live Frontier first, journal/Status second, and Inara plus other providers at the same third priority. Preserve observation time and source; this policy does not supply fields a provider does not expose.

**Acceptance:** unknown cargo or unknown fitted capability never becomes a definite “no limpets” warning; replay does not inflate usage; fresh authoritative snapshots correct supported deltas; preferences apply consistently across trusted displays.

### 3. Session definitions and history

Define the observed login session, current location, recent observation window and durable history separately. Travel may invalidate local prospect context without deleting historical totals. The inherited plan's contradictory login-only and per-jump reset instructions are superseded by these explicit boundaries.

A tonnes-per-hour metric must identify its denominator. Native G3F measures an observed window from `LoadGame` to the latest mining event; that is not proven active mining time. Missing or discontinuous evidence must remain visibly partial. Reuse the existing history transactions, provenance and bounded cursor queries for additional records.

**Acceptance:** restart, replay, duplicate imports and account switching preserve correct attribution; recent-window limits are visible; historical browsing remains bounded; mixed cargo and missing cost basis cannot produce mining-profit claims.

### 4. Readable layout and navigation

Organize cargo, limpets, prospects, rings and statistics into responsive, readable groups. The inherited central-circle/notch design may be retained as optional styling where it improves readability. Derive any cut-outs and borders from shared rendered geometry; do not reinstate a fixed `110rem × 56rem` canvas, hand-tuned pixel coordinates or whole-page scaling as a universal layout rule.

Decorative SVG must not intercept pointer input or obscure content. Keyboard focus must remain visible, and touch controls must remain usable after resizing. Telemetry updates must not remeasure the entire layout.

Keep configuration in the launcher. Main and paired displays must have equivalent gameplay information and operational access for the selected commander. Optional mining-context automatic navigation should respect manual navigation, remain within the mining workspace and use launcher preferences.

**Acceptance:** 16:9 desktop, narrower resized windows and paired touch displays show readable groups without clipped controls; decoration cannot block input; repeated telemetry does not trigger whole-layout measurement or unwanted tab switching.

### 5. Shared-state performance

Extend the existing Go ingestion, projections and SQLite history. Add no duplicate journal reader, database writer or browser-side accounting engine. In normal mode, display work should follow the visible tab; optional low-latency mode may prepare all views while retaining bounded caches and shared computation.

**Acceptance:** opening additional displays does not multiply provider calls or ingestion; hidden tabs release demand appropriately; re-entry receives current state; measurements distinguish backend work, browser rendering and native WebView overhead.

## Unsupported assumptions to remove

- Do not estimate refinery-bin progress from `LaunchDrone`. Show witnessed refinement totals and last-observed refinement time; bin fill remains unknown without a documented source.
- Do not describe the latest two prospect records as two verified live prospector objects. Their identity and lifecycle are not established by a rolling buffer.
- Do not adopt guessed synthesis quantities, including the conflicting four/eight-limpet assumptions in the inherited plan and implementation.
- Do not treat a cached market quote multiplied by refined tonnes as earned profit. Future quote requests need consent, freshness and clearly separated valuation labels.
- Do not copy misspelled example identifiers such as `lowtemperaturedianond`, obsolete Node RPC scaffolding or “all results” labels for truncated buffers.

## Sources and attribution

Retained requirements originate in **Varzeki's DAEDALUS mining work**. The original plan names **ObservatoryProspectorBasic** and `ProspectorBasic.cs`, `ProspectorSettings.cs`, `TrackedData.cs`, `RingDetails.cs` and `MaterialData.cs` as research references. It supplies no verified upstream URL, pinned revision or license proof. Preserve that historical attribution without claiming a fresh upstream review. Identify and review the actual source and applicable notices before importing additional logic or assets.

[original-plan]: https://github.com/Jakoczig/eridion/blob/06a69635623c5bfacff51785a33fbb5f09948c9d/MINING_PLAN.md
[legacy-source]: https://github.com/Jakoczig/eridion/tree/06a69635623c5bfacff51785a33fbb5f09948c9d
[native-source]: https://github.com/Jakoczig/eridion/tree/864c724057a7fe28d9d42f5985fa5e1828d3b030
[activity]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/activitystate/model.go
[mining-ui]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-parity-activity.js
[operations-ui]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/src/client/components/go-operations.js
[accounting]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/go/internal/history/mining_ops.go
