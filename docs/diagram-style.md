# Diagram style

This is the written contract for Mathua's diagrams: what the existing assets look like, what the
renderer does to them, and the rules a new figure follows. It is an **analytical-tier** doc
(AGENTS.md section 6) and carries measurements, so `scripts/check_ste.py` does not cover it.

## 1. The decision

New and reclaimed figures target the **Algebrica cool dialect**: the one the majority of the live
library uses. The reason is fit rather than authorship. A lesson page shows several figures drawn
from the same source, so a figure that does not match its neighbours is worse than no figure, and
the cool palette is what most pages already carry.

The alternative was a new unified house style (theme-aware ink through `currentColor`, one
palette, an explicit font and background) applied to new figures and migrated onto the vendored
ones later. That is the better long-term answer and the worse answer for this pass: it makes every
new figure a visible outlier next to the vendored ones until a migration lands, and the migration
is a larger job than drawing the figures.

## 2. The dialects already in the tree

Assets live under `web/next-app/public/diagrams/`. Three dialects coexist, and they are not
interchangeable.

| Dialect | Where | Ink | Axes and guides | Captions | Fill | Structure |
|---|---|---|---|---|---|---|
| Algebrica cool | `diagrams/algebrica/`, vendor | `#333333` | `#BABABA` | `#636363` | `#E6E6E6` | tool export: `<?xml?>`, `version="1.1"`, `xmlns:xlink`, `<g fill="none" fill-rule="evenodd">`, arrowheads as filled paths, `<text>` with no `font-family`, no background |
| Algebrica warm | `diagrams/algebrica/`, vendor | `#312f2f` | `#BEB8B7` | `#636060` | `#E9E3E2`, `#F4EFEE` | the same tool dialect, a warmer palette |
| Mathua local | `diagrams/algebrica/` and `diagrams/arithmetic/` | `#333333` | `#BABABA` | `#636363` | `#E6E6E6` | hand-written: `<svg width="Npx">`, an opaque white `<rect>` background, and the arithmetic set uses bare `<text>` (theme ink) plus `#8a8f98` |

The palette is measured, not chosen: across the vendored SVGs, `#333333` and `#BABABA` are the
most frequent colors by a wide margin, and the second family is the same syntax in warm grays.

## 3. What the renderer does to every figure

`web/next-app/components/LessonDiagram.tsx` decides how a figure reaches the page, and it is not
neutral:

- An **SVG is fetched and inlined** after a DOMPurify pass (`SVG_PROFILE` keeps shapes, text and
  containers; scripts, event handlers, `foreignObject` and external references never survive).
  The SVG's own fills and strokes are what ship; nothing remaps them to theme tokens, so the
  "themeable" wording in the comments describes an intent, not the code.
- A **PNG goes through `next/image`**, so its type is baked into the file.
- `web/next-app/app/globals.css:716` forces the type of every inlined figure to a monospace
  face: `.lesson-diagram text { font-family: var(--font-jetbrains-mono), ... }`. That is why the
  vendored files carry **no `font-family`** — they are drawn to inherit the host font, and the
  host font is JetBrains Mono. A figure that hard-codes `font-family="serif"` is overridden.
- `.lesson-diagram` paints the figure panel with `background: var(--surface)`.
- Every inlined figure is bounded to **180px tall** — `LessonDiagram`'s `maxHeight` default, handed
  to the panel as the `--lesson-diagram-max-height` custom property — and the SVG scales down to
  fit it. The bound used to sit on the panel next to `overflow: hidden`, which **clipped** a taller
  figure instead of shrinking it. Because 123 of the 127 mapped SVGs are taller than 180px and the
  arithmetic set is 360x200, nearly every diagram lost its bottom edge at any width that did not
  force a scale, which on a desktop panel is every width. Fixed 2026-10-15;
  `web/next-app/e2e/lessonDiagramFit.spec.ts` holds it, with a negative control that restores the
  old rule and asserts the same measurement reports a clip.

**Dark mode, and why local figures paint a background.** `--surface` is near-white in the light
theme and `oklch(0.21 0 0 / 0.8)` in the dark theme, and there is no `filter` or dark-specific rule
for `.lesson-diagram`. A vendored cool or warm SVG is transparent with dark ink, so on the dark
panel its strokes and labels sit dark-on-dark. Every Mathua-authored figure avoids this by carrying
an opaque `<rect fill="white">` of the viewBox's size. That is why the two cool variants differ on
exactly this one point: the local figure paints a background and the vendored one does not, and a
new local figure keeps the rect so it stays legible in both themes.

## 4. The cool dialect, as rules

A new figure matches by following these, read off the vendored set and the local one:

- **Canvas.** `<svg width="Wpx" height="Hpx" viewBox="0 0 W H" ...>` in pixels, sized to the
  content. The vendored files keep `version="1.1"` and `xmlns:xlink`; new hand-written files need
  neither, but the closing shape is the same.
- **Ink and axes.** Primary ink `#333333`; axes, guide lines and construction lines `#BABABA`;
  annotations `#636363`; light fill `#E6E6E6`.
- **Strokes.** `stroke-width="1"` for guides, `1.5` for axes, `2` for the emphasized curve or
  line. Guides are dashed with `stroke-dasharray="4"`.
- **Arrowheads.** A small filled triangle in `#BABABA`, about 7.4 long and 6.2 wide. The vendored
  files write it as a `<path fill-rule="nonzero">` and the local ones as a `<polygon>`; the shape is
  the same, so either is a match.
- **Type.** `<text font-size="11|12|13|14" fill="...">`. Set `text-anchor="middle"` to center a
  label on its point. Do not bother with `font-family`: the stylesheet overrides it with JetBrains
  Mono at render time, so the `font-family="serif"` the local files carry is inert.
- **Markers.** Key points are small filled `<circle>` or `<ellipse>` markers of radius about 2.5
  in the primary ink.
- **Emphasis.** The technique the vendored set uses is weight, not color: the highlighted curve is
  solid ink at `stroke-width="2"` and the rest of the same function is drawn light. Where a
  function is compared across a family, the trig colors `#2980b9` (sine), `#e74c3c` (cosine) and
  `#27ae60` (tangent) are the established accents.
- **Background.** An opaque `<rect fill="white">` the size of the viewBox, as every local figure
  has, so the figure is legible on the dark panel (section 3). This is the one place a local figure
  deliberately differs from a vendored one.
- **A fetched raster is converted onto these greys, because the renderer applies nothing.** A
  lesson figure reaches the page as a plain `<img>` with no filter, so a tone that is not baked into
  the asset does not exist, and an OpenStax drawing arrives in saturated teal and orange beside line
  art. The conversion is grey, the tone stretched to the full range, then a lookup onto the four
  greys above plus white — which is a dialect color too: it is the fill of 40 vendored shapes and
  the `<rect>` every local figure carries, so the paper stays white rather than becoming a grey card.
  `docs/figure-reclaim.md` section 2 records why, and section 7 the recipe and its measurements.

## 5. Where a new figure is wired

There are two paths, and a figure needs the one that matches where it appears.

1. **A concept diagram** (the dual-coding card shown while a learner practices) is keyed by concept
   in the `conceptDiagrams` map at `internal/engine/engine.go:762`, which
   `diagramForConcept` reads. Add the concept id and the asset path to that map, place the asset,
   and add a `data/diagrams/meta.json` entry.
2. **A lesson figure** is a markdown image in the lesson body whose target is an absolute
   `/diagrams/...` path, or a bare filename that the renderer resolves against the vendored
   directory. It needs **no** `meta.json` entry and **no** engine mapping: the body references it
   and `public/` serves it. Give the image real alt text, because the alt is the accessible name
   whenever no metadata title exists. A metadata entry is for a concept diagram, and the audit
   rejects one for an asset the engine map does not name (section 6), so the two paths do not
   overlap.

The metadata entry carries `concepts`, `title`, `description`, `source`, `upstream` and `license`.
For a vendored Algebrica asset, `upstream` is the `github.com/antoniolupetti/algebrica` link. For
Mathua's own work, `source` is `local` and `license` is `Mathua (own work)`.

**Publish the copy.** The audit and the engine read `data/diagrams/meta.json`; the page fetches
`web/next-app/public/diagrams/meta.json` through `LessonDiagram.tsx:44`. Nothing generates one
from the other, so a mapping is only live after the copy:

```bash
cp data/diagrams/meta.json web/next-app/public/diagrams/meta.json
```

`scripts/audit_lessons.py` now fails when the two disagree, which is rule 15 there. Before that
rule was added the published copy was 18 entries behind the source, so 18 diagrams lost their
title and their attribution line at runtime while every check passed.

## 6. What the checkers enforce

`scripts/audit_lessons.py` reads the map in `internal/engine/engine.go` and requires, for every
mapped asset: a `meta.json` entry, a `source` of `algebrica` or `local`, a non-empty `title`, and
for an Algebrica SVG an `upstream` link. It also rejects a `meta.json` entry for an asset the map
does not name, and it requires Mathua-authored mappings outside the vendored directory to be SVG.

**The mislabel that was fixed.** `scripts/gen-diagrams.py` writes into the vendored directory and
24 of its outputs were recorded as `source: "algebrica"` with `upstream: null` and Algebrica's
`CC BY-NC 4.0` license, while those same 24 paths were exactly the `UNLINKED_GRANDFATHERED` set in
the audit — an allowlist that existed only to stop the upstream rule firing on our own work. On
2026-10-15 the 24 were relabeled `source: local` and `Mathua (own work)`, and
`UNLINKED_GRANDFATHERED` was emptied. Do not add entries back: a vendored Algebrica SVG must link
its upstream, and Mathua-authored diagrams are not Algebrica's to license.

**The rendering guard.** `web/next-app/e2e/lessonDiagramFit.spec.ts` measures a step diagram in
`/learn` and fails if the panel clips it. `lessonFigures.spec.ts` cannot see that defect: it
measures the reference panel, where the article renders through `KatexContent` as a plain `<img>`,
and an `<img>` does scale under `max-height`. Only the inlined-SVG renderer clipped.

**The text-box guard.** `web/next-app/e2e/diagramTextBoxes.spec.ts` measures every `source: local`
figure and fails when a label's rendered box leaves the `viewBox`, because an inline `<svg>` clips to
its viewport and a figure is drawn to the pixel in its own coordinate system. Twelve of the seventy
local figures were clipping their own text when it was written, and each looked like a mistake rather
than a crop: the place-value row read `1,247 = 1 thousand + 2 hundreds + 4 ten`, the resultant
vector's `u+v` sat at x=410 in a 400-wide canvas, `linear-equation-graph.svg` carried its `-4` tick
24px below a 300-tall one, and `inverse-trig-graphs.svg` had three labels left behind by an earlier
edit, one of them a duplicate `-π/2` at y=284.

The measurement is the rendered `getBBox()` rather than an estimate from character counts, and the
two agree because the host face is monospace: JetBrains Mono advances 0.6 em per glyph, as does the
fallback beside it. The spec asserts that ratio before it measures anything, so a machine whose
monospace is narrower fails as a broken harness instead of passing a corpus that clips on the web.
Its negative control draws a label past a 100-wide canvas and requires the measurement to report it;
restoring the old `vector-addition.svg` makes the real case fail by the 35.2px it was wrong by. Only
the local set is measured, for the reason section 7 gives.

## 7. Open items

- The 24 relabeled assets still sit under `diagrams/algebrica/`, which is now a partly false
  directory name. Moving them to `diagrams/local/` updates the engine map, the metadata keys and
  the published copy together; that is a separate change and is not done here. New local figures
  go under `diagrams/arithmetic/` or a sibling that names their subject, not the vendored
  directory.
- Dark mode is settled for local figures and left alone for vendored ones (section 3): a new local
  figure carries its own white background, and the vendored cool and warm figures stay transparent
  and so stay dark-on-dark on the dark panel. Repainting the panel or migrating the vendored set
  are the larger options that remain open.
- **Forty-one of the fifty-seven vendored SVGs clip their own commentary.** Most of the Algebrica
  figures carry a `<text>` paragraph — the author's description of the figure, in annotation grey at
  the top of the canvas — and in 41 of them that paragraph is wider than the `viewBox`:
  `heaviside-function-1.svg` runs 111.6px past a 313-wide canvas, so the sentence is cut mid-word
  ("...jumps from 0 to 1 at the origin. The filled" and then nothing). The cause is the face, not the
  file: section 3's rule forces every inlined figure's text to JetBrains Mono at 0.6 em, while the
  files were drawn for a narrower face and their canvases were sized to the drawing rather than to
  the prose. Two options and neither is taken here — widen each canvas, which shrinks the drawing
  under the 180px bound, or drop the commentary, which is the author's voice and not ours to remove.
  The local set is measured and guarded (section 6); this one needs the decision first.
