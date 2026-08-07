export const monieraveBrand = {
  monieraveDark: "#292A2E",
  monieraveForest: "#111113",
  monieraveMint: "#A7A8AD",
  monieraveOffWhite: "#F3F3F1",
  monieraveSage: "#D7D7DA",
} as const;

export const walletAssets = {
  fallback: "/assets/3d/monierave-wallet-fallback.png",
  textures: {
    primary: "/assets/3d/cards/monierave-primary.svg",
    secondary: "/assets/3d/cards/monierave-secondary.svg",
    supporting: "/assets/3d/cards/monierave-supporting.svg",
  },
} as const;

export const walletMaterials = {
  card: {
    metalness: 0.08,
    roughness: 0.58,
  },
  holderExterior: {
    metalness: 0.04,
    roughness: 0.66,
  },
  holderInterior: {
    metalness: 0.06,
    roughness: 0.5,
  },
  satin: {
    metalness: 0.52,
    roughness: 0.34,
  },
  token: {
    metalness: 0.08,
    roughness: 0.38,
  },
} as const;

export const walletTimeline = {
  cardReveal: 1.1,
  closeComplete: 5.55,
  duration: 6.2,
  heroHold: 2.35,
  holderOpen: 0.75,
  retract: 4.25,
  tokenLaunch: 1.7,
} as const;

export type WalletVector = [number, number, number];

export const walletTransforms = {
  holder: {
    baseDepth: 1.08,
    baseHeight: 0.32,
    hingeY: -0.82,
    panelDepth: 0.16,
    panelHeight: 2.12,
    panelWidth: 4.35,
  },
} as const;

export const walletResponsive = {
  camera: {
    fov: 32,
    position: [0, 0.45, 10.8] as WalletVector,
    target: [0, 0.12, 0] as WalletVector,
  },
  dpr: [1, 1.5] as [number, number],
  groupScale: {
    compact: 0.72,
    standard: 0.98,
    wide: 0.98,
  },
  shadowMapSize: 512,
} as const;
