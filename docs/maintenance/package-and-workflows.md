# Package and workflow maintenance

Reviewed 22 November 2026, following the inherited-file cleanup at `40d07eb38a1614b6dd63cd1ab3e55556a7f6da15`.

## Why these files needed correction

The earlier cleanup retained the package manifests and workflows because the inherited application still uses them. Retention did not establish that their contents were current. This follow-up corrects their identity, internal consistency, and build/publication behavior.

| File | Finding | Correction |
| --- | --- | --- |
| `package.json` | DAEDALUS name/description, no repository metadata, Node 20 minimum below a locked direct dependency's requirement. | ERIDION identity, repository/homepage links, private application package, and a Node 24 engine range aligned with CI. |
| `package-lock.json` | Both root version fields still said 2.0.0 while the manifest said 2.2.0. | Synchronize root name, version, and engine metadata. Preserve all 946 non-root package entries and their dependency resolutions/integrities. |
| `ci.yml` | Generic name, cached `node_modules` could skip installation, insufficient cache identity, no explicit permissions/timeouts, and old Go series. | Explicit legacy scope, read-only token, checkout without persisted credentials, npm download caching with unconditional `npm ci`, lock/toolchain-aware build cache, bounded runs, and reviewed toolchain pins. Add a root-metadata consistency check. |
| `release.yml` | Published a stable DAEDALUS v2.2.0 release from the inherited application and downloaded moving media binaries. | Retain manual legacy installer packaging as artifact-only output, with read-only repository permissions, pinned media URLs/digests, source metadata, and checksums. Remove release/tag publication. |

## Package identity and compatibility

The repository is an application, so `private: true` prevents accidental npm publication; it does not change the repository's public visibility or software license. Repository and homepage URLs point to `Jakoczig/eridion`. No issue-tracker URL is advertised because GitHub issues were disabled when checked.

The existing **2.2.0** version is preserved. It is consumed by legacy Windows version resources, interface labels, preferences, and provider metadata. Renaming the npm package does not rename legacy binaries, profile directories, fonts, API application identities, or the inherited interface. A native release version needs a separate coordinated decision.

The declared Node range is **>=24 <25**. Both main-branch workflows pin **Node.js 24.22.1** and **Go 1.26.10**. Node 20 was not a valid advertised minimum for the locked `@yao-pkg/pkg` 6.15.0, whose own engine requires Node 22 or later.

## What main's CI establishes

Main still contains the inherited application. Its three jobs cover advisory legacy JavaScript lint, the shared client build with package/lock metadata validation, and the legacy Windows wrapper compilation/startup check.

Inherited lint violations remain advisory and visible. Main has no native `go/` tree or native/security test scripts, so those jobs are not copied into this workflow. Native Go/Wails, journal, paired-display, and provider checks remain on the native development stack.

Every Node job runs `npm ci`; the cache stores npm downloads instead of treating a restored `node_modules` directory as a completed install. Superseded CI runs on the same ref are cancelled. The manual installer workflow is separate and is not triggered by pushes.

## Manual installer artifacts

The retained `release.yml` filename now represents **ERIDION legacy installer artifacts**. Manual execution builds the inherited Windows application and uploads an expiring artifact containing its installer, the project license, a scope README, `build-metadata.json`, and `SHA256SUMS`.

The metadata records the actual checkout commit, ref, workflow run/attempt, package version, Node/npm/Go versions, lockfile hash, and media-helper inputs. Artifact names include the source SHA and run identity. The workflow's repository token has `contents: read`; it contains no release action, tag creation, or push step.

Media inputs were pinned from the publishers' release metadata:

- [yt-dlp 2026.10.26](https://github.com/yt-dlp/yt-dlp/releases/tag/2026.10.26), Windows executable SHA-256 `f5389d3698ecde4ef06f4e6b1ce0294fe9ac1396228c8990b8d3416d6d177a7d`.
- [Gyan FFmpeg 9.1](https://github.com/GyanD/codexffmpeg/releases/tag/9.1), essentials ZIP SHA-256 `b9bbbb3acece000cbeec994f856649354e52c773807841e78908374b24d658fd`.

The workflow verifies these digests before extraction or packaging. The required yt-dlp executable is retained because the inherited NSIS installer expects it. These are publisher/API digests; this configuration review did not independently build or download the Windows media binaries.

The existing packaged-service smoke script skips its WebView check under CI and checks service startup. It does not prove installer, real-game, paired-device, or full native acceptance. Hosted runner images, action major tags, and Chocolatey packaging tools are still maintained inputs; the workflow is not claimed to be a bit-for-bit reproducible or production-accepted release process.

This change applies to **main's workflow definition**. Historical draft branches retain their pinned files, and GitHub manual runs can select another branch. No historical branch was rewritten and no native release was published.

## Remaining coordinated work

- Integrate native source and its existing verification gates before creating a real native ERIDION release pipeline; resolve the recorded G3E/G3F preview-label mismatch at that stage.
- Review dependency upgrades separately. Next 12/React 17 and the current `next export` build contract cannot be replaced safely through a metadata-only lockfile refresh.
- The SVG maintenance tool directly imports transitive `@xmldom/xmldom` 0.7.13, whose lockfile carries a critical-issues deprecation notice. Its dependency declaration/update and output validation remain a dedicated maintenance task.
- Review remaining packaging-tool pins, third-party distribution requirements, and end-to-end acceptance before any production publication.

Source references: [GitHub workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax), [setup-node v4 caching behavior](https://github.com/actions/setup-node/blob/v4/README.md), and [manual workflow branch selection](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manually-run-a-workflow).
