export type PipelinePoint = readonly [number, number, number];

export const pipelineBrand = {
  background: "#050505",
  black: "#040506",
  graphite: "#111216",
  graphiteEdge: "#282A30",
  nodeBlue: "#9AB7FF",
  offWhite: "#E8E7E2",
  silver: "#BFC1C6",
  silverDark: "#72747B",
  warmMetal: "#A16D53",
} as const;

export const pipelineCamera = {
  dpr: [1, 1.5] as [number, number],
  position: [0, 0, 12] as PipelinePoint,
  zoom: 98,
} as const;

// These paths are traced from the visible centerlines in the 735 x 420 reference.
export const pipelinePaths = {
  rearLoop: [
    [-3.45, 0.78, -1.4],
    [-3.6, 1.38, -1.4],
    [-3.26, 2.12, -1.4],
    [-2.38, 2.2, -1.4],
    [-1.3, 1.82, -1.4],
    [-0.18, 1.93, -1.4],
    [2.2, 2.78, -1.4],
  ],
  rearDrop: [
    [-3.42, 0.88, -1.55],
    [-2.96, 0.25, -1.55],
    [-2.22, -0.56, -1.55],
    [-1.28, -1.3, -1.55],
    [-0.18, -1.98, -1.55],
  ],
  upperRail: [
    [-5.2, -0.68, -0.55],
    [-3.58, -0.05, -0.55],
    [-1.84, 0.52, -0.55],
    [-0.56, 0.48, -0.55],
    [0.66, 0.78, -0.55],
    [2.18, 1.46, -0.55],
    [3.95, 2.18, -0.55],
  ],
  middleRail: [
    [-5.18, -1.72, -0.2],
    [-3.62, -1.03, -0.2],
    [-1.7, -0.37, -0.2],
    [-0.42, -0.16, -0.2],
    [0.76, 0.07, -0.2],
    [2.1, 0.63, -0.2],
    [3.98, 1.68, -0.2],
  ],
  lowerRail: [
    [-5.05, -2.35, -0.9],
    [-3.48, -1.65, -0.9],
    [-1.48, -1.06, -0.9],
    [-0.18, -0.94, -0.9],
    [1.1, -1.03, -0.9],
    [2.58, -1.45, -0.9],
    [4.08, -2.05, -0.9],
  ],
} satisfies Record<string, readonly PipelinePoint[]>;

export const carrierSegments = [
  {
    color: pipelineBrand.offWhite,
    end: [-1.95, -1.18, 1.15] as PipelinePoint,
    metalness: 0.58,
    radius: 0.205,
    roughness: 0.23,
    start: [-3.34, -2.1, 1.15] as PipelinePoint,
  },
  {
    color: pipelineBrand.graphite,
    end: [0.82, 0.68, 1.15] as PipelinePoint,
    metalness: 0.68,
    radius: 0.198,
    roughness: 0.19,
    start: [-1.94, -1.17, 1.15] as PipelinePoint,
  },
  {
    color: pipelineBrand.silver,
    end: [3.56, 2.47, 1.15] as PipelinePoint,
    metalness: 0.9,
    radius: 0.21,
    roughness: 0.14,
    start: [0.81, 0.67, 1.15] as PipelinePoint,
  },
] as const;

export const supportSegments = [
  {
    end: [-2.42, -1.23, -1.05] as PipelinePoint,
    start: [-2.42, 0.22, -1.05] as PipelinePoint,
  },
  {
    end: [-0.84, -1.0, -1.05] as PipelinePoint,
    start: [-0.84, 0.58, -1.05] as PipelinePoint,
  },
  {
    end: [1.18, -1.18, -1.05] as PipelinePoint,
    start: [1.18, 0.78, -1.05] as PipelinePoint,
  },
  {
    end: [2.65, -1.52, -1.05] as PipelinePoint,
    start: [2.65, 1.13, -1.05] as PipelinePoint,
  },
] as const;
