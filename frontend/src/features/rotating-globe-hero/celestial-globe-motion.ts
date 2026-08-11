export const CELESTIAL_GLOBE_DURATION_SECONDS = 28;
export const CELESTIAL_POLE_TWINKLE_SECONDS = 2.6;

export function normalizeGlobeReviewTime(time: number) {
  if (!Number.isFinite(time)) return 0;
  const normalized = time % CELESTIAL_GLOBE_DURATION_SECONDS;
  return normalized < 0
    ? normalized + CELESTIAL_GLOBE_DURATION_SECONDS
    : normalized;
}

export function resolveGlobeReviewState(time: number) {
  const normalizedTime = normalizeGlobeReviewTime(time);
  return {
    animationDelay: `${-normalizedTime}s`,
    degrees: (normalizedTime / CELESTIAL_GLOBE_DURATION_SECONDS) * 360,
    normalizedTime,
  };
}
