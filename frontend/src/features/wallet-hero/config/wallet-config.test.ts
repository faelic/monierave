import { describe, expect, it } from "vitest";

import { monieraveBrand, walletAssets, walletTimeline } from "./wallet-config";

describe("wallet hero configuration", () => {
  it("keeps the requested Monierave palette centralized", () => {
    expect(monieraveBrand).toEqual({
      monieraveDark: "#292A2E",
      monieraveForest: "#111113",
      monieraveMint: "#A7A8AD",
      monieraveOffWhite: "#F3F3F1",
      monieraveSage: "#D7D7DA",
    });
  });

  it("keeps the original Monierave motion phases in a valid order", () => {
    expect(walletTimeline.holderOpen).toBeLessThan(walletTimeline.cardReveal);
    expect(walletTimeline.cardReveal).toBeLessThan(walletTimeline.tokenLaunch);
    expect(walletTimeline.tokenLaunch).toBeLessThan(walletTimeline.heroHold);
    expect(walletTimeline.heroHold).toBeLessThan(walletTimeline.retract);
    expect(walletTimeline.retract).toBeLessThan(walletTimeline.closeComplete);
    expect(walletTimeline.closeComplete).toBeLessThan(walletTimeline.duration);
  });

  it("uses replaceable SVG card faces and a static fallback", () => {
    expect(
      Object.values(walletAssets.textures).every((path) =>
        path.endsWith(".svg"),
      ),
    ).toBe(true);
    expect(walletAssets.fallback).toMatch(/\.(png|webp|avif)$/);
  });
});
