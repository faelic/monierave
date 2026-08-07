import { describe, expect, it } from "vitest";

import { selectWalletCapability } from "./wallet-runtime-policy";

const capableEnvironment = {
  deviceMemory: 8,
  hardwareConcurrency: 8,
  prefersReducedMotion: false,
  saveData: false,
  viewportEligible: true,
  webglAvailable: true,
};

describe("selectWalletCapability", () => {
  it("allows the interactive scene on capable devices", () => {
    expect(selectWalletCapability(capableEnvironment)).toEqual({
      mode: "interactive",
    });
  });

  it.each([
    ["reduced-motion", { prefersReducedMotion: true }],
    ["save-data", { saveData: true }],
    ["compact-viewport", { viewportEligible: false }],
    ["low-power", { deviceMemory: 2 }],
    ["low-power", { hardwareConcurrency: 2 }],
    ["webgl-unavailable", { webglAvailable: false }],
  ] as const)("selects the %s fallback", (reason, overrides) => {
    expect(
      selectWalletCapability({ ...capableEnvironment, ...overrides }),
    ).toEqual({
      mode: "static",
      reason,
    });
  });
});
