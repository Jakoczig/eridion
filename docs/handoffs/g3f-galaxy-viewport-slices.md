# G3F — Galaxy camera / XYZ slice separation

Base: G3E PR #27, `64ddd186f692e48b2e2c1f5a342be20de5d1ac12`.
Branch: `go/g3f-galaxy-viewport-slices`.

## Owner-observed defect

The 2026-11-22 Windows screenshots show fixed exterior X/Z, X/Y and Y/Z
rectangles with an unexpectedly small density/star image floating inside.
G3E `fittedFrame()` used `max(X/100,Y/8,Z/100)` LY per SVG unit.
At X=508, Y=117, Z=508 LY, Y alone expanded the main X/Z display
to 1,462.5 LY across; the requested X/Z volume therefore occupied
only 34.7% of its available square. The old projection's dynamic
`localGeometry` viewBox and selection clip made the side strips look
especially small.

## Targeted repair

- Keep exactly the G3E 100:8 fixed CSS grid and all touch controls.
- For the primary X/Z projection, derive a square, equal-LY-axis camera
  from **X and Z only**; Y is data-selection thickness, never a reason
  to shrink the main map.
- Use each panel's actual canonical SVG viewBox (100×100, 100×8, 8×100)
  rather than dynamically resizing the viewBox to match the current
  selection. This eliminates a second internal letterbox.
- Thin orthographic side strips intentionally display the selected
  Y range on their limited short axis. Their short-axis presentation
  scale is independent of the face-on panel; apparent two-dimensional
  distances within these compressed overview strips must **not** be
  interpreted as equal pixel/LY scale. All original XYZ positions and
  numerical distances remain unchanged.
- XYZ +/- edits update the *exact bounded spatial query* and
  selection-marker filter, but do not automatically zoom or pan X/Z
  camera. After a slice edit, Y presentation follows the current slice
  to keep the thin side strips usable. The camera deliberately moves
  only through Zoom, pan, Fit Route, mode selection or automatic framing.
- Remove the redundant selection clip from the assembled application
  map. The spatial backend already restricts point records to the
  requested XYZ volume; the visible map camera still clips the pixels
  to its rectangular viewport. Out-of-volume markers remain filtered.
- Keep signed spatial data, local-first search, atlas, browser privacy,
  owner/viewer and no-upload boundaries unchanged. No online source
  or new provider request is added.

## Geometry tradeoff

A genuinely full-bleed thin strip, exact full XYZ selection coverage,
and the **same LY-per-pixel in all three projections** cannot all be
guaranteed at once for independently adjustable axes. G3F prioritizes
the maintainer's full-size fixed panels and a physically isotropic
**main X/Z view**, while the side strips intentionally compress Y.
If an inspected X/Z slice exceeds the stable main-camera bounds,
some points fall outside the viewport until deliberate zoom/pan;
the query still covers the full selection and no false coordinates
or elongated circular markers are generated. Empty regions with
no community-reported positions remain empty: real data is not
fabricated to fill gaps.

## Regressions and acceptance

- New G3F service regressions verify camera independence from
  Y thickness, correct fixed side viewBoxes, absence of redundant
  content clip and XYZ editing with a frozen camera.
- The existing G3E real-browser viewport script now checks that XYZ
  changes cannot move the camera's X/Z marker positions. All existing
  bounding-rectangle, route/FSD, owner/viewer and provider checks remain.
- CI Windows/Linux desktop and legacy/security results must be read
  on the exact G3F head; source tests are not a substitute for actual
  Windows gameplay acceptance.

**Owner physical checklist:** compare the opening Galaxy projection
against previous 1/3-width appearance; zoom several steps and switch
to Local; change each X/Y/Z independently and verify fixed panel
rectangles and stable X/Z camera; confirm the side strips remain
occupied where actual points exist; then test Fit Route, real route
overlay, game jump/FSD restoration, and paired-laptop read-only view.
Also verify that enlarging a selection beyond the camera requires
intentional Zoom − to inspect the newly included outer region.
