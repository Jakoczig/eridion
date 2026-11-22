# Building ERIDION

ERIDION has two source paths while the native rewrite is in progress. Choose the path deliberately.

| Application | Source | Build entry point |
| --- | --- | --- |
| Native ERIDION preview | [Current native branch][native-branch] | React asset export plus the Go application under `go/`. |
| Inherited legacy application | `main`, and retained legacy folders on native branches | Root npm development, installer, and standalone-service scripts. |

The native branch described here is G3F, reviewed at [864c724][native-source] on 22 November 2026. The [README](README.md) identifies preview availability and known limitations. [CONTRIBUTING.md](CONTRIBUTING.md) explains how to choose a branch and verification scope.

## Native Windows and Linux preview

### Requirements

- Git.
- Node.js and npm for building the interface. Native CI pins **Node.js 24.22.1**.
- **Go 1.26.10**, matching the native module's pinned toolchain.
- A C compiler with CGO enabled for SQLite; Windows requires a compatible GCC/MinGW toolchain on `PATH`.
- Windows 10/11 x64 with Microsoft Edge WebView2 Runtime, or a Linux graphical session with GTK 4, WebKitGTK 6, and session D-Bus.
- Linux stored provider credentials additionally require an unlocked Secret Service and `secret-tool` (`libsecret-tools` on Ubuntu).

For Linux compilation, the Ubuntu 24.04 CI uses `build-essential`, `pkg-config`, `libgtk-4-dev`, and `libwebkitgtk-6.0-dev`. Headless native-window checks additionally use `xvfb` and `dbus-x11`; these are test dependencies.

Wails v3 is resolved through Go modules. A separate Wails CLI, running Node service, Python runtime, and NSIS are not needed for this native executable build.

### Fetch source and prepare embedded assets

```sh
git clone --branch go/g3f-galaxy-viewport-slices --single-branch https://github.com/Jakoczig/eridion.git
cd eridion
npm ci
npm run build:go-assets
cd go
go mod download
go mod verify
```

Rebuild the embedded assets after changing the React interface. These commands require the native branch; `main` does not yet contain its Go tree.

### Windows — PowerShell

From `go/`:

```powershell
$env:CGO_ENABLED = "1"
go build -o bin/eridion.exe ./cmd/daedalus
.\bin\eridion.exe
```

### Linux

From `go/`:

```sh
CGO_ENABLED=1 go build -o bin/eridion ./cmd/daedalus
./bin/eridion
```

The source directory `cmd/daedalus` is a retained compatibility name. These local commands choose an ERIDION executable name. The current CI/preview packager instead uses an intermediate `go/bin/daedalus[.exe]` and packages it as `eridion[.exe]`; consult the workflow before packaging.

Use `--journal-dir "/path/to/journals"` to override discovery and `--data-dir "/path/to/separate-profile"` for an isolated development profile. Native profile/journal configuration does not use the legacy `.env-example`.

If Ubuntu 24.04 reports a WebKit `bwrap` or user-namespace startup error, follow the [Linux startup note][linux-startup] for an executable-specific AppArmor configuration. Keep WebKit's sandbox enabled.

### Verification and previews

The [native desktop workflow][native-workflow] records the actual build, lint, Go, race, asset, browser, and smoke checks. The [current preview PR][preview-pr] identifies workflow runs and available artifacts. Archive milestone labels alone do not establish the compiled revision.

For a local startup check, `eridion --smoke-test` runs the isolated offline service check; `eridion --desktop-smoke-test` opens a native window and checks React/RPC startup. Invoke the executable using its path as above. These checks do not establish real-game or paired-device acceptance.

## Legacy application — retained maintenance path

The inherited application has a React/Next.js client, a Node service, and a Windows Go wrapper using the vendored `go-webview2` dependency. These folders remain useful during migration. The root commands below do not build native ERIDION.

### Run the legacy development service

Use Node.js 24 and npm, matching the declared Node 24 engine range and packaging target. The main-branch workflows pin Node.js 24.22.1 and Go 1.26.10. From a checkout containing the legacy source:

```sh
npm ci
```

Start with automatic journal discovery. The legacy service defaults to the current user's `Saved Games/Frontier Developments/Elite Dangerous` directory.

```sh
npm run dev
```

For an explicit journal-directory override, use the CLI option. For example, in PowerShell:

```powershell
npm run dev -- --save-game-dir "C:\Users\YOUR_USER\Saved Games\Frontier Developments\Elite Dangerous"
```

The option accepts the journal folder or a Saved Games base containing the expected Frontier subdirectory. Replace the example with the actual path.

Copy `.env-example` to `.env` only if legacy environment configuration is needed (`cp` in a POSIX shell or `Copy-Item` in PowerShell). Leave `LOG_DIR` unset for discovery. The inherited environment parser treats only `/`-prefixed paths as absolute; other values are relative to `src/service/`, so use `--save-game-dir` for Windows drive-letter paths.

The legacy browser interface defaults to `http://localhost:3300`. `npm start` is an alias for this development path. The legacy WebView wrapper is Windows-specific; a browser service target does not establish native platform support.

### Windows installer build

The legacy full build needs Windows, Node/npm, Go, NSIS, and the compiler/reference assemblies used by its Windows helper programs. In particular, [build-service.js](scripts/build/build-service.js) locates a .NET Framework C# compiler, `System.Runtime` facade, Windows Runtime assembly, and Windows SDK metadata for audio-meter and thumbnail helpers. Check the supported paths in that script when preparing the build machine.

```sh
npm ci
npm run build
```

Outputs remain legacy-named:

- `build/bin/DAEDALUS Terminal.exe` — Go wrapper.
- `build/bin/DAEDALUS Service.exe` — packaged Node service.
- `dist/DAEDALUS Setup.exe` — NSIS installer.

The wrapper and service must be kept with their required companion files. The installer contains WebView2 bootstrap logic; the application is not dependency-free. Media helper availability depends on the supplied binaries, and binary/version verification is a separate packaging concern.

| Command | Legacy purpose |
| --- | --- |
| `npm run build:client` | Export the React interface. |
| `npm run build:app` | Build the Windows Go wrapper. |
| `npm run build:service` | Package the Node service and Windows helpers. |
| `npm run build:verify` | Run the legacy binary verification script. |
| `npm run build:package` | Assemble the Windows installer. |
| `npm run build:debug` | Build the client, wrapper, and service in debug mode. |
| `npm run build:data` | Regenerate derived reference data, including the dated wiki snapshot. |
| `npm run build:assets` | Rebuild assets and derived data. |
| `npm run build:clean` | Remove generated build output. |

The example environment defaults code signing off. Enable signing only with a certificate you control and the required signing tool configuration. The manually dispatched workflow in `.github/workflows/release.yml` now builds **legacy installer review artifacts only**. It records the checkout commit, package/tool versions, media-helper pins, and checksums, then uploads an expiring artifact. It has read-only repository permissions and creates no release or tag. The installed binaries retain their inherited DAEDALUS names. See the [package and workflow maintenance note](docs/maintenance/package-and-workflows.md).

### Legacy standalone service

```sh
npm ci
npm run build:standalone
```

The current script targets Node 24 on Linux x64, macOS x64, and Windows x64. It writes `dist/daedalus-terminal-service-linux`, `-mac`, and `-windows`, plus the copied `game_voicelines` directory. These are browser-service packages, with no native window, installer, or native ERIDION feature guarantee.

Review the scripts before invoking destructive clean/package steps: the standalone build recreates its `dist/` output directory.

## Reference data and maintenance tools

The [wiki snapshot note](resources/data/fandom/README.md) explains the retained data and attribution. [Maintenance tools](docs/maintenance/development-tools.md) documents the preserved SVG annotator and why historical probes were retired.

[native-branch]: https://github.com/Jakoczig/eridion/tree/go/g3f-galaxy-viewport-slices
[native-source]: https://github.com/Jakoczig/eridion/tree/864c724057a7fe28d9d42f5985fa5e1828d3b030
[native-workflow]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/.github/workflows/go-desktop.yml
[linux-startup]: https://github.com/Jakoczig/eridion/blob/864c724057a7fe28d9d42f5985fa5e1828d3b030/docs/testing/g1a-windows-linux-smoke.md#build-prerequisites
[preview-pr]: https://github.com/Jakoczig/eridion/pull/28
