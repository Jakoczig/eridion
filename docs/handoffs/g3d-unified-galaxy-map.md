# G3D — Unified Galaxy Map

Stacked on G3C `657a79ae22f54c3f3a0718208f31e20c82a30da8`; branch `go/g3d-unified-galaxy-map`. Final immutable head, CI run links and independently verified preview/archive hashes are recorded in the G3D draft PR delivery evidence. Earlier PRs and the publisher remain untouched.

## Delivered

One persistent map interface, compact left controls, right XZ/XY/YZ group sharing light-year scale, visible global navigation, accessible focus and measured >=44px primary controls. Search, explicit owner EDSM lookup, View System and always-on route/current/destination overlays remain. Duplicate Local Volume page, primary layer/route switches, placeholder Navigate and permanent diagnostics are removed. Advanced Settings retain signed installation, Bubble, rollback, caching and consent.

Galaxy Map, Local Atlas and Show Map During FSD are independent persisted presentation settings; existing profiles default to both renderers available and FSD map off. Renderer settings do not grant download consent or select devices. Server authorization remains unchanged. Both renderers disabled is a supported navigation-overlay state.

Automatic framing uses current and remaining actual route coordinates, excludes completed legs, and refuses to infer missing coordinates. Bounded Go context now reserves upcoming legs and destination before historical points. Local entry is <=800 LY X/Z and <=400 LY Y; exit is >1200 LY X/Z or >600 LY Y, also constrained by <=256 intersecting 800-LY root indexes and existing server budgets. A complete verified spatial query is required to display Local; missing/over-budget detail falls back to the enabled density renderer. Without either renderer, only navigation overlays remain. No fake player/Sol location is created.

Pan and axis adjustments persist across journal updates and automatic FSD overlays. Fit Route restores following. Mode switching resets sizing; zoom retains centre and orientation. XYZ adjusters change one axis, clamp 20–1200 LY X/Z and 20–600 LY Y, and scale steps 10/50/100 LY. Queries debounce 180ms, retry only while pending or on cache revision, and stable frames do not repeatedly query. Existing backend byte/block/point, disk, signed-trust and read-only-viewer protections are retained.

FSD correction removes the route requirement, recognizes authoritative Status FSDJump before clearing charge, retains one episode through journal StartJump, and uses legacy FSDCharging only when Flags2 is absent. An explicit modern hyperdrive false is not replaced by generic charge. Existing fresh/live identity gates, selected-display policy, charge/jump settings, restoration and manual episode blocking remain. Show Map During FSD controls map visibility inside that same controller.

## Verification evidence

Focused controller tests initially failed for no-route/map policy and Go legacy-charge/episode continuity, then passed after correction. Remaining-leg context regression advances a 220-hop route to hop 210 and verifies every upcoming coordinate within the 192-point cap. Tests cover persisted independent settings, unselected viewers, unknown positions, hysteresis, asymmetric axis bounds, exact XYZ pixels/clipping, route overlays and queued resize after unmount.

Local Windows: 170 JavaScript tests, 28 build/asset tests, full Go tests, Go vet, security/build lint, React static build/export, executable offline smoke and native React bootstrap/shutdown passed. G2G, G2O, G2P, G2Q, G3C and G3D browser checks passed in disposable synthetic profiles. Final Windows/Linux native, race/security and artifact results must be read from the exact-head PR evidence, not inferred from these local results.

[Before/after screenshots and measured layout](g3d-map-evidence). Prior G3C Galaxy screenshots and new G3D density/Local desktop/touch screenshots contain synthetic navigation. G3D Local response is explicitly injected synthetic UI evidence; it does not claim new live publisher validation or real-game acceptance. The unchanged signed spatial backend retains its existing verification regressions. A short 1.2s synthetic browser sample measured 3.41 MB JS heap and 1.54% main-thread busy time, with no stable-view spatial requests. This is not total native RSS/CPU or a statistical speedup claim.

## Manual acceptance

1. Owner: verify independent renderers and both-off behavior; enabling Local must not permit downloads. Use Advanced only with explicit consent, install signed data/Bubble, then restart offline.
2. Touch/laptop: reach primary and side navigation in one tap; inspect all three planes, search and View System; zoom through both mode thresholds without centre jumps.
3. Gameplay: short route -> Local when verified coverage exists; long route -> Galaxy; completed legs disappear from framing and approaching destination becomes Local. Missing positions remain unknown.
4. Pan/change each XYZ axis; jump/update route and confirm inspection remains; Fit Route resumes automatic following. Test asymmetric extents and routes crossing the volume.
5. Selected owner and paired laptop: no-route charge/jump, cancellation, arrival restore, manual dismissal/navigation through one episode; new episode may open again. Disable map during FSD while retaining general FSD info. Check unselected viewers remain unaffected.

Physical game, real touchscreen and paired-device acceptance remain owner tasks. Tiny windows may need vertical scrolling in the left control column; headers and right projections remain accessible. Public catalogue coverage remains incomplete. Legacy Node/launcher, authentication, private profiles, SQLite and upload permissions are unchanged. No release, merge, deployment or production uploads.
