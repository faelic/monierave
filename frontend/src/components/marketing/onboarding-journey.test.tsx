import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OnboardingJourney } from "./onboarding-journey";

describe("OnboardingJourney", () => {
  it("presents the supported onboarding sequence in order", () => {
    const { container } = render(<OnboardingJourney />);

    expect(
      screen.getByRole("heading", {
        level: 2,
        name: "From sign-up to your first transfer.",
      }),
    ).toBeInTheDocument();

    const stepTitles = screen
      .getAllByRole("heading", { level: 3 })
      .map((heading) => heading.textContent);
    expect(stepTitles).toEqual([
      "Create your profile",
      "Verify your email",
      "Make a transfer",
    ]);

    const motionTypes = Array.from(
      container.querySelectorAll("[data-step-motion]"),
      (step) => step.getAttribute("data-step-motion"),
    );
    expect(motionTypes).toEqual(["account", "verification", "transfer"]);
    expect(
      container.querySelector('[data-motion-part="profile-scan"]'),
    ).toBeInTheDocument();
    expect(
      container.querySelector('[data-motion-part="profile-node"]'),
    ).toBeInTheDocument();
    expect(
      container.querySelector('[data-motion-part="verification-rings"]'),
    ).toBeInTheDocument();
    expect(
      container.querySelector('[data-motion-part="verification-sweep"]'),
    ).toBeInTheDocument();
  });
});
