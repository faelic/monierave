# Monierave Wallet Hero

## Direction

The wallet hero is an original Monierave animation built from native Three.js
objects through React Three Fiber. It is not a reconstruction of an external
video. The animation communicates three ideas: secure storage, multiple payment
instruments, and money moving through the Monierave network.

The holder opens, three branded cards rise into an asymmetric fan, two abstract
payment tokens travel along separate curves, and the composition closes into a
seamless `6.2s` loop.

## Structure

```text
src/features/wallet-hero/
  adaptive-wallet-visual.tsx
  monierave-wallet-scene.tsx
  wallet-runtime-policy.ts
  components/
    animation-controller.tsx
    banking-cards.tsx
    digital-card-holder.tsx
    payment-token.tsx
    responsive-camera.tsx
    scene-fallback.tsx
    studio-lighting.tsx
  config/
    wallet-config.ts
  motion/
    wallet-motion.ts
```

`adaptive-wallet-visual.tsx` is the public product boundary. It owns dynamic
loading, reduced-motion and low-power handling, viewport cleanup, fallback
continuity, failure recovery, and the pause control.

## Scene Design

- `walletMotionRoot` owns the restrained movement of the whole composition.
- `backPanelPivot` and `frontPanelPivot` provide real hinge transforms.
- Each card has an independent pivot, position, rotation, and stagger.
- Each payment token follows its own `CatmullRomCurve3` path.
- One native `THREE.AnimationClip` controls the complete deterministic loop.

The animation is intentionally calm. It has no pointer parallax, random physics,
camera orbit, particles, bloom, or exaggerated spring motion.

## Configuration

Replaceable values live in `config/wallet-config.ts`:

- `monieraveBrand` contains the five brand colors.
- `walletAssets` contains the card textures and static fallback.
- `walletMaterials` controls physical material properties.
- `walletTimeline` defines product-owned motion phases.
- `walletTransforms` defines holder proportions and hinge placement.
- `walletResponsive` defines camera, scale, DPR, and shadow limits.

Motion poses and token paths live in `motion/wallet-motion.ts`. Phase timing can
be adjusted there without changing any mesh component.

## Card Artwork

Card faces are replaceable SVG textures loaded in sRGB:

```text
public/assets/3d/cards/
  monierave-primary.svg
  monierave-secondary.svg
  monierave-supporting.svg
```

Final Figma exports should preserve the current card aspect ratio and safe
padding. Never place real account numbers, names, balances, or credentials in
these decorative assets.

## Runtime

- The scene is dynamically imported and does not block authentication.
- A static image remains visible while WebGL and textures load.
- DPR is clamped to `1-1.5`; shadows use a `512px` map.
- Animation unmounts outside its intersection boundary.
- Reduced motion, compact screens, save-data, low-power devices, and missing
  WebGL receive the static fallback.
- Geometry and cloned textures are disposed when the scene unmounts.
- The canvas has no keyboard or pointer interaction.
