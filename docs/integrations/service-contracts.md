# External service contracts

This file records what ERIDION actually sends to and accepts from each external service, and how well each contract has been verified. It describes the code at G3I (`go/internal/providers`). Adding an endpoint, parameter or permission means changing `allowedEndpoint` (`adapters.go`), the capability table (`types.go`) and this document in the same change.

## Rules shared by every service

| Rule | Implementation |
| --- | --- |
| Endpoint allowlist | `allowedEndpoint`: HTTPS only, no user-info or fragment, exact host and path. A request to any other URL fails before it is sent. Redirects are not followed. |
| Transport | 8 s timeout per request; `User-Agent: ERIDION/<version>`; replies above 2 MiB are rejected. |
| Rate limiting | A `429` sets a per-host cooldown from `Retry-After` (seconds or HTTP date, clamped to 1 min–1 h; 1 min if absent). During the cooldown no request goes to that host. Each capability also has a minimum interval (`IntervalSeconds`). |
| Upload retries | Only an explicit `429`/`503` is retried, at most 3 times with exponential backoff (2, 4, 8 min plus jitter). After that the outcome is terminal `failed`. A timeout or transport error is `unknown_delivery` and is never retried silently, so an event cannot be delivered twice. |
| Consent | Uploads are off by default. They are enabled per provider and per category, only for the commander the live journal has witnessed, and consent is stored in the history database. Read lookups run only when the owner asks for them. |
| Credentials | API keys and tokens are kept in the OS vault (`vault_*.go`). They are never logged, never echoed into status, and never sent to any host but their own. |
| Source priority | Fresh Frontier data first, then the journal, then third-party services, which all rank equally (D0; `history/source_priority.go`). Third-party values never override the journal. |

Verification status:

- **live-verified**: checked against the real service by the owner.
- **fixture-verified**: covered by synthetic transport tests only.
- **source-verified**: the contract was read in the service's published code or specification.

No status below is live-verified. Automated tests never call production services.

## Frontier (Companion API)

| Item | Contract |
| --- | --- |
| Endpoints | `https://auth.frontierstore.net/auth` (browser) and `/token`. `GET https://companion.orerve.net/{profile,market,shipyard,fleetcarrier,communitygoals}`. |
| Auth | OAuth2 authorization code with PKCE. Redirect `https://eridion.fitoserver.org/auth/frontier/callback`; bearer access token. Refresh happens 30 s before expiry; a failed refresh asks the owner to reconnect. |
| Gate | Needs Frontier developer approval of the client. Read-only: there are no write endpoints. |
| Identity | `profile` must be read and match the selected, journal-witnessed commander (FID and name) before any other endpoint is used. CAPI module IDs are applied to the journal loadout only after commander, ship ID and slot are all verified (G3I Task 16). |
| Freshness | Snapshots carry the fetch time and outrank the journal while fresh (D0). |
| Errors | A non-200 reply or a failed refresh leaves the previous state in place, with a status reason. |
| Pagination | None. |
| Evidence | EDCD/FDevIDs `Frontier API` notes. **fixture-verified**; live use waits for application approval. |

## EDSM

| Item | Contract |
| --- | --- |
| Upload | `POST https://www.edsm.net/api-journal-v1` with commander name, API key and a batch of the allowed journal events (travel, exploration, loadout, materials, ranks). |
| Read (account) | `POST /api-logs-v1/get-logs` for the last 24 h flight log. |
| Read (public) | `GET /api-v1/sphere-systems` (`systemName`, `radius`, `showId`, `showCoordinates`), `GET /api-system-v1/stations?systemName=` and `/api-system-v1/stations/{market,shipyard,outfitting}?marketId=`. These run only for the owner-selected system, with no fan-out. |
| Auth | A personal API key bound to the commander. Public reads send no key. |
| Receipts | The top-level `msgnum` is 1xx OK, 2xx batch refused, or 5xx saved for later (`accepted_for_later`). Each event pairs positionally with a per-event reply: 1xx confirmed, 2xx/3xx rejected, 4xx rejected as an ignorable discard. Unreadable or uncorrelatable replies are `unknown_delivery`. Only confirmed events count as sent (`edsm_receipts.go`). |
| Evidence | EDSM journal API as cited by the EDMC EDSM plugin. The EDSM documentation pages refuse automated fetches. **fixture-verified**. |

## EDDN

| Item | Contract |
| --- | --- |
| Upload | `POST https://eddn.edcd.io:4430/upload/`. Schemas: `journal/1`, `commodity/3`, `outfitting/2` and `shipyard/2`, validated locally against the bundled EDCD schemas (BSD-3, pinned in `upstream-sources.lock.json`). |
| Auth | None. An anonymous per-installation `uploaderID`. |
| Acceptance | Any 2xx is accepted (the relay returns no per-message receipt). |
| Station stock | MarketID is kept exactly. Complete-or-suppressed: each item is classified by exact FDevIDs numeric ID (`catalog.ClassifyInventory`). Ordinary stock is published in full. Items that depend on the commander, rank, PowerPlay, Guardian unlocks or tech-broker unlocks are omitted, as the outfitting/2 and shipyard/2 schemas require. An item that is unknown, has an ambiguous symbol or has an unreviewed entitlement suppresses the whole snapshot, and the reason is shown in the provider status. |
| Evidence | EDCD/EDDN `live` schemas. **fixture-verified**: synthetic fixtures only, no production uploads in tests. |

## Inara

| Item | Contract |
| --- | --- |
| Endpoint | `POST https://inara.cz/inapi/v1/` (JSON `header` and `events`). |
| Auth | A personal API key in `header.APIkey`. The application must be whitelisted by Inara, so the gate stays closed until approval. |
| Read | `getCommanderProfile` (needs the verified local commander name) and `getCommunityGoalsRecent`. |
| Upload | Travel, loadout, materials, ranks (pilot 0–13, navy 0–14 with exact `rankProgress`), and credits (cash and loan from the same session's LoadGame; nothing is sent when either is unknown). |
| Acceptance | `header.eventStatus` must be 200/202, and every event status must be 200/202. |
| Evidence | Inara API documentation. **fixture-verified**. |

## EDAstro

| Item | Contract |
| --- | --- |
| Upload | `POST https://edastro.com/api/journal`, for selected public events only (`ScanOrganic`; carrier events, each category with its own opt-in). |
| Read | `GET /api/starsystem?q=` (one system name, up to 128 characters) and `GET /gec/json/rare`. |
| Auth | None. |
| Evidence | edastro.com API details page; the API is unversioned. **fixture-verified**. |

## EliteBGS

| Item | Contract |
| --- | --- |
| Endpoints | `GET https://elitebgs.app/api/ebgs/v5/{systems,factions,stations}?name=<exact>&page=<1..10>` and `/ticks`. |
| Current vs history | Current lookups send **only** `name` and `page`. On these routes `count` is "Number of history records" and switches the reply to history mode (it disables `timeMin`/`timeMax`). It is never sent for a current lookup (G3I Task 17). Historical lookups are not implemented. |
| Pagination | `page` is the server's page number. The reply envelope reports `page`, `pages` and `total`, which are surfaced as facts. Pages are capped at 10, with at most 32 docs per page. |
| Validation | Every returned doc must match the requested name, case-insensitively; otherwise the reply is refused. An empty `docs` list is a valid empty result. An error body or non-JSON reply is refused. |
| Auth | None. These are community observations, never treated as causal attribution, and nothing is uploaded. |
| Evidence | `elite-kode/elitebgs` `server/routes/elite_bgs_api/v5/{systems,factions,stations}.js` at `d458e6fc2709e178b0d05bf89169de6d8f71717d`; the public docs site is a client-rendered app with no fetchable specification. **source-verified + fixture-verified**. |

## Canonn

| Item | Contract |
| --- | --- |
| Endpoint | `GET https://api.canonn.tech/<collection>?system.id64=<id64>&_limit=32&_start=<offset>`, for the Guardian, Thargoid, biology and generation-ship collections. |
| Validation | Every site must carry the exact requested `system.id64`. Contributor names (for example `discoveredBy`) are dropped. |
| Auth | None; nothing is uploaded. |
| Evidence | Canonn API documentation. **fixture-verified**. |

## Raven Colonial

| Item | Contract |
| --- | --- |
| Endpoint | `GET /api/system/<id64>`, `/api/project/<uuid>` and `/api/project/<uuid>/{last,stats}` on the published Azure host. |
| Validation | Only project progress fields are kept, never commander or contributor lists. |
| Auth | None. Writes are unavailable until a canonical host and authentication model are approved. |
| Evidence | The Raven About page. **fixture-verified**. |

## Spansh

| Item | Contract |
| --- | --- |
| Endpoint | `GET https://spansh.co.uk/api/{system,body,station,dump}/<decimal id>`, with no query string. |
| Validation | The response entity must have the exact requested ID (`id64` or `market_id`). Body and station replies must carry a parent `system_id64`, and coordinates must be valid. |
| Not used | Route-plotting job APIs. |
| Auth | None. |
| Evidence | docs.spansh.co.uk (OpenAPI). **fixture-verified**. |

## Not covered here

BioScan contracts (Tasks 6/18) are G3J. Live verification of any upload needs a deliberate, consented action by the owner, and none has been made.
