import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SectionReveal } from "./section-reveal";

describe("SectionReveal", () => {
  it("keeps content available and reveals it after motion initializes", async () => {
    const { container } = render(
      <SectionReveal immediate>
        <h2>Visible product section</h2>
      </SectionReveal>,
    );

    expect(
      screen.getByRole("heading", { name: "Visible product section" }),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(container.querySelector("[data-reveal-visible]")).toHaveAttribute(
        "data-reveal-visible",
        "true",
      ),
    );
  });
});
