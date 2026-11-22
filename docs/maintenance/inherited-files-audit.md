# Inherited-file audit and cleanup

**Reviewed:** 22 November 2026. **Age cutoff:** last committed file change before 22 October 2026.

**Default-branch baseline:** [eff5fe7][main-before], containing the inherited application plus the rewritten ERIDION README. **Native comparison:** [G3F `864c724`][native-source]. Existing native draft PRs remain pinned to their reviewed source; this cleanup does not merge that stack into `main`.

## Result and scope

The default branch contained **1,092 tracked files**. **1,091** met the age cutoff; the ERIDION README was the only newer file. The [complete inventory](inherited-file-inventory.json) assigns a date, category, and disposition to every older file.

Age was an audit filter, not a deletion rule. Detailed content review covered all nine older Markdown documents, notices and provenance notes, the loose development utilities, and source backups. Other runtime/source/assets were classified by role and checked against their build or application consumers. This is not a new line-by-line functional or security certification of all retained code; the existing ERIDION correction work remains necessary.

The inherited tree includes work by Varzeki, Iain Collins, and third-party authors. Attribution and active dependency status take precedence over author name or age.

| Disposition of older files | Count |
| --- | ---: |
| Keep in place | 1,073 |
| Rewrite | 3 |
| Replace with the ERIDION contribution guide | 1 |
| Integrate into the native mining roadmap and remove old plans | 3 |
| Move the useful SVG annotation tool | 1 |
| Delete obsolete probes and redundant source backups | 10 |
| **Total older files** | **1,091** |

The README receives navigation/build-documentation updates as part of integration, although it is newer than the cutoff. New audit and roadmap documents are not counted as pre-existing files.

## Planning and project documentation

| Original file | Last change | Decision and retained value |
| --- | --- | --- |
| [MINING_PLAN.md][old-mining] | 5 May 2026 | Integrate into [the native mining roadmap](../roadmap/mining.md), then remove the old root plan. Keep useful ring/hotspot, target, limpet, session, history, and usability requirements. Replace obsolete Node scaffolding, false “stub page” assumptions, and unsupported inference rules. |
| [MINING_LAYOUT_OVERLAP_PLAN.md][old-overlap] | 5 May 2026 | Integrate its readable grouping and optional circle/cutout ideas into the same roadmap. Remove superseded dimensions, pending-work claims for already implemented legacy JSX, and placeholder configuration panels. |
| [NOTCH_PLAN.md][old-notch] | 5 May 2026 | Remove the 935-line experiment transcript after preserving its useful layout principles. Repeated abandoned “final” solutions and hand-tuned coordinates do not define the native design. |
| `BUILD.md` | 8 April 2026 | [Rewrite](../../BUILD.md) with distinct native and legacy paths, actual dependencies and commands, current source entry points, and correct standalone targets/output names. Remove unsupported size, dependency-free, and build-time promises. |
| `CONTRIB.md` | 12 April 2026 | Replace with [CONTRIBUTING.md](../../CONTRIBUTING.md). Preserve navigation/route concepts while distinguishing native Go behavior from legacy Node handlers. Correct the primary-navigation path and document current project requirements. |
| `resources/data/fandom/README.md` | 8 April 2026 | [Rewrite as provenance and maintenance documentation](../../resources/data/fandom/README.md). Keep content attribution and the dated snapshot; retire Python 2 bootstrap commands as present-day setup advice. |

### Why the mining plans were not kept verbatim

The old plan says that mining pages and a service handler do not exist. They already exist in the inherited application. Native G3F has a different shared-Go implementation, with bounded recent observations, ring geometry, and persistent refinement/sale records. Neither the old plan's checkmarks nor the old dashboard's existence proves complete native parity.

The layout plans describe missing panels that already have JSX and CSS implementations. Their proposed column sizes differ from the actual legacy dashboard. Native mining uses responsive cards and historical-operation views instead of that fixed, scaled canvas.

The consolidated roadmap preserves unresolved gameplay value: hotspot attribution, launcher-managed targets and warnings, truthful session/history metrics, readable responsive layouts, equal trusted-display access, and measured shared-state performance. It rejects guessed refinery progress, inferred live prospector identities, guessed synthesis yields, quote-derived “profit,” and misspelled sample identifiers.

Historical attribution to Varzeki and the plan's named ObservatoryProspectorBasic research reference is retained. No unverified upstream URL, revision, or license has been invented.

## Development utilities and backups

| Original file or group | Decision | Evidence and retained knowledge |
| --- | --- | --- |
| `_dev-scripts-backup/annotate-svgs.mjs` | Move to [scripts/annotate-device-svgs.mjs](../../scripts/annotate-device-svgs.mjs). | Unique logic produces device-label annotations consumed by `src/client/components/keybinds-visual.js`. Executable logic and relative-root resolution are preserved; usage comments identify the new location. It remains a manual, unvalidated utility. |
| `_tmp/audit-svgs.mjs` | Delete. | Its annotation regex misses ordinary `data-label=` attributes, so its reported count is unreliable. Keep annotation/transform checks as requirements for future maintained validation. |
| `_tmp/check-ch.mjs`, `check-svg.mjs`, `debug-ch.mjs` | Delete. | One-off SVG probes, including a developer-specific Windows path and partial geometry inspection. Source diagrams and the substantive annotation tool remain. |
| `_tmp/test-cs.js` | Delete. | Machine-local Steam/ControlSchemes probe; maintained discovery is already in `src/service/lib/event-handlers/keybinds.js`. |
| `extract-armour.js` | Delete. | Requires a private external drive/cache and depends on a specific serialized field order. It is not a usable catalogue import pipeline. Canonical IDs, variants, and provenance belong in reviewed catalogue maintenance. |
| `_dev-scripts-backup/test-status-watcher.js` | Delete. | Concurrent watchers share a content cursor, allowing one to suppress another's samples. Its mtime delta is not native display latency. Preserve the measurement objective, with independent observer state and separate ingestion/render timings. |
| `src/app/third_party/go-webview2/common.go.bak2021` | Delete backup only. | Not a Go compilation input; the current `common.go`, notices, and historical revision remain. |
| `src/app/third_party/go-webview2/webview.go.bak2021` | Delete backup only. | Current `webview.go` remains. |
| `src/app/third_party/go-webview2/pkg/edge/chromium.go.bak` | Delete backup only. | Current `chromium.go` remains. |

No build/runtime references to the retired plans, probes, or backup filenames were found. The retained annotator was not executed and no controller diagram was regenerated. Its dependency and validation limitations are documented in [maintenance tools](development-tools.md).

## Files deliberately preserved

| Category | Older files | Reason |
| --- | ---: | --- |
| Application and build source | 224 | The inherited app and its build paths remain present. Native migration and known behavioral fixes require scoped implementation changes, not deletion by age. |
| Data and runtime assets | 611 | Includes voice lines, reference datasets, wiki source/exporter, and interface assets with current consumers or retained provenance. |
| Build and packaging resources | 165 | Icons, installer resources, SDK/binary inputs, and related build material remain connected to retained packaging paths. |
| Vendored dependency source | 57 | Retain component implementations until an explicit dependency retirement establishes replacement and compatibility. |
| Component READMEs | 3 | Keep `go-webview2/README.md`, `go-webview2/webviewloader/README.md`, and `src/service/lib/say/README.md` as upstream/component documentation. They are not ERIDION application setup guides. |
| License notices | 4 | Keep the root GPL text, WebView2 MIT notice, Microsoft loader SDK BSD notice, and say.js MIT notice. Embedded copyright/license headers, including WikiTeam's exporter notice, also remain. |
| Dataset/binary provenance | 3 | Keep the wiki dump's `config.txt` and titles inventory, plus `resources/ffmpeg-version.txt`. These describe historical artifacts, not automatically verified current versions. |
| Project configuration | 7 | Keep functional dependency manifests, lockfile, ignore rules, npm configuration, and workflows. Rewrite only the misleading example environment as described below. |

### Active uses that prevent destructive cleanup

- `scripts/build/build-data.js` reads the dated wiki XML to enrich commodity reference data. Full codex-page/index export is disabled; the export configuration and titles list preserve the snapshot's origin.
- `scripts/build/build-standalone.js` copies `game_voicelines/`; `src/service/lib/covas-player.js` resolves those recordings at runtime.
- `src/app/go.mod` selects the local `go-webview2` source through a `replace` directive; the loader embeds the architecture-specific SDK DLLs.
- The legacy mining JSX/CSS still serves the legacy application. Removing obsolete planning files does not retire that interface.
- The controller visualizer consumes annotated SVGs; keeping those assets and the substantive annotator preserves maintenance knowledge.
- `resources/ffmpeg-version.txt` is historical metadata. The release workflow downloads a moving archive and the build copies an available local binary, so the text marker is not proof of the packaged FFmpeg version.

The vendored `say` directory was not proven necessary by a non-vendor import search, but the package remains declared and component retirement was not established. Keep its source and license pending a dedicated dependency decision; do not claim that age or a text search alone proves safe removal.

### Example environment correction

The old `.env-example` enabled signing and named Iain Collins's certificate. It is rewritten as a **legacy-only example**, with signing off and an explicit user-owned certificate placeholder. Its journal override is left unset for discovery; the build guide documents the working CLI override for Windows drive-letter paths, which the inherited `LOG_DIR` environment parser mishandles. Native journal/profile configuration is described through launcher settings and CLI options instead.

No live environment, certificate, account, provider configuration, release workflow, or deployed publisher was changed.

## Verification and maintenance

This cleanup requires content/dependency checks, not a gameplay test cycle:

- Confirm each removed file is older than the cutoff and exists in the reviewed baseline.
- Preserve useful requirements, original attribution, vendor notices, and active build inputs.
- Check internal links, anchors, referenced source paths, build commands, and current/legacy/planned labels.
- Check the moved annotator's syntax and compare executable content with its source; do not run it against assets.
- Verify the exhaustive inventory covers each eligible tracked file exactly once.
- Verify the published commit changes only the reviewed paths and retains the expected parent.

When native development is integrated later, carry these documentation/cleanup changes forward deliberately. Historical draft branches may still contain the retired files; their pinned evidence is not rewritten by this cleanup. New plans should describe current native scope and unresolved requirements rather than copying the old implementation transcript.

[main-before]: https://github.com/Jakoczig/eridion/tree/eff5fe719b0a82611255cdf846d166a24a7cc228
[native-source]: https://github.com/Jakoczig/eridion/tree/864c724057a7fe28d9d42f5985fa5e1828d3b030
[old-mining]: https://github.com/Jakoczig/eridion/blob/06a69635623c5bfacff51785a33fbb5f09948c9d/MINING_PLAN.md
[old-overlap]: https://github.com/Jakoczig/eridion/blob/06a69635623c5bfacff51785a33fbb5f09948c9d/MINING_LAYOUT_OVERLAP_PLAN.md
[old-notch]: https://github.com/Jakoczig/eridion/blob/06a69635623c5bfacff51785a33fbb5f09948c9d/NOTCH_PLAN.md
