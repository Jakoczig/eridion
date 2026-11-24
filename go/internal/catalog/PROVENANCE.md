# Bundled reference data

These are static reference facts selected from data already committed in this
repository. They describe stock modules and item names, not measured player ship
statistics. The runtime never opens resource directories or personal game files.

- `shipyard.csv`, `outfitting.csv`, `material.csv`, `microresources.csv`, `commodity.csv` are exact
  copies of the corresponding `resources/data/edcd/fdevids/` source files.
- `modules.json` selects `symbol`, `ukName` (as `name`), `class`, `rating`, `mass`,
  `power`, and `ukDiscript` (as `description`) from every JSON file under
  `resources/data/edcd/coriolis/modules/`. Missing values remain null. Generation
  used the repository's existing bundled data; no network source was fetched.
- The public provenance label is `bundled_edcd_coriolis_reference`. The reference
  catalog's `massT` and `powerMW` describe stock modules. Observed engineering
  modifiers remain separate, and are not silently applied to these values.

The bundled snapshots may omit newer modules or labels. Unknown item symbols are
preserved; absent reference fields remain unknown.

G2M-A adds only the exact existing bundled `shipyard.csv` for human ship labels; it does not fetch or synthesize hull performance or license new ship imagery. Existing EDCD/FDevIDs attribution and repository third-party licensing continue to apply.

## G3I refresh (24 November 2026)

- `shipyard.csv`, `outfitting.csv`, `commodity.csv`, `material.csv` and `microresources.csv` are byte-identical to EDCD/FDevIDs revision `c7c29f7a13df543d4bc8b947eab29f88e06100fa` (the repository's default head when checked). The same bytes are kept in `resources/data/edcd/fdevids/`. Hashes, revision, check date and licence notes are recorded in `/upstream-sources.lock.json`; `TestUpstreamLockConformance` fails if any bundled copy drifts.
- Coverage before → after: 38 → 49 hulls (adds Python MkII, Type-8 Transporter, Mandalay, Cobra MkV, Corsair, Panther Clipper MkII, Type-11 Prospector, Caspian Explorer, Kestrel Mk II, Lynx Highliner, Anaconda Mk II); 1,068 → 1,243 module IDs (175 added, none removed, 23 changed: 3 hangar and 1 limpet relabels, 1 xeno scanner name/class/rating, 18 PowerPlay entitlement renumbers); commodities 256 → 270; microresources 79 rows with placeholder IDs → 196 with real IDs.
- 22 module symbols are shared by several numeric IDs (a standard module plus newer pre-built variants). They are listed and reviewed in `collisions-reviewed.txt`; the catalogue indexes outfitting by numeric ID first and keeps a symbol-to-candidates index, so a shared symbol is never collapsed (`catalog.ResolveModule`).
- `entitlement-review.csv` is a manual, evidence-backed review of entitlement availability for EDDN station publication. It is not upstream data and the refresh tool never writes it.
- Refresh only through `go run ./cmd/upstream-sources` (see `docs/testing/upstream-conformance.md`). FDevIDs carries no licence file; attribution to EDCD is retained in the README and the lock.
- `modules.json` (Coriolis-derived stock statistics) was not refreshed; its upstream revision was never recorded (lock status `unpinned`). Identifying a module does not imply statistics coverage for it.
