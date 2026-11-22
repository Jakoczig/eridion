# G3E — Galaxy geometry, search and native defaults

Base: unchanged G3D `79a034d6a65f9d83272f7655f35ed68f70eeecdf`. Branch: `go/g3e-galaxy-geometry-search`. Final-head CI, draft PR, previews and independently verified SHA-256 evidence are recorded in the PR delivery.

## Implementation

One shared fixed 100:8 main/edge layout replaces all span-dependent grid proportions. Only the container size changes panel rectangles. A padded common coordinate frame uses `max(X/100, Y/8, Z/100)` LY per viewBox unit, preserving light-year scale across all projections. SVG aspect preservation, Canvas positions, clipping, routes, circles and pan share that effective transform. Exact bounded spatial queries retain the selected XYZ volume, not the padded display extent. Density pixel requests account for the selection footprint. Selection clip paths and 3D marker filtering prevent padding from showing out-of-volume observations.

One Search form supports Enter and click, rejects whitespace and synchronously blocks duplicates. It checks local/cached data first. Only an owner with already-enabled EDSM can fall back externally for an unavailable system. The existing server authorization, consent, cache, rate limits and fixed-host read-only client remain authoritative. Viewers cannot initiate fallback. Search results, View System and FSD inspection persist; cooldown notices remain honest.

Fresh native windows use a logical-DIP 16:9 size up to 1440x810, bounded by monitor work area with window-chrome space. Screen-aware startup adjustment runs once. Dashboard creation prefers the launcher's monitor. Valid saved dashboard dimensions preserve their aspect ratio and are clamped only when the usable screen shrinks. Atomic private-profile `dashboard-window.json` stores normal size and maximized state. Resize events update memory; close/quit writes once. Resizing and maximize/restore remain unrestricted. Map modes never resize native windows. The launcher opens normally using restored normal dimensions; dashboard maximization is restored on opening.

## Evidence

Focused failing-then-passing tests cover fixed layout, aspect preservation, local-first authorized fallback, blank input, duplicate prevention, projected 3-4-5 distance, round markers, independent selection clipping, monitor fitting and saved non-16:9 dimensions. Existing G3C/G3D exact XYZ, Canvas, FSD, privacy and owner/viewer regressions remain enabled.

Browser checks cover 1920x1080, 1600x900, 1366x768, 820x1180 portrait and 680x580 compact windows. They compare all panel rectangles through mode switching and multiple XYZ adjustments, verify coordinate changes, and retain headers, 44px touch targets, Enter Search, zoom transitions, stable request counts, no-route FSD and dismissal/restoration. Local spatial responses are explicitly synthetic, not gameplay/publisher acceptance.

[Before/after XYZ screenshots](g3e-map-evidence/). [G3D baseline](https://github.com/Jakoczig/eridion/tree/79a034d6a65f9d83272f7655f35ed68f70eeecdf/docs/handoffs/g3d-map-evidence). Exact-head screenshots for every viewport are included in G3E CI evidence. Local Windows Go tests, native React smoke, focused geometry/search and retained renderer/FSD checks passed. Read the PR for final Windows/Linux CI, Linux race/security results, counts and verified artifacts.

No dependency, listener, upload permission, publisher, private game data or legacy application changes.

## Owner acceptance

1. Start a fresh disposable Windows/Linux profile and verify usable 16:9 sizing. Resize the dashboard to non-16:9, close/reopen/restart, maximize/restore and move between scaled monitors. Confirm saved normal dimensions survive.
2. On owner and paired touch displays, switch modes and adjust each XYZ axis, pan and zoom. Rectangles must stay fixed until window resize. Padding is intentional; circles and distances must remain undistorted.
3. Search cached systems offline. Enter/click must agree. With separately enabled EDSM, search an uncached system as owner and verify caching/cooldown. Viewer or disabled-provider searches must not cause network traffic.
4. Verify short/long and nearing-destination routes, manual inspection/Fit Route, no-route FSD charge/jump/cancel/arrival and dismissal. Check signed offline Bubble and cached paired displays.

Physical gameplay, real touch, live EDSM and actual window-manager maximize/restore gestures remain manual acceptance. Headless smoke does not prove desktop gestures. Catalogue coverage and the existing Chrome inspection navigation-timing issue remain limitations. Do not merge, release or deploy.
