export type VaultFallbackReason =
  | "compact-viewport"
  | "low-power"
  | "reduced-motion"
  | "save-data"
  | "webgl-unavailable";

export type VaultCapability =
  { mode: "interactive" } | { mode: "static"; reason: VaultFallbackReason };

export type VaultEnvironment = {
  deviceMemory: number | undefined;
  hardwareConcurrency: number | undefined;
  prefersReducedMotion: boolean;
  saveData: boolean;
  viewportEligible: boolean;
  webglAvailable: boolean;
};

export function selectVaultCapability(
  environment: VaultEnvironment,
): VaultCapability {
  if (environment.prefersReducedMotion) {
    return { mode: "static", reason: "reduced-motion" };
  }

  if (environment.saveData) {
    return { mode: "static", reason: "save-data" };
  }

  if (!environment.viewportEligible) {
    return { mode: "static", reason: "compact-viewport" };
  }

  if (
    (environment.deviceMemory !== undefined && environment.deviceMemory <= 2) ||
    (environment.hardwareConcurrency !== undefined &&
      environment.hardwareConcurrency <= 2)
  ) {
    return { mode: "static", reason: "low-power" };
  }

  if (!environment.webglAvailable) {
    return { mode: "static", reason: "webgl-unavailable" };
  }

  return { mode: "interactive" };
}

type NavigatorWithHints = Navigator & {
  connection?: {
    saveData?: boolean;
  };
  deviceMemory?: number;
};

function hasWebGLSupport() {
  try {
    const canvas = document.createElement("canvas");

    return Boolean(canvas.getContext("webgl2") || canvas.getContext("webgl"));
  } catch {
    return false;
  }
}

export function readVaultEnvironment(): VaultEnvironment {
  const navigatorWithHints = navigator as NavigatorWithHints;

  return {
    deviceMemory: navigatorWithHints.deviceMemory,
    hardwareConcurrency: navigator.hardwareConcurrency,
    prefersReducedMotion: window.matchMedia("(prefers-reduced-motion: reduce)")
      .matches,
    saveData: Boolean(navigatorWithHints.connection?.saveData),
    viewportEligible: window.matchMedia("(min-width: 64rem)").matches,
    webglAvailable: hasWebGLSupport(),
  };
}

export const vaultFallbackMessages: Record<VaultFallbackReason, string> = {
  "compact-viewport":
    "The lightweight version is active to preserve clarity on this screen.",
  "low-power":
    "The lightweight version is active to protect performance on this device.",
  "reduced-motion":
    "The still version is active because reduced motion is enabled.",
  "save-data":
    "The lightweight version is active because data saving is enabled.",
  "webgl-unavailable":
    "The lightweight version is active because WebGL is unavailable.",
};
