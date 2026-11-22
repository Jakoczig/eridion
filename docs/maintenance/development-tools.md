# Retained development tools

## Controller SVG annotations

[scripts/annotate-device-svgs.mjs](../../scripts/annotate-device-svgs.mjs) preserves Varzeki's inherited SVG annotation utility, formerly `_dev-scripts-backup/annotate-svgs.mjs`.

It derives `data-label` and `data-labels` annotations for controller diagrams, including device-qualified controls, transformed label boxes, leader paths, and related icons. The [legacy visualizer](../../src/client/components/keybinds-visual.js) consumes those attributes, so this capability remains useful even though the tool is not part of ordinary builds.

**Status: manual reference utility, not a validated asset-generation command.** Moving it did not execute it or regenerate any diagram. Its executable logic and repository-relative path resolution were preserved; only its name and usage comments changed.

The tool writes diagram files in place and relies on `@xmldom/xmldom`, currently a transitive lockfile dependency. Before promoting it to supported automation:

1. Declare and review its XML dependency explicitly.
2. Add a preview/output-directory mode so regeneration can be reviewed before replacing source assets.
3. Check supported device mappings, annotation counts, transforms, and leader relationships against representative diagrams.
4. Verify repeatability and inspect the generated diff.

The inherited idempotence claim has not been established by this cleanup.

## Retired experiments

The removed `_tmp/` files were machine-specific probes, incomplete SVG diagnostics, or duplicated Steam/ControlSchemes discovery. The old SVG audit's attribute-count regex did not correctly count ordinary `data-label=` attributes.

The removed `extract-armour.js` relied on an external developer-drive cache and a serialization-specific regex. Future catalogue updates belong in the reviewed catalogue pipeline, with canonical identifiers, variants, provenance, and explicit unknowns.

The removed status-watcher experiment shared a content cursor between concurrently active watchers, allowing one observer to suppress another's samples. Its file-mtime difference did not measure native ERIDION or display latency. Future measurements must isolate observer state and separately measure file-to-ingestion, projection, transport, and rendered-display latency on representative workloads.

Original files remain available at the [pre-cleanup source revision](https://github.com/Jakoczig/eridion/tree/eff5fe719b0a82611255cdf846d166a24a7cc228). See the [audit decisions](inherited-files-audit.md) for the complete disposition.
