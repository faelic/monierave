import type { CSSProperties } from "react";

import {
  CELESTIAL_GLOBE_DURATION_SECONDS,
  CELESTIAL_POLE_TWINKLE_SECONDS,
  resolveGlobeReviewState,
} from "./celestial-globe-motion";
import styles from "./rotating-globe-hero.module.css";

type CelestialMoneyGlobeProps = {
  className?: string | undefined;
  paused?: boolean;
  reducedMotion?: boolean;
  reviewTime?: number | null;
};

export function CelestialMoneyGlobe({
  className,
  paused = false,
  reducedMotion = false,
  reviewTime = null,
}: CelestialMoneyGlobeProps) {
  const reviewState =
    reviewTime === null ? null : resolveGlobeReviewState(reviewTime);
  const freezeMotion = paused || reducedMotion || reviewState !== null;
  const sphereStyle = {
    "--globe-duration": `${CELESTIAL_GLOBE_DURATION_SECONDS}s`,
    ...(reviewState
      ? { animationDelay: reviewState.animationDelay }
      : undefined),
    animationPlayState: freezeMotion ? "paused" : "running",
  } as CSSProperties;
  const starStyle = {
    "--pole-duration": `${CELESTIAL_POLE_TWINKLE_SECONDS}s`,
    animationPlayState: freezeMotion ? "paused" : "running",
  } as CSSProperties;

  return (
    <svg
      aria-hidden="true"
      className={className}
      data-globe-motion={reducedMotion ? "static" : "active"}
      data-globe-review-degrees={reviewState?.degrees ?? "live"}
      data-globe-review-time={reviewState?.normalizedTime ?? "live"}
      data-testid="celestial-money-globe"
      focusable="false"
      viewBox="0 0 200 200"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g>
        <g
          className={styles.sphere}
          data-testid="celestial-globe-sphere"
          style={sphereStyle}
        >
          <circle className={styles.globeStroke} cx="100" cy="100" r="78" />
          {[22, 46, 66].map((radius) => (
            <ellipse
              className={styles.globeThinStroke}
              cx="100"
              cy="100"
              key={`horizontal-${radius}`}
              rx="78"
              ry={radius}
            />
          ))}
          {[22, 46, 66].map((radius) => (
            <ellipse
              className={styles.globeThinStroke}
              cx="100"
              cy="100"
              key={`vertical-${radius}`}
              rx={radius}
              ry="78"
            />
          ))}
        </g>
        <line
          className={styles.globeStroke}
          data-testid="celestial-globe-axis"
          x1="100"
          x2="100"
          y1="14"
          y2="186"
        />
        <g className={styles.poleStar} style={starStyle}>
          <polygon
            className={styles.poleFill}
            points="100,6 103,14 100,22 97,14"
          />
        </g>
        <polygon
          className={styles.poleFill}
          points="100,178 103,186 100,194 97,186"
        />
      </g>
    </svg>
  );
}
