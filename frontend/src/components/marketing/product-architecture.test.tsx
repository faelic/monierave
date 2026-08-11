import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ProductArchitecture } from "./product-architecture";

describe("ProductArchitecture", () => {
  it("maps the supported Monierave workspace capabilities", () => {
    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      value: () => ({
        addEventListener: () => undefined,
        matches: false,
        removeEventListener: () => undefined,
      }),
    });

    const { container } = render(<ProductArchitecture />);

    expect(
      screen.getByRole("heading", {
        level: 2,
        name: "Everything important, connected.",
      }),
    ).toBeInTheDocument();

    const nodeTitles = screen
      .getAllByRole("heading", { level: 3 })
      .map((heading) => heading.textContent);
    expect(nodeTitles).toEqual([
      "Verified access",
      "Dashboard overview",
      "Accounts",
      "Transfers",
      "Beneficiaries",
      "Posted activity",
      "Profile & security",
    ]);

    expect(container.querySelectorAll("[data-node-stage]")).toHaveLength(7);
  });
});
