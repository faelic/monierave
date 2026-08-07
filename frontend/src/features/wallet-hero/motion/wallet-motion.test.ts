import { describe, expect, it } from "vitest";

import { walletTimeline } from "../config/wallet-config";
import { createWalletAnimationClip } from "./wallet-motion";

function endpointValues(
  clip: ReturnType<typeof createWalletAnimationClip>,
  name: string,
) {
  const track = clip.tracks.find((candidate) => candidate.name === name);
  if (!track) {
    throw new Error(`missing animation track: ${name}`);
  }
  const size = track.getValueSize();
  return {
    end: Array.from(track.values.slice(-size) as ArrayLike<number>),
    start: Array.from(track.values.slice(0, size) as ArrayLike<number>),
  };
}

describe("Monierave wallet animation", () => {
  it("uses one deterministic clip with independent scene objects", () => {
    const clip = createWalletAnimationClip();
    const trackNames = new Set(clip.tracks.map((track) => track.name));

    expect(clip.duration).toBe(walletTimeline.duration);
    for (const name of [
      "walletMotionRoot.position",
      "walletMotionRoot.quaternion",
      "backPanelPivot.quaternion",
      "frontPanelPivot.quaternion",
      "primaryCardPivot.position",
      "secondaryCardPivot.position",
      "supportingCardPivot.position",
      "paymentTokenA.position",
      "paymentTokenB.position",
    ]) {
      expect(trackNames).toContain(name);
    }
  });

  it("returns every persistent object to its starting transform", () => {
    const clip = createWalletAnimationClip();

    for (const name of [
      "walletMotionRoot.position",
      "walletMotionRoot.quaternion",
      "backPanelPivot.quaternion",
      "frontPanelPivot.quaternion",
      "primaryCardPivot.position",
      "primaryCardPivot.quaternion",
      "primaryCardPivot.scale",
      "secondaryCardPivot.position",
      "secondaryCardPivot.quaternion",
      "secondaryCardPivot.scale",
      "supportingCardPivot.position",
      "supportingCardPivot.quaternion",
      "supportingCardPivot.scale",
    ]) {
      const values = endpointValues(clip, name);
      expect(values.end).toEqual(values.start);
    }
  });

  it("keeps transaction tokens hidden at both loop boundaries", () => {
    const clip = createWalletAnimationClip();

    for (const name of ["paymentTokenA.scale", "paymentTokenB.scale"]) {
      const values = endpointValues(clip, name);
      expect(values.start).toEqual([0, 0, 0]);
      expect(values.end).toEqual([0, 0, 0]);
    }
  });
});
