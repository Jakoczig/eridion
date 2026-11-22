# Contributing to ERIDION

ERIDION continues DAEDALUS and ICARUS while developing a native Go/Wails application. Preserve the project and component licenses, attribution, and data provenance when changing inherited work.

## Choose the correct implementation

Read the [project status](README.md#project-status) and [build guide](BUILD.md) first. Native work is currently on [the G3F branch][native-branch], inspected at `864c724057a7fe28d9d42f5985fa5e1828d3b030`; `main` still contains the inherited application plus current project documentation.

For native feature work, branch from the agreed native head and keep the change scoped to the relevant development stack. Existing draft PRs, releases, deployment, and publisher operations remain separate maintainer decisions. A passing build is not permission to publish a release.

For documentation changes, identify whether a statement describes the legacy application, an implemented native feature, or planned work. Use pinned source links for historical evidence. Update [the mining roadmap](docs/roadmap/mining.md) when its native coverage changes rather than restoring retired implementation transcripts.

## Native feature work

Use the existing shared Go ingestion and history architecture. Do not add a separate journal watcher or browser-side accounting engine for a new page.

| Concern | Native source area |
| --- | --- |
| Journal/state/history | `go/internal/journal/`, `gamestate/`, domain projections, and `history/`. |
| External services | `go/internal/providers/`, with shared consent, identity, caching, and request controls. |
| Native API | `go/internal/webapp/`; validate requests and account/session attribution on the host. |
| Primary navigation | `src/client/components/primary-navigation.js`. |
| Side navigation | `src/client/lib/navigation-items.js`. |
| Native page routing | `src/client/components/go-terminal.js` and its delegated native workspaces. |
| Presentation | Native `go-*` React components and styles under `src/client/`. |

These paths refer to the native branch. Adding a Next.js route or a Node `getHandlers()` entry alone does not implement a native feature.

When adding a workspace:

1. Extend the appropriate native projection or bounded history query; preserve source and unknown/partial states.
2. Expose the operation through the host's validated API and reuse existing update subscriptions.
3. Add navigation, routing, and presentation to the shared interface.
4. Verify selected-commander behavior, paired-display behavior, reconnection, and the relevant empty/offline states.

### Product requirements for new work

These are the agreed correction targets; existing code does not yet satisfy all of them.

- Valid live Frontier facts take priority, followed by journal/Status evidence, then Inara and other providers equally. Compare matching identities and observation freshness, not only fetch time.
- Main and trusted paired displays receive equivalent gameplay information and operational permissions. Configuration belongs in the launcher.
- Single-commander mode should automatically select its configured default. Account changes must not mix old state or outstanding replies.
- Preserve separate ship, suit, SRV, and fighter identities. Unknown catalogue fields remain unknown.
- Keep ingestion and state shared. Normal display work follows the visible tab; any low-latency prewarming remains bounded and shared.
- Route and game-control operations are explicit user actions through the configured host service. Their planned existence does not imply they are implemented.

## Legacy fixes and reference code

The inherited tab pattern remains useful for deliberately scoped legacy fixes:

- Primary tabs are defined in `src/client/components/primary-navigation.js`, with side navigation in `src/client/lib/navigation-items.js`.
- Next.js pages under `src/client/pages/` provide routes and index redirects.
- Legacy Node handler classes expose `getHandlers()` and are registered by `src/service/lib/event-handlers.js`.
- Native behavior must be checked separately when a shared React component changes.

The former `CONTRIB.md` pointed at the wrong primary-navigation file and described the Node route as a universal architecture. This guide replaces it; the original remains in Git history.

## Verification

Choose checks that address the actual change and report what they establish.

- Documentation: validate links, paths, commands, branch scope, and factual status. Do not run a full gameplay cycle solely for wording changes.
- Native logic: add focused behavioral regressions for the failure or contract being changed, including relevant attribution, replay, ordering, and boundary cases.
- UI behavior: inspect affected layouts and interaction at desktop and resized/touch sizes; preserve keyboard focus and unknown states.
- Integration changes: use synthetic fixtures and documented contracts; separate fixture/schema coverage from live-service acceptance.
- Performance changes: measure the affected workload, including additional displays when relevant. Report what was included and excluded.

The [build guide](BUILD.md) and [native workflow][native-workflow] list commands. Run broader CI gates when required by the affected development branch. Preserve useful evidence without uploading real journals, credentials, or private account responses.

## Upstream references and data

Record the source, revision, license, and transformation when importing algorithms or datasets. A familiar plugin name or an old “implemented” checkmark is not proof of native parity.

Keep vendor notices beside retained code. Do not rewrite upstream component READMEs as if they were ERIDION's own product instructions. The [inherited-file audit](docs/maintenance/inherited-files-audit.md) records which material was retained, remade, integrated, or removed.

[native-branch]: https://github.com/Jakoczig/eridion/tree/go/g3f-galaxy-viewport-slices
[native-workflow]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/.github/workflows/go-desktop.yml
