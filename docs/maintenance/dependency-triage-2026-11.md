# Dependency and advisory triage — 25 November 2026

Branch `go/g3l-reliability-build-identity`, based on `go/g3k-native-performance` at `01639056c59bc5418d0b508bb9166ce0d19e0c26`. The advisories were regenerated on the day with `npm audit --json`, using npm 11 and Node 24.22.1 against the committed `package-lock.json`. `npm audit fix --force` was not used.

## Baseline

| | Critical | High | Moderate | Low | Packages |
| --- | --- | --- | --- | --- | --- |
| Before G3L (lockfile at the G3K base) | 7 | 26 | 9 | 3 | 45 |
| After G3L compatible updates | 7 | 23 | 9 | 3 | 42 |

The audit count is the number of affected *packages*. It is not a count of exploitable defects in ERIDION. Before G3L, 38 of the 45 were reachable only from development and build tools. The other 7 were also reachable from the legacy Node runtime's production dependency tree (`npm ls --omit=dev`).

## How each advisory can be reached

ERIDION has three separate surfaces:

1. **Native runtime.** This is the Go desktop executable (`go/cmd/daedalus`). It embeds the static export of the shared React client (`build/client` → `go/internal/frontend/dist`) and serves it from its own loopback/paired HTTPS server. No Node.js code runs at native runtime: no Next.js server, image optimizer, middleware, rewrites or `node_modules`. Only Next's *client* framework chunks and React run in the WebView or paired browser.
2. **Legacy runtime.** This is the inherited Node service (`src/service`), packaged by `@yao-pkg/pkg` for the legacy Windows application, together with the same client. It is kept intact but is not the native product.
3. **Build tools.** These cover the Next static export, font, icon and SVG generation, `node-gyp`, the test and lint tools, and `nodemon`.

| Package (severity) | Native runtime | Legacy runtime | Build only | Disposition |
| --- | --- | --- | --- | --- |
| `next` 12.3.7 (critical; direct dev dependency) | **Not reachable.** All 17 advisories concern the Next server: image optimizer RCE/DoS, middleware/proxy bypass, rewrites SSRF/smuggling, cache poisoning, response forwarding. A static export served by Go contains none of these. | Not reachable. The legacy app also serves the static export; `next dev` is a development tool. | Yes (`next build`/`export`) | **Deferred: separate migration.** The only fix is Next 16 (major), which removes `next export`, requires React 18+ and changes the asset contract that both the Go embedding and the legacy package rely on. It needs its own branch with every browser and visual gate. Not changed here. |
| `postcss` (high; transitive of Next) | Not reachable. It only processes our own CSS at build time. | Not reachable | Yes | Deferred with the Next migration. The advisories need attacker-controlled CSS or source maps at build time. |
| `axios` 1.15.0 (high; direct) | Not reachable. Native providers use Go `net/http` with the G3I provider policy. | **Reachable** from `src/service/lib/edsm.js`, `event-handlers/galnet.js`, `mining.js` and `integrations.js` (outbound provider HTTP). | — | **Updated** to 1.21.1 (same major; ≥14 days old). Fixes all 37 listed advisories (all `<1.20.0`). |
| `follow-redirects` 1.16.0 (high; via axios, http-proxy) | Not reachable | Reachable through axios redirects | — | **Updated** to 1.16.3 (fixes all three). |
| `form-data` 4.0.5 (critical title, high in this range; via axios) | Not reachable | Reachable only if multipart bodies are built from attacker-controlled field names; legacy calls send JSON or query strings. | — | **Updated** to 4.0.6. The critical advisory affects only `<2.5.4`. The installed 4.x line carries the CRLF advisory, fixed in 4.0.6. |
| `ws` 8.20.0 (high; direct) | Not reachable. Native sockets use `github.com/coder/websocket`. | **Reachable** from the legacy socket server (`src/service/lib/socket-server.js`); loopback-bound. | — | **Updated** to 8.22.2. It fixes both advisories (`<8.21.0`) and is ≥14 days old; 8.23.0 was only 13 days old. |
| `glob` 13 / `minimatch` 10 / `brace-expansion` 5.0.5 (moderate/high) | Not reachable | Reachable only with attacker-chosen patterns. Legacy `elite-log.js`/`elite-json.js` use fixed journal patterns. | `scripts/build/build-data.js` | `brace-expansion` **updated** to 5.0.12 (within `minimatch`'s `^5.0.5`). The `glob` fix is `glob@14` (major); **deferred**, since patterns are not attacker-controlled. 5.0.13 and 5.1.0 were too new. |
| `jimp`, `jpeg-js`, `resize-img`, `to-ico` | Not reachable | Not reachable | Icon generation | No change. Inputs are repository images. |
| `minimist`, `mkdirp`, `request`, `tough-cookie`, `qs`, `uuid`, `form-data` 2.x (via `request`) | Not reachable | Not reachable | Legacy build helpers | No change. Needs replacement of the abandoned `request` chain in a legacy-build maintenance task. |
| `tar`, `cacache`, `make-fetch-happen`, `node-gyp`, `ttf2woff2`, `@tootallnate/once`, `ip-address` | Not reachable | Not reachable | Native add-on builds for font tooling | No change. Build-time only, with no untrusted archives. `ttf2woff2`'s install script is not approved by npm 11. |
| `svgtofont`, `svg2ttf`, `svgo`, `@xmldom/xmldom` | Not reachable | Not reachable | Icon font build | No change. Inputs are repository SVGs. `@xmldom/xmldom` 0.7.13 needs a dedicated tool update (see main's `package-and-workflows.md`). |
| `nodemon`, `simple-update-notifier`, `semver`, `braces`, `micromatch`, `readdirp`, `decode-uri-component`, `source-map-resolve`, `url-regex`, `nanoid`, `source-map-js`, `js-yaml`, `undici`, `esbuild`, `@yao-pkg/pkg` | Not reachable | Not reachable (`pkg` packages but is not shipped) | Development tooling | No change. Fixes are major or dev-only, and they are not part of any shipped surface. |

## Results

Changes (`package.json` and `package-lock.json` only):

| Package | Before | After | Released | Why this version |
| --- | --- | --- | --- | --- |
| `axios` (direct, range `^1.21.1`) | 1.15.0 | 1.21.1 | 2026-11-09 | Newest 1.x at least 14 days old; fixes every listed axios advisory. |
| `ws` (direct, range `^8.22.2`) | 8.20.0 | 8.22.2 | 2026-11-08 | Fixes both advisories (`<8.21.0`). 8.23.0 was 13 days old. |
| `follow-redirects` (via axios, http-proxy) | 1.16.0 | 1.16.3 | 2026-11-08 | Fixes all three. |
| `form-data` (nested under axios) | 4.0.5 | 4.0.6 | 2026-06-12 | Fixes the 4.x CRLF advisory. |
| `brace-expansion` (nested under glob/minimatch) | 5.0.5 | 5.0.12 | 2026-09-14 | Fixes both 5.x advisories within `minimatch`'s `^5.0.5`. 5.0.13 and 5.1.0 were too new. Only this lockfile entry was changed, so dev-only copies are untouched. |

After the change, the audit reports 42 packages (7 critical, 23 high, 9 moderate, 3 low). axios, ws and follow-redirects are clear. The `form-data` and `brace-expansion` findings that remain are the dev-only copies: `form-data` 2.3.3 under `request`, and `brace-expansion` 1.x/2.x under build tools. The production copies (4.0.6, 5.0.12) are outside the vulnerable ranges. In the legacy production tree only `glob`/`minimatch` (moderate, major-version fix) remain, and nothing remains in the native runtime.

Evidence:
- `npm ci` from the updated lockfile succeeds.
- Legacy security regression suite (`npm test`): 219/219.
- `lint:security`, `lint:go-assets` and `test:go-assets`: 37/37.
- The client static build and every browser/visual gate pass on the final source (see the G3L handoff ledger).

The axios/ws users are in the legacy service only. No native code path changed.

## What this does not establish

- No advisory is claimed to be a proven exploit in ERIDION. No claim is made that an unreachable advisory is harmless in some other deployment, for example running `next start` or `next dev` exposed to a network.
- The Next 12 → 16 migration is the largest remaining dependency risk for the shared client and build. It is a product-level maintenance task with full visual regression and is outside G3L.
