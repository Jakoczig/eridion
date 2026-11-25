# Native release readiness — 25 November 2026

Status: **not release-ready.** G3L produces identity-verified, unsigned, expiring CI previews only. No release, tag, signing, deployment or publication was performed or configured.

## What is now in place (G3L Task 10)

- **One identity record.** `scripts/build/go-build-identity.cjs` builds it from the exact checked-out commit:
  - schema `eridion-build-identity/1`;
  - full and short revision;
  - version `2.2.0-g3l+<12-hex>`, following the version policy `<package version>-<milestone>+<revision>`;
  - milestone;
  - target OS and architecture;
  - commit timestamp, used as the source date for reproducibility, not the build time;
  - executable name and artifact name.

  It refuses a tree with tracked or untracked changes.
- **One record feeds every consumer.** The same record supplies:
  - the Go linker values (`internal/buildinfo`);
  - `eridion --build-info`, which prints the embedded identity and opens no window, profile or network;
  - the packaged `BUILD-IDENTITY.json`;
  - the artifact name;
  - the Actions outputs.
- **Packaging fails on any mismatch.** These must all agree, or `packageIdentified` stops:
  - the executable's own report: linker values, plus Go's independent `vcs.revision`/`vcs.modified`;
  - the manifest;
  - the file and artifact names;
  - the README milestone;
  - `SHA256SUMS`, which covers the executable, README and manifest.

  After writing, the package is re-read and verified.
- **Pull requests build the source commit.** The desktop workflow checks out the PR head, not GitHub's synthetic merge commit. The merge commit is recorded separately as `syntheticMergeCommit` with `syntheticMergeBuilt: false`.

## Configuration compared with main (read-only)

Main's corrections were compared with the native branch without merging main or overwriting native workflows: `eff5fe7` (README), `40d07eb` (inherited-file consolidation) and `ffc5f66` (package metadata and legacy workflows).

| Item | Main (`ffc5f66`) | Native stack (this branch) | Action in G3L |
| --- | --- | --- | --- |
| `package.json` identity | `eridion`, private, repository/homepage, Node `>=24 <25` | `daedalus`, Node `>=20`, no `private` | Not changed. Renaming touches lockfile root metadata and legacy packaging; adopt it with the native/main integration. Recorded. |
| `release.yml` | Artifact-only legacy installer, `contents: read`, pinned media digests, no release step | **Still the inherited definition**: manual `workflow_dispatch`, `contents: write`, unpinned `latest` media downloads, `softprops/action-gh-release` creating tag `v2.2.0` (non-prerelease) | **Not changed and not dispatched.** This is a release-readiness blocker: on this branch, running "Release" manually would publish a stable legacy release. Port main's artifact-only version when the stacks are integrated, or remove it. Changing it was outside "explicitly scoped configuration" here. |
| `ci.yml` | Legacy scope, read-only token, npm cache with unconditional `npm ci`, timeouts | Native-era legacy gates with stacked branch filters | Only `go/g3k-native-performance` was added to `pull_request.branches`, so the G3L PR runs both workflows. |
| README/summaries | Rewritten for native development | Older | Not changed. Consolidate at integration. |

## Repository protection (read-only, 25 Nov 2026)

`gh api` showed:
- `main` is **not protected** (HTTP 404: branch not protected);
- there are no repository rulesets;
- the repository is public;
- auto-merge and delete-branch-on-merge are off.

No required status checks are enforced. The stacked draft PRs rely on review discipline. Enabling protection or required checks is an owner decision and outside this milestone; nothing was changed.

## Remaining blockers before any native release

1. Code signing (Windows Authenticode; Linux packaging/signing policy) and a release process with provenance. Nothing is signed today.
2. Replace or neutralize the inherited `release.yml` on the native stack (see above).
3. Protected default branch with required native and legacy checks.
4. Next 12 → 16 static-export migration, which is the largest dependency risk (see `dependency-triage-2026-11.md`), with full visual regression.
5. Physical/real-game acceptance: Windows WebView2 and Linux WebKitGTK windows with a real game session, paired devices, and large real journal folders. CI and this environment use synthetic fixtures; the native window cannot be opened here.
6. A coordinated product decision on the native release version, replacing the inherited 2.2.0.
