# Elite Dangerous wiki snapshot

This directory preserves a **27 May 2022** snapshot of material from the [Elite Dangerous wiki at Fandom](https://elite-dangerous.fandom.com). It is historical reference data, not a current game-data feed.

The inherited notice identifies the material as **Creative Commons Attribution–ShareAlike** and links to [Fandom's licensing information](https://www.fandom.com/licensing). Preserve source attribution, applicable notices, and backlinks when using derived text. This note does not infer a license version that is absent from the local evidence.

## Why the snapshot is kept

The snapshot is still a build input. [scripts/build/build-data.js](../../../scripts/build/build-data.js) reads:

```text
elite_dangerousfandomcom-20220527-wikidump/
  elite_dangerousfandomcom-20220527-current.xml
```

Its `codexArticles()` transformation parses the snapshot and uses article quotations to enrich commodity reference data. The active outputs are `commodity-descriptions.json` and `all-commodites.json` (the retained filename spelling); full codex-page and index export is currently disabled. Removing the XML because it is old would break that regeneration path. Existing source data and generated descriptions must not be treated as authoritative for new ships, modules, game mechanics, or current community events.

## Preserved provenance

| File | Purpose |
| --- | --- |
| The dated XML dump | Source article content consumed by the data builder. |
| The dated titles list | Historical page inventory from the export. |
| `config.txt` in the dump directory | Historical exporter configuration and resume metadata. Treat it as data; it is not ERIDION application configuration. |
| `dumpgenerator.py` | The inherited WikiTeam exporter, with its original copyright and GPL notice. It documents how the snapshot was obtained. |

The exporter and the wiki content have separate licenses. Keep the WikiTeam notice in the exporter and the content attribution above.

## Maintenance policy

The previous README's Python 2 bootstrap and `sudo` installation commands are retired as present-day setup instructions. The exporter is retained for historical provenance, not added to the native build or runtime.

A future refresh must identify the actual source, applicable terms, export date, and transformation; review the resulting content and attribution before replacing the snapshot. Use a maintained export mechanism chosen for that work. Do not silently fetch new wiki content during ordinary builds.

Changes to the derived data should be reviewed alongside the parser and its consumers. Native ERIDION's provider/source policy remains separate from this dated offline reference corpus.
