import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { IlluminatedClosingCta } from "./illuminated-closing-cta";

vi.mock("@/features/auth/auth-provider", () => ({
  useAuth: () => ({ status: "anonymous", user: null }),
}));

describe("IlluminatedClosingCta", () => {
  it("presents the final conversion actions without an oversized wordmark", () => {
    const { container } = render(<IlluminatedClosingCta />);

    expect(
      screen.getByRole("heading", {
        name: "Ready to get started?",
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/create your profile, verify your email/i),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /get started/i })).toHaveAttribute(
      "href",
      "/signup",
    );
    expect(screen.getByRole("link", { name: "Sign in" })).toHaveAttribute(
      "href",
      "/login",
    );
    expect(container.querySelector("svg text")).not.toBeInTheDocument();
    expect(
      container.querySelector("[data-wordmark-lit]"),
    ).not.toBeInTheDocument();
  });
});
