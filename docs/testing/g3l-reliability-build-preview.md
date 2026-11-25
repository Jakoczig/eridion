# ERIDION G3L experimental unsigned preview

Windows x64 requires Windows 10/11 and Microsoft Edge WebView2 Runtime. Linux x64 requires GTK 4 and WebKitGTK 6 (Ubuntu 24.04: libgtk-4-1 libwebkitgtk-6.0-4). Extract the archive and run eridion.exe / eridion from a user-owned directory. Linux may require chmod +x eridion.

## Verify before running

These are experimental unsigned executables, not installers or releases. No profiles, keys or credentials are included. Keep the legacy application intact.

1. Check the hashes: Windows `Get-FileHash -Algorithm SHA256 eridion.exe, README.md, BUILD-IDENTITY.json`; Linux `sha256sum -c SHA256SUMS`.
2. Check the identity: `eridion --build-info` (Windows `eridion.exe --build-info`) prints the identity embedded at build time. It opens no window and touches no profile or network. Its `version`, `revision`, `milestone` and `target` must equal `BUILD-IDENTITY.json`, and `vcs.revision` must equal `revision` with `vcs.modified` false.
3. The artifact name has the form `eridion-<version>-<milestone>-<12-hex revision>-<os>-amd64-experimental-unsigned-preview`. For pull request builds, `BUILD-IDENTITY.json` records the pull request head that was built and, separately, the GitHub synthetic merge commit, which was not built.

## What to test

G3L changes reliability and delivery. It does not change features.

- **Large journal folders.** A journal folder with more than 4,096 files is read in bounded batches. The newest journals are read first, so live play is never delayed while older history imports. When old history arrives out of order, local state is rebuilt in chronological order once.
- **Restore retry.** If a backup restore is cancelled or fails part-way, the original database is untouched and you can retry. An unexplained staging file is moved aside, never overwritten.
- **Event-driven navigation and FSD.** FSD charge, jump and arrival reach the selected owner and viewer screens as they happen, without one-second sampling. Gameplay acceptance:
  1. Test no-route and routed charge, jump, cancel and arrival.
  2. Test rapid consecutive jumps, manual dismissal, and switching between Galaxy and Local.
  3. Hide and resume a paired display: it shows the current state, without replaying stale transitions.
- **Signed spatial atlas.** A malformed signed spatial manifest (for example `"roots":[null]`) is rejected with the density fallback instead of crashing.
