# BioScan conformance

ERIDION evaluates the community exobiology criteria of EDMC-BioScan natively in Go. This file records which upstream revision is reproduced, how each predicate is evaluated, where the native evaluator deliberately differs, and what remains unsupported. It describes the evaluator at G3J (`go/internal/fieldops/bioeval.go`).

Passing these tests means the native evaluator matches the pinned upstream semantics on fixtures. It does not show that either is correct about the game, and the evaluator is not at complete parity while some predicates remain unsupported.

## Sources

| Source | Revision | Used for | Licence |
| --- | --- | --- | --- |
| [Silarn/EDMC-BioScan](https://github.com/Silarn/EDMC-BioScan/tree/fc6b85c3de48c8fb79fa9e4f79ad780e48f56629) | `fc6b85c3de48c8fb79fa9e4f79ad780e48f56629` (upstream head on 2026-11-24) | `rulesets/*.py` (19 files), `species.py`, `regions.py`, `nebula_data/sectors.py`; predicate semantics from `load.py` `value_estimate` | GPL-2.0-or-later; GPL-3.0 selected |
| [Silarn/EDMC-ExploData](https://github.com/Silarn/EDMC-ExploData/tree/fd4e188defd6405e46398fb0a54ac8669c9e2189) | `fd4e188defd6405e46398fb0a54ac8669c9e2189` (upstream head on 2026-11-24) | `bio_data/genus.py`: genus names, clonal distances, colour (variant) dependencies; Scan field mapping from `journal_parse.py` | GPL-2.0-or-later; GPL-3.0 selected |

The data files are vendored byte-identical under `go/internal/fieldops/upstream/`, with each repository's `LICENSE.txt`. They are pinned in `upstream-sources.lock.json` (`silarn-bioscan`, `silarn-explodata`) and marked `-text` so Windows checkouts keep the exact bytes.

`go/internal/fieldops/bioconvert` reads these files as Python literal data: dicts, lists, tuples, strings, numbers and booleans. Any expression, name or call is rejected. No Python is executed, imported or required, at build time or at runtime.

To regenerate (from `go/`):

```sh
go run ./cmd/bioscan-rules            # bio-rules.json from the vendored files
go run ./cmd/bioscan-rules -manifest  # predicate-support.json from the new rules
```

`TestBioRulesGeneratedFromPinnedSources` regenerates both files in CI and fails on any difference. It also fails if a rule key has no declared support level. Upstream changes follow `docs/testing/upstream-conformance.md`: check, fetch, diff, update the vendored files and the lock, regenerate, review.

Not vendored, so the dependent predicates remain unsupported:

- `nebula_data/reference_stars.py` (nebula distances);
- ExploData `RegionMapData.py` (galactic region lookup; a separately attributed derivative).

## Coverage

| | Sourced species | Without a fully supported alternative |
| --- | --- | --- |
| Before G3J (hand-converted rules, body-only evaluator) | 116 | 74 / 116 |
| G3J | 119 (116 ruleset literals plus Bark Mound, Amphora Plant and Radicoida Unicus from `species.py`) | 53 / 119 (51 / 116 ruleset literals; 2 / 3 `species.py`) |

A species has a *fully supported alternative* when at least one of its rulesets uses only natively supported predicates. Such a species can be decided once the journal evidence exists.

All 53 remaining species need the galactic region map (`regions`, or the singular `region` used by the 8 Brain Trees), except two:

- Electricae Radialem, whose alternatives need nebula proximity (decidable natively only by a nebula sector name);
- Stratum Aranaemus, whose upstream ruleset list is empty.

The machine-readable manifest is `go/internal/fieldops/predicate-support.json`. It lists every predicate's support level and the journal evidence it needs, and for every species: its rulesets, their support, whether a fully supported alternative exists, and its colour dependency (`parent-star`: 65, `materials`: 26, `none`: 28).

Identity is genus symbol plus species symbol. Genus and species are never merged, and localized labels never identify a genus (Task 6).

## Inference model

Each predicate is three-valued:

- **pass**;
- **fail**;
- **unknown**: the evidence it needs is missing, or the capability is not bundled.

These combine as follows:

- A ruleset **fails** if any predicate fails, **passes** if all pass, and is unknown otherwise.
- A species is **possible** if any alternative passes, **excluded** if all fail, and **unknown** otherwise.
- Colour dependencies can still exclude a possible or unknown species, as upstream does.
- An unknown result lists what is still missing, for example "temperature not scanned" or "galactic region map not bundled".

Insufficient evidence never becomes a positive prediction. *Possible* is a community suggestion, never a confirmed observation.

## Predicates

| Key | Evidence | Native rule (= upstream unless noted) |
| --- | --- | --- |
| `atmosphere` | `Scan.AtmosphereType` (an empty `Atmosphere` means none) | `'Any'` fails on none; a list must contain the type |
| `atmosphere_component` | `Scan.AtmosphereComposition` | a missing gas counts as 0 %; fails below the required percent. Unknown when the body has an atmosphere but no composition was scanned |
| `min_gravity` / `max_gravity` | `Scan.SurfaceGravity` (m/s²) | ÷ **9.797759**, inclusive bounds |
| `min_temperature` / `max_temperature` | `Scan.SurfaceTemperature` (K) | inclusive; a zero value is skipped, as upstream |
| `min_pressure` / `max_pressure` | `Scan.SurfacePressure` (Pa) | ÷ **101231.65625**; minimum inclusive, **maximum exclusive** (`>=` fails); a zero value is skipped, as upstream |
| `max_orbital_period` | `Scan.OrbitalPeriod` (s) | exclusive |
| `volcanism` | `Scan.Volcanism` (`""` is none; the older `"No volcanism"` is normalized to `""`) | list: substring, or exact with a `=` prefix. `'Any'` needs some volcanism; `'None'` needs none; `'!x'` needs some volcanism without `x` |
| `body_type` | `Scan.PlanetClass` | membership |
| `distance` | `Scan.DistanceFromArrivalLS` | fails below the value |
| `system` | system name | equality |
| `star` | star Scans of the system | any star matches a type, or a `(type, luminosity)` tuple with an optional `a`/`b`/`ab`/`z` suffix. A plain list entry never matches (upstream behaviour). Fails only after `FSSAllBodiesFound`, and only when the retained scans cover the reported body count |
| `main_star` | primary star Scan (distance 0) | as `star`, for the primary (not used by the pinned rules) |
| `parent_star` | primary and parent star Scans; parents derived from the body name as ExploData does | passes if the primary or a parent matches. Fails only when the primary and every parent are scanned |
| `bodies` | body Scans of the system | any body of the listed classes. Fails only after `FSSAllBodiesFound`, and only when the retained scans cover the reported body count |
| `guardian` | system `StarPos` | within a bundled guardian-zone radius |
| `tuber` | system `StarPos` | within a bundled tuber zone's min–max shell |
| `nebula` | system name | **partial**: passes on a nebula sector name; otherwise unknown |
| `regions` | galactic region | **unsupported**: always unknown |
| `region` (singular, Brain Tree) | galactic region | **unsupported**: always unknown. Upstream has no handler for this key, so upstream never restricts (see divergences) |

`star_check` mirrors upstream:
- A, B, F, G, K and M include their giant and supergiant types;
- D, C and W match by prefix;
- every other type matches exactly.

### Colours (variants)

ExploData colour tables can exclude species:

- **Star-dependent:** the first scanned parent star whose type has a colour supplies it, then the primary when it is not a parent.
  - With no colour, the species (or, for a genus-level table, the whole genus) is excluded only when that evidence is complete.
  - Otherwise it is unknown, which includes a black-hole primary: upstream then also considers stars orbiting the black hole, which is not modelled.
- **Element-dependent:** colours come from the body's `Materials`. Excluded when the materials are known and none qualifies; unknown when the materials were not scanned.

The resulting colours are reported as `variants`. A colour found while an earlier parent star is still unscanned is marked provisional in the candidate's evidence, because upstream would prefer that earlier parent once it is scanned.

## Intentional divergences

| Case | Upstream | Native | Reason |
| --- | --- | --- | --- |
| Missing body field (temperature, pressure, gravity, volcanism, atmosphere, orbital period) | treated as passing (`not value`) or not reachable | **unknown** | Missing evidence must not become a prediction (Task 6). A present zero still passes, as upstream |
| Star or body absent while the system is not fully scanned | eliminates | **unknown** until `FSSAllBodiesFound` | Upstream treats an incomplete scan as negative evidence |
| Nebula with no system coordinates | passes (`continue`) | **unknown** | Same reason |
| `regions` | evaluated with the region map | **unknown** | Region map not bundled (provenance) |
| Species with an empty ruleset (Stratum Aranaemus) | never listed | listed as **unknown**, "no sourced criteria" | Not claimed impossible |
| `"No volcanism"` | compared verbatim | treated as none | Older journal spelling of the same fact |
| Singular `region` key (8 Brain Trees) | ignored, so the species passes anywhere | **unknown** | A stated restriction that cannot be checked must not become a positive prediction (G3J review) |
| `FSSAllBodiesFound` with fewer retained scans than the reported body count (gap, quarantined file, more than 512 bodies) | treated as complete | **unknown** | Absence is evidence only when the scans are actually complete |

Quirk reproduced deliberately: a plain list inside a `star` list never matches.

## Boundary fixtures

These are in `go/internal/fieldops/bio_conformance_test.go`. Expected values are derived by hand from the pinned literals and `load.py`.

| Case | Expected | Pre-G3J native |
| --- | --- | --- |
| Tussock Stigmasis at 1012.8 Pa (max_pressure 0.01) | excluded (0.010005 ≥ 0.01) | possible (÷ 101325, `>`) |
| Pressure exactly at `0.01 × 101231.65625` | excluded (exclusive maximum) | — |
| Tussock Stigmasis at 2.7043 m/s² (max_gravity 0.276) | excluded (0.27601) | possible (÷ 9.80665) |
| Gravity exactly at `0.276 × 9.797759` | possible (inclusive) | — |
| Aleoida Arcus at 175 K / 180 K / 174.99 K / 180.01 K | possible / possible / excluded / excluded | — |
| Missing temperature | unknown, "temperature not scanned" | unknown |
| Aleoida Arcus at zero pressure | not excluded (zero skipped) | — |
| Bacterium Nebulus without materials / without a colour element / with polonium | unknown / excluded / possible (Gold) | possible in all three |
| Aleoida Arcus under an F primary / without star evidence | possible (Teal) / unknown | possible |
| B IVab star for `[('B','IV'), ('B','V')]` | pass | not evaluated |
| Gamma Velorum zone, Sol, no coordinates | pass / fail / unknown | not evaluated |
| Tuber zone Inner S-C Arm B 1 at 200 ly / at its centre | pass / fail (inside the 150 ly inner radius) | not evaluated |
| `bodies` without `FSSAllBodiesFound` / after it / with a water giant | unknown / fail / pass | not evaluated |
| Radicoida Unicus in HIP 87621 | possible | not sourced |

## Owner acceptance still required

These are fixture results. The following need real-game observation by the owner, because they cannot be shown with synthetic data:

- that possible, excluded and unknown candidates match what is actually found on real bodies, especially near the pressure and gravity boundaries above;
- predicted colour variants compared with the variant actually logged;
- Brain Tree, Bark Mound, Amphora Plant and the region-dependent species, which stay unknown natively;
- localized clients (for example German or French) showing the same candidates as English for the same body;
- live sample spacing during a real Log, Sample, Analyse cycle.
