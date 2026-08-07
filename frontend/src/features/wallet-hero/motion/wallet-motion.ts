import {
  AnimationClip,
  CatmullRomCurve3,
  Euler,
  Quaternion,
  QuaternionKeyframeTrack,
  Vector3,
  VectorKeyframeTrack,
} from "three";

import { walletTimeline, type WalletVector } from "../config/wallet-config";

type Pose = { time: number; value: WalletVector };

const names = {
  backPanel: "backPanelPivot",
  frontPanel: "frontPanelPivot",
  primaryCard: "primaryCardPivot",
  secondaryCard: "secondaryCardPivot",
  supportingCard: "supportingCardPivot",
  tokenA: "paymentTokenA",
  tokenB: "paymentTokenB",
  wallet: "walletMotionRoot",
} as const;

const rootPosition: Pose[] = [
  { time: 0, value: [0, -0.12, 0] },
  { time: 0.75, value: [0, -0.04, 0] },
  { time: 1.8, value: [0, 0.04, 0] },
  { time: 4.25, value: [0, 0.04, 0] },
  { time: 5.55, value: [0, -0.04, 0] },
  { time: walletTimeline.duration, value: [0, -0.12, 0] },
];

const rootRotation: Pose[] = [
  { time: 0, value: [-0.04, -0.34, -0.045] },
  { time: 0.75, value: [-0.015, -0.29, -0.025] },
  { time: 1.8, value: [0.035, -0.2, 0] },
  { time: 4.25, value: [0.035, -0.2, 0] },
  { time: 5.55, value: [-0.015, -0.29, -0.025] },
  { time: walletTimeline.duration, value: [-0.04, -0.34, -0.045] },
];

const backPanelRotation: Pose[] = [
  { time: 0, value: [0, 0, 0] },
  { time: 0.8, value: [0, 0, 0] },
  { time: 1.6, value: [-0.62, 0, 0] },
  { time: 4.5, value: [-0.62, 0, 0] },
  { time: 5.4, value: [0, 0, 0] },
  { time: walletTimeline.duration, value: [0, 0, 0] },
];

const frontPanelRotation: Pose[] = [
  { time: 0, value: [0, 0, 0] },
  { time: 0.72, value: [0, 0, 0] },
  { time: 1.52, value: [0.88, 0, 0] },
  { time: 4.42, value: [0.88, 0, 0] },
  { time: 5.3, value: [0, 0, 0] },
  { time: walletTimeline.duration, value: [0, 0, 0] },
];

const cardPositions = {
  primary: [
    { time: 0, value: [0, -0.55, 0.08] },
    { time: 1.05, value: [0, -0.55, 0.08] },
    { time: 1.75, value: [-0.36, 0.52, 0.5] },
    { time: 2.2, value: [-0.55, 0.92, 0.58] },
    { time: 4.35, value: [-0.55, 0.92, 0.58] },
    { time: 5.05, value: [0, -0.55, 0.08] },
    { time: walletTimeline.duration, value: [0, -0.55, 0.08] },
  ],
  secondary: [
    { time: 0, value: [0, -0.55, -0.08] },
    { time: 1.18, value: [0, -0.55, -0.08] },
    { time: 1.9, value: [0.02, 0.82, 0.22] },
    { time: 2.3, value: [0.04, 1.28, 0.26] },
    { time: 4.25, value: [0.04, 1.28, 0.26] },
    { time: 5.12, value: [0, -0.55, -0.08] },
    { time: walletTimeline.duration, value: [0, -0.55, -0.08] },
  ],
  supporting: [
    { time: 0, value: [0, -0.55, -0.22] },
    { time: 1.3, value: [0, -0.55, -0.22] },
    { time: 2, value: [0.36, 1.02, -0.02] },
    { time: 2.4, value: [0.54, 1.62, -0.06] },
    { time: 4.18, value: [0.54, 1.62, -0.06] },
    { time: 5.18, value: [0, -0.55, -0.22] },
    { time: walletTimeline.duration, value: [0, -0.55, -0.22] },
  ],
} satisfies Record<string, Pose[]>;

const cardScales = {
  primary: [
    { time: 0, value: [0.94, 0.04, 0.94] },
    { time: 1.05, value: [0.94, 0.04, 0.94] },
    { time: 1.62, value: [1, 1, 1] },
    { time: 4.72, value: [1, 1, 1] },
    { time: 5.05, value: [0.94, 0.04, 0.94] },
    { time: walletTimeline.duration, value: [0.94, 0.04, 0.94] },
  ],
  secondary: [
    { time: 0, value: [0.94, 0.04, 0.94] },
    { time: 1.18, value: [0.94, 0.04, 0.94] },
    { time: 1.76, value: [1, 1, 1] },
    { time: 4.65, value: [1, 1, 1] },
    { time: 5.12, value: [0.94, 0.04, 0.94] },
    { time: walletTimeline.duration, value: [0.94, 0.04, 0.94] },
  ],
  supporting: [
    { time: 0, value: [0.94, 0.04, 0.94] },
    { time: 1.3, value: [0.94, 0.04, 0.94] },
    { time: 1.9, value: [1, 1, 1] },
    { time: 4.58, value: [1, 1, 1] },
    { time: 5.18, value: [0.94, 0.04, 0.94] },
    { time: walletTimeline.duration, value: [0.94, 0.04, 0.94] },
  ],
} satisfies Record<string, Pose[]>;

const cardRotations = {
  primary: [
    { time: 0, value: [0, 0, 0] },
    { time: 1.05, value: [0, 0, 0] },
    { time: 2.2, value: [-0.04, -0.08, -0.1] },
    { time: 4.35, value: [-0.04, -0.08, -0.1] },
    { time: 5.05, value: [0, 0, 0] },
    { time: walletTimeline.duration, value: [0, 0, 0] },
  ],
  secondary: [
    { time: 0, value: [0, 0, 0] },
    { time: 1.18, value: [0, 0, 0] },
    { time: 2.3, value: [-0.02, 0.02, 0.015] },
    { time: 4.25, value: [-0.02, 0.02, 0.015] },
    { time: 5.12, value: [0, 0, 0] },
    { time: walletTimeline.duration, value: [0, 0, 0] },
  ],
  supporting: [
    { time: 0, value: [0, 0, 0] },
    { time: 1.3, value: [0, 0, 0] },
    { time: 2.4, value: [0.02, 0.08, 0.11] },
    { time: 4.18, value: [0.02, 0.08, 0.11] },
    { time: 5.18, value: [0, 0, 0] },
    { time: walletTimeline.duration, value: [0, 0, 0] },
  ],
} satisfies Record<string, Pose[]>;

const tokenCurves = [
  new CatmullRomCurve3([
    new Vector3(-0.4, -0.35, 0.9),
    new Vector3(-1.65, 0.55, 1.1),
    new Vector3(-1.3, 2.7, 0.9),
    new Vector3(0.2, 3.2, 0.72),
    new Vector3(1.65, 2.35, 0.85),
    new Vector3(1.95, 0.45, 1),
    new Vector3(0.45, -0.35, 0.9),
  ]),
  new CatmullRomCurve3([
    new Vector3(0.3, -0.35, 0.82),
    new Vector3(1.45, 0.2, 0.95),
    new Vector3(2.05, 1.45, 0.86),
    new Vector3(1.15, 2.85, 0.72),
    new Vector3(-0.5, 2.45, 0.88),
    new Vector3(-1.85, 1.2, 1.04),
    new Vector3(-0.35, -0.35, 0.82),
  ]),
] as const;

function smoothstep(value: number) {
  const clamped = Math.min(1, Math.max(0, value));
  return clamped * clamped * (3 - 2 * clamped);
}

function sampleTimes() {
  const frameStep = 1 / 30;
  const times: number[] = [];
  for (let time = 0; time < walletTimeline.duration; time += frameStep) {
    times.push(time);
  }
  times.push(walletTimeline.duration);
  return times;
}

function findSegment(poses: Pose[], time: number) {
  for (let index = 0; index < poses.length - 1; index += 1) {
    const current = poses[index]!;
    const next = poses[index + 1]!;
    if (time >= current.time && time <= next.time) {
      return { current, next };
    }
  }
  const last = poses.at(-1)!;
  return { current: last, next: last };
}

function sampleVector(poses: Pose[], time: number) {
  const { current, next } = findSegment(poses, time);
  if (current.time === next.time) {
    return new Vector3(...current.value);
  }
  const progress = smoothstep(
    (time - current.time) / (next.time - current.time),
  );
  return new Vector3(...current.value).lerp(
    new Vector3(...next.value),
    progress,
  );
}

function sampleQuaternion(poses: Pose[], time: number) {
  const { current, next } = findSegment(poses, time);
  const from = new Quaternion().setFromEuler(
    new Euler(...current.value, "XYZ"),
  );
  if (current.time === next.time) {
    return from;
  }
  const progress = smoothstep(
    (time - current.time) / (next.time - current.time),
  );
  return from.slerp(
    new Quaternion().setFromEuler(new Euler(...next.value, "XYZ")),
    progress,
  );
}

function vectorTrack(
  name: string,
  poses: Pose[],
  property: "position" | "scale" = "position",
) {
  const times = sampleTimes();
  return new VectorKeyframeTrack(
    `${name}.${property}`,
    times,
    times.flatMap((time) => sampleVector(poses, time).toArray()),
  );
}

function quaternionTrack(name: string, poses: Pose[]) {
  const times = sampleTimes();
  return new QuaternionKeyframeTrack(
    `${name}.quaternion`,
    times,
    times.flatMap((time) => sampleQuaternion(poses, time).toArray()),
  );
}

function tokenTracks(name: string, index: 0 | 1) {
  const times = sampleTimes();
  const start = index === 0 ? 1.7 : 1.95;
  const end = index === 0 ? 4.35 : 4.5;
  const positions: number[] = [];
  const quaternions: number[] = [];
  const scales: number[] = [];

  for (const time of times) {
    const progress = smoothstep((time - start) / (end - start));
    const point = tokenCurves[index].getPointAt(progress);
    positions.push(...point.toArray());

    const rotation = new Quaternion().setFromEuler(
      new Euler(
        0.5 + progress * Math.PI * (index === 0 ? 1.8 : -1.6),
        Math.sin(progress * Math.PI) * 0.5,
        progress * Math.PI * (index === 0 ? 1.2 : -1.1),
      ),
    );
    quaternions.push(...rotation.toArray());

    const reveal = smoothstep((time - start) / 0.16);
    const hide = smoothstep((end - time) / 0.16);
    const scale = Math.min(reveal, hide);
    scales.push(scale, scale, scale);
  }

  return [
    new VectorKeyframeTrack(`${name}.position`, times, positions),
    new QuaternionKeyframeTrack(`${name}.quaternion`, times, quaternions),
    new VectorKeyframeTrack(`${name}.scale`, times, scales),
  ];
}

export function createWalletAnimationClip() {
  return new AnimationClip("monieraveWallet", walletTimeline.duration, [
    vectorTrack(names.wallet, rootPosition),
    quaternionTrack(names.wallet, rootRotation),
    quaternionTrack(names.backPanel, backPanelRotation),
    quaternionTrack(names.frontPanel, frontPanelRotation),
    vectorTrack(names.primaryCard, cardPositions.primary),
    vectorTrack(names.primaryCard, cardScales.primary, "scale"),
    quaternionTrack(names.primaryCard, cardRotations.primary),
    vectorTrack(names.secondaryCard, cardPositions.secondary),
    vectorTrack(names.secondaryCard, cardScales.secondary, "scale"),
    quaternionTrack(names.secondaryCard, cardRotations.secondary),
    vectorTrack(names.supportingCard, cardPositions.supporting),
    vectorTrack(names.supportingCard, cardScales.supporting, "scale"),
    quaternionTrack(names.supportingCard, cardRotations.supporting),
    ...tokenTracks(names.tokenA, 0),
    ...tokenTracks(names.tokenB, 1),
  ]);
}
