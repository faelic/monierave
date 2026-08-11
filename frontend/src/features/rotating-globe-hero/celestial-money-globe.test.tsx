import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  CELESTIAL_GLOBE_DURATION_SECONDS,
  CELESTIAL_POLE_TWINKLE_SECONDS,
  resolveGlobeReviewState,
} from "./celestial-globe-motion";
import { CelestialMoneyGlobe } from "./celestial-money-globe";

describe("CelestialMoneyGlobe", () => {
  it("preserves the exact approved SVG geometry", () => {
    render(<CelestialMoneyGlobe paused reviewTime={0} />);
    const globe = screen.getByTestId("celestial-money-globe");
    const sphere = screen.getByTestId("celestial-globe-sphere");
    const axis = screen.getByTestId("celestial-globe-axis");
    const circle = sphere.querySelector("circle");
    const ellipses = [...sphere.querySelectorAll("ellipse")];
    const polygons = globe.querySelectorAll("polygon");

    expect(globe).toHaveAttribute("viewBox", "0 0 200 200");
    expect(circle).toHaveAttribute("cx", "100");
    expect(circle).toHaveAttribute("cy", "100");
    expect(circle).toHaveAttribute("r", "78");
    expect(ellipses).toHaveLength(6);
    expect(
      ellipses.map((ellipse) => [
        ellipse.getAttribute("rx"),
        ellipse.getAttribute("ry"),
      ]),
    ).toEqual([
      ["78", "22"],
      ["78", "46"],
      ["78", "66"],
      ["22", "78"],
      ["46", "78"],
      ["66", "78"],
    ]);
    expect(axis).toHaveAttribute("x1", "100");
    expect(axis).toHaveAttribute("x2", "100");
    expect(axis).toHaveAttribute("y1", "14");
    expect(axis).toHaveAttribute("y2", "186");
    expect(polygons).toHaveLength(2);
    expect(polygons[0]).toHaveAttribute("points", "100,6 103,14 100,22 97,14");
    expect(polygons[1]).toHaveAttribute(
      "points",
      "100,178 103,186 100,194 97,186",
    );
    expect(sphere).not.toContainElement(axis);
  });

  it("maps deterministic review times onto exact quarter turns", () => {
    expect(CELESTIAL_GLOBE_DURATION_SECONDS).toBe(28);
    expect(CELESTIAL_POLE_TWINKLE_SECONDS).toBe(2.6);
    expect(resolveGlobeReviewState(0)).toMatchObject({
      animationDelay: "0s",
      degrees: 0,
      normalizedTime: 0,
    });
    expect(resolveGlobeReviewState(7).degrees).toBe(90);
    expect(resolveGlobeReviewState(14).degrees).toBe(180);
    expect(resolveGlobeReviewState(21).degrees).toBe(270);
    expect(resolveGlobeReviewState(28).degrees).toBe(0);
    expect(resolveGlobeReviewState(-7).degrees).toBe(270);
  });

  it("exposes deterministic and reduced-motion states", () => {
    const { rerender } = render(<CelestialMoneyGlobe paused reviewTime={7} />);
    const globe = screen.getByTestId("celestial-money-globe");
    const sphere = screen.getByTestId("celestial-globe-sphere");

    expect(globe).toHaveAttribute("data-globe-review-time", "7");
    expect(globe).toHaveAttribute("data-globe-review-degrees", "90");
    expect(sphere).toHaveStyle({
      animationDelay: "-7s",
      animationPlayState: "paused",
    });

    rerender(<CelestialMoneyGlobe reducedMotion />);
    expect(globe).toHaveAttribute("data-globe-motion", "static");
    expect(sphere).toHaveStyle({ animationPlayState: "paused" });
  });
});
