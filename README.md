# ERIDION

**A native companion and second-screen interface for Elite Dangerous.**

ERIDION (ERID) brings commander information, ship and fleet data, navigation, exploration, and field tools into one interface for desktop windows and paired displays. The native application is being developed for Windows and Linux, with a shared Go service, a React interface, and local SQLite history.

ERIDION continues the work of [DAEDALUS Terminal](https://github.com/Varzeki/daedalus), itself a fork of [ICARUS Terminal](https://github.com/iaincollins/icarus) by Iain Collins. This repository is the home of ERIDION's development.

[Native development branch][native-branch] · [Preview downloads and status][preview-pr] · [Build from source](#build-the-native-preview) · [Development priorities](#development-priorities) · [License](LICENSE)

## Project status

**ERIDION is an experimental developer preview.** The native work is progressing through staged development branches and draft pull requests.

| Track | Where to find it | Status |
| --- | --- | --- |
| Native ERIDION | [go/g3f-galaxy-viewport-slices][native-branch] and [PR #28][preview-pr] | Current preview described by this README, as of 22 November 2026. |
| Legacy application | [main](https://github.com/Jakoczig/eridion/tree/main) | Contains the inherited application; the native development stack has not been merged here. |
| Planned corrections | [Development plans and Claude handoff](docs/plans/README.md) | Recorded audit, prioritized implementation tasks and performance acceptance; application changes remain pending. |

The native preview has Windows and Linux build and automated verification workflows. Real-game behavior, physical paired devices, and live provider coverage still need acceptance for each relevant change. Preview artifacts are unsigned experimental executables, not a stable ERIDION installer.

## What the native preview includes

- **Journal and game-state ingestion:** incremental journal reading, supported sidecar data, commander attribution, and persistent local history.
- **Commander, ship, and fleet workspaces:** observed ship status, owned ships, saved loadouts, stored modules, commander information, and community-goal views.
- **Navigation and maps:** game-route tracking, jump and FSD context, system views, galaxy density and local XYZ views, and optional cached atlas data.
- **Exploration assistance:** system and body records, scan information, organic-scan tracking, and a partial native BioScan implementation.
- **Field tools:** imported route itineraries and progress, journal-derived BGS activity, trade and mining records, materials and engineering workspaces, and supported ship power scenarios.
- **Multiple displays:** native desktop windows plus HTTPS browser pairing for a second computer, tablet, or phone.
- **Optional integrations:** provider controls, separate read and publication consent, cached reference data, and account-linked services where supported.

Coverage varies by feature. Ship and module identification, fleet completeness, data precedence, biological prediction rules, display permissions, and parts of the interface are under correction. Unknown or unsupported values should remain visible as unknown. Existing native helpers implement selected workflows; ERIDION does not run arbitrary EDMC Python plugins.

### Try a preview

1. Open [the current preview PR][preview-pr] for its source commit, CI results, artifact links, and packaging notes.
2. Extract the appropriate Windows x64 or Linux x64 archive into its own directory. Check the supplied `SHA256SUMS` and the PR's verification notes.
3. Run `eridion.exe` on Windows or `eridion` from a Linux graphical session. Configure the journal location and the services you want to use.

GitHub Actions downloads may require sign-in and are retained for a limited time. Some archive milestone labels lag the source revision. Use the artifact linked from the matching workflow run and its recorded source/checkout information; archive labels alone do not identify a build. The [preview prerequisites][preview-setup] and [pairing guide][pairing-guide] provide setup details. Older milestone documents describe the behavior at that milestone.

| Platform | Runtime requirements |
| --- | --- |
| Windows x64 | Windows 10/11 and Microsoft Edge WebView2 Runtime. |
| Linux x64 | GTK 4, WebKitGTK 6, and a graphical session with session D-Bus. Stored provider credentials also require an unlocked Secret Service and `secret-tool`. |

The native application keeps its own profile and history. Preserve an existing installation and profile when trying a new experimental build.

## Build the native preview

Use the [ERIDION build guide](BUILD.md) for native Windows/Linux prerequisites, source commands, runtime setup, and troubleshooting. It explicitly selects the native development branch and keeps legacy development/packaging commands in their own section.

The native application embeds its React interface and runs a shared Go service. Node.js is needed to build the interface, while the installed application uses the operating system's WebView. The source entry point remains named `go/cmd/daedalus` for compatibility.

## Data and external services

The native preview stores journal history, derived game state, and configuration locally. It is not a memory-only application. Provider credentials use operating-system protection: Windows DPAPI or a Linux Secret Service-backed encrypted vault.

External access is optional. Read access and publication are separately controlled; enabling reads does not enable uploads. Depending on the selected service and publication category, requests can include game information and commander identifiers. Each provider also receives ordinary connection metadata such as the request's IP address.

| Service | Native integration scope |
| --- | --- |
| Frontier | OAuth and supported Companion API reads for the connected account. Endpoint availability and freshness vary; login success alone does not validate every field. |
| EDSM | Reference lookups and supported account/journal workflows, subject to their respective controls. |
| EDDN and EDAstro | Supported publication adapters with explicit consent and category controls. Their presence does not imply complete event coverage or production acceptance. |
| Inara | An adapter exists, but application approval and API-contract corrections remain outstanding. Full working synchronization is not claimed. |
| Spansh, Canonn, EliteBGS, and Raven | Selected reference-data capabilities. Spansh currently uses website workflows and local CSV itineraries; native in-app route generation is planned. |

See the [provider capability definitions][provider-capabilities], [Frontier connection contract][frontier-contract], and [persistent history and integration handoff][history-handoff] for implementation details.

Paired displays use HTTPS and explicit pairing. The current preview still restricts some paired-display information and actions. Equal gameplay access on all trusted displays, with configuration confined to the launcher, is part of the correction plan below.

## Development priorities

These requirements guide the next corrections and feature work. They are deliberately separate from the implemented preview features above.

| Area | Intended behavior |
| --- | --- |
| Data authority | **Valid live Frontier data first; journal and Status evidence second; Inara and every other provider at the same third priority.** Preserve identity, source, observation time, and freshness so stale or unrelated data cannot replace valid current facts. |
| Commander selection | Automatically select the configured default commander when multi-commander mode is off. Keep multi-commander selection explicit and account data isolated. |
| Trusted displays and settings | Give every paired display the same gameplay information and operational access as the main display. Keep all configuration in the launcher. A legitimate commander change should update trusted displays consistently. |
| Exosuit and fleet accuracy | Add a separate **Commander → Exosuit** tab for the known suit, loadout, and attributable live on-foot data. Keep suits, ships, SRVs, and fighters distinct; correct missing fleet entries, ship names, and outfitted-module identification. |
| Spansh route planning | Plan routes inside ERIDION, fill the current location with one action, and import the result directly. An active Spansh itinerary takes precedence over the game's route in the Route tab. |
| Key controls | Add launcher-managed bindings for explicit game actions, starting with ship controls such as landing gear and later a next-waypoint action that opens the galaxy map, searches, selects, and plots. These game-input actions are not implemented in the current native preview. |
| Native helpers | Add controllable native integrations for relevant EDMC workflows, including the **LandingPad** docking helper. Review current upstream algorithms and licenses, then verify native behavior against them. Full plugin parity is not assumed. |
| Performance | Measure startup, replay, CPU, memory, storage, and display latency. Keep ingestion and state shared; make normal display updates follow the visible tab, with an optional low-latency mode that prepares all views. |
| API and algorithm maintenance | Audit every service against its current documented contract, including Inara. Check upstream data and algorithm revisions during build maintenance, pin reviewed updates, and verify their behavior before adoption. |

## Development and documentation

For the next native coding batch, start with the [Claude coding handoff](docs/handoffs/2026-11-22-claude-coding-handoff.md) and [development planning index](docs/plans/README.md), including the [detailed performance plan](docs/superpowers/plans/2026-11-22-native-performance.md). The handoff pins the native source separately from main and preserves the existing draft PR stack.

Build and contribution guidance is in [BUILD.md](BUILD.md) and [CONTRIBUTING.md](CONTRIBUTING.md). The [native mining roadmap](docs/roadmap/mining.md) consolidates useful inherited mining and layout requirements. The [inherited-file audit](docs/maintenance/inherited-files-audit.md) records the cleanup decisions and retained provenance.

| Location on the native branch | Purpose |
| --- | --- |
| [go/][go-source] | Native desktop host, shared service, journal processing, history, providers, and domain logic. |
| [src/client/][client-source] | Shared React interface and native-service presentation. |
| [src/service/][legacy-source] | Legacy Node service retained during the transition. |
| [docs/handoffs/][handoffs] and [docs/testing/][testing] | Milestone scope, limitations, verification notes, and acceptance checklists. |
| [docs/licenses/g2k/][field-credits] | Pinned native field-tool sources and license notices. |

For changes to the native branch, relevant checks include the following, with the build prerequisites installed:

```sh
npm test
npm run lint:security
npm run lint:go-assets
npm run test:go-assets
npm run build:go-assets
cd go
go test ./...
go vet ./...
```

The CI workflow contains additional platform, race, smoke, and browser checks. Choose verification appropriate to the change; a documentation edit does not require a complete gameplay test cycle.

For bug reports, include the build commit, operating system, affected view, expected behavior, and reproduction steps. Remove commander identifiers and other private information from shared journals or diagnostics, and never include credentials.

## Credits

ERIDION builds on years of work by the Elite Dangerous tools community:

- **[DAEDALUS Terminal](https://github.com/Varzeki/daedalus)** by Varzeki and **[ICARUS Terminal](https://github.com/iaincollins/icarus)** by Iain Collins — the project's foundation and inherited interface. The ICARUS name was suggested by [SpaceNinjaBear](https://www.reddit.com/user/SpaceNinjaBear).
- **[EDCD / FDevIDs](https://github.com/EDCD/FDevIDs)** and **[EDSM](https://www.edsm.net)** — game identifiers, community data, and stellar cartography.
- **[SrvSurvey](https://github.com/njthomson/SrvSurvey)**, **[Elite Observatory BioInsights](https://edjp.colacube.net/observatory)**, and **[Elite Dangerous Exploration Buddy](https://www.panostrede.de/EDEB/)** — inherited exploration and biological-analysis references.
- **[EDMC-BioScan](https://github.com/Silarn/EDMC-BioScan)** and **[EDMC-ExploData](https://github.com/Silarn/EDMC-ExploData)** — licensed reference rules and biological data used by the native field tools.
- **[BGS-Tally](https://github.com/aussig/BGS-Tally)** and **[Neutron Dancer](https://github.com/dwomble/EDMC-NeutronDancer)** — field-activity and itinerary workflow references.
- **[Elite Dangerous Odyssey Materials Helper](https://github.com/jixxed/ed-odyssey-materials-helper)** — inherited engineer imagery and engineering-workflow references.
- **[LandingPad](https://github.com/bgol/LandingPad)** — inherited docking-layout references and the selected reference for the planned native docking helper.
- **[Canonn Research Group](https://canonn.science/)**, **[Spansh](https://spansh.co.uk/)**, **[Inara](https://inara.cz/)**, and the historical **EDDB** project — community research, services, and reference material.
- **[James Panter](https://codepen.io/jpanter/pen/PWWQXK)** — the inherited loading animation; **[edassets.org](https://edassets.org)** — icon references; **[Jura](https://fonts.google.com/specimen/Jura)** — the bundled font under the Open Font License; **[Serge Zaitsev](https://github.com/zserge)** — foundational WebView work.

See the [native field-tool credits and retained notices][field-credits] for exact pinned sources, licenses, and integration limits. A reference or credit does not imply full feature parity or endorsement.

## License and game attribution

ERIDION is free, open-source software under **GPL-3.0-or-later**; see [LICENSE](LICENSE). The original ICARUS Terminal was released under the ISC License, and third-party components retain their respective licenses and notices.

Elite Dangerous is copyright Frontier Developments plc. ERIDION is an independent community project and is not affiliated with or endorsed by Frontier Developments. The software is provided without warranty as described in its license.

[native-branch]: https://github.com/Jakoczig/eridion/tree/go/g3f-galaxy-viewport-slices
[preview-pr]: https://github.com/Jakoczig/eridion/pull/28
[preview-setup]: https://github.com/Jakoczig/eridion/blob/go/g3f-galaxy-viewport-slices/docs/testing/g3e-map-refinement-preview.md
[pairing-guide]: https://github.com/Jakoczig/eridion/blob/go/g3f-galaxy-viewport-slices/docs/testing/g1b-secure-remote-displays.md
[provider-capabilities]: https://github.com/Jakoczig/eridion/blob/go/g3f-galaxy-viewport-slices/go/internal/providers/types.go
[frontier-contract]: https://github.com/Jakoczig/eridion/blob/go/g3f-galaxy-viewport-slices/docs/protocols/eridion-frontier-website-return.md
[history-handoff]: https://github.com/Jakoczig/eridion/blob/go/g3f-galaxy-viewport-slices/docs/handoffs/g2j-persistent-history-integrations.md
[go-source]: https://github.com/Jakoczig/eridion/tree/go/g3f-galaxy-viewport-slices/go
[client-source]: https://github.com/Jakoczig/eridion/tree/go/g3f-galaxy-viewport-slices/src/client
[legacy-source]: https://github.com/Jakoczig/eridion/tree/go/g3f-galaxy-viewport-slices/src/service
[handoffs]: https://github.com/Jakoczig/eridion/tree/go/g3f-galaxy-viewport-slices/docs/handoffs
[testing]: https://github.com/Jakoczig/eridion/tree/go/g3f-galaxy-viewport-slices/docs/testing
[field-credits]: https://github.com/Jakoczig/eridion/tree/go/g3f-galaxy-viewport-slices/docs/licenses/g2k
