# Landing Hero Reconstruction

## Reference

- **Measured:** Source image:
  `/Users/favour/Downloads/WhatsApp Image 2026-08-07 at 9.06.26 AM.jpeg`
- **Measured:** Source dimensions: `735 x 420`.
- **Measured:** Source aspect ratio: `1.75:1`.
- **Measured:** Canonical viewport: `1440 x 823` at DPR `1`.
- **Measured:** Proportional scale from source to canonical viewport:
  approximately `1.959`.
- **Unknown:** The original page viewport, source font, exact CSS values, hidden
  pipe geometry, and full-resolution material textures are not present in the
  compressed screenshot.

## Observable Composition

| Element          | Source evidence                   | Canonical target                    | Confidence        |
| ---------------- | --------------------------------- | ----------------------------------- | ----------------- |
| Header           | About `40px` tall                 | About `78px` tall                   | Measured estimate |
| Outer margin     | About `38px`                      | About `74px`                        | Measured estimate |
| Heading origin   | About `(38, 88)`                  | About `(74, 172)`                   | Measured estimate |
| Heading box      | About `292 x 109px`               | About `572 x 214px`                 | Measured estimate |
| Heading lines    | Three                             | Three                               | Observed          |
| Supporting copy  | About `230px` wide at `y=215`     | About `451px` wide at `y=421`       | Measured estimate |
| Primary CTA      | About `65 x 19px` at `y=255`      | About `127 x 37px` at `y=500`       | Measured estimate |
| Pipeline art     | Roughly `x=300..720`, `y=55..315` | Roughly `x=588..1411`, `y=108..617` | Measured estimate |
| Trust-strip copy | Around `y=378`                    | Around `y=741`                      | Measured estimate |
| Trust marks      | Around `y=393..410`               | Around `y=770..803`                 | Measured estimate |

## Visual Characteristics

- **Observed:** The background is near-black with a soft graphite glow behind
  the visual and a subtle lower vignette.
- **Observed:** The heading is a heavy geometric sans-serif with tight leading,
  compact tracking, rounded bowls, and three lines.
- **Observed:** Header controls are deliberately small and low-contrast compared
  with the hero heading.
- **Observed:** The CTA is a compact white pill with dark text and a small
  trailing arrow.
- **Observed:** The right visual contains silver tubes, dark structural rails,
  warm metallic coupling rings, and restrained cool-blue node lights.
- **Observed:** The visual is framed as a stable studio composition rather than
  an interactive scene.
- **Observed:** The lower trust strip is intentionally quiet and has much lower
  contrast than the hero.
- **Inferred:** The source likely uses a custom or commercial geometric sans.
  The exact family cannot be identified reliably from this JPEG.
- **Inferred:** Manrope is a closer local substitute than Source Sans 3 for the
  headline because of its geometric width, rounded forms, and compact heavy
  weights. Source Sans 3 remains suitable for dense product UI copy.
- **Unknown:** The customer logos, hidden pipe paths, material maps, and mobile
  layout cannot be reconstructed exactly from the supplied evidence.

## Palette

The values below are reconstruction tokens chosen from visible regions of the
compressed reference. JPEG compression and the absence of source color data
make exact sampling unreliable.

| Token         | Value     | Evidence             |
| ------------- | --------- | -------------------- |
| Hero black    | `#050505` | Observed / estimated |
| Raised black  | `#101012` | Observed / estimated |
| Graphite      | `#292a2e` | Observed / estimated |
| Primary white | `#f3f3f1` | Observed / estimated |
| Muted silver  | `#a7a8ad` | Observed / estimated |
| Pipe silver   | `#b9bbc0` | Observed / estimated |
| Warm coupling | `#9b725e` | Observed / estimated |
| Node blue     | `#8da8e8` | Observed / estimated |

Semantic success, warning, danger, and information colors remain separate from
the monochrome brand palette.

## Current Conflicts

- The current hero uses a cream/green palette instead of black and metallic
  neutrals.
- The current Fraunces editorial headline conflicts with the geometric reference.
- The current hero is vertically centered and oversized instead of using the
  reference's compact upper-left alignment.
- The current hero has an eyebrow, two CTAs, and a fee note that are not present
  in the reference composition.
- The current marketing visual is an animated wallet, not a pipeline scene.
- The current header is a light, bordered, sticky bar with only two navigation
  links; the reference uses a compact dark header with a denser navigation row.
- The current page has no quiet trust/capability strip inside the opening
  viewport.

## Implementation Plan

1. Remap existing frontend brand tokens to black, graphite, white, and cool
   metallic neutrals while preserving semantic status colors.
2. Use Manrope locally for marketing display typography and retain Source Sans 3
   for application copy.
3. Rebuild the marketing header density and homepage hero around the canonical
   `1440 x 823` composition.
4. Create a static procedural React Three Fiber pipeline scene with reusable
   curves, tube geometry, couplings, and light nodes.
5. Add a lightweight SVG fallback for compact viewports, reduced motion,
   save-data, or unavailable WebGL.
6. Keep all existing content below the visible hero and preserve routes,
   authentication, and banking behavior.
7. Capture canonical reference, rendered, side-by-side, overlay, and difference
   artifacts, then iterate against measurable layout differences.
