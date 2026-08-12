import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { CurrencyIdentity } from "@/features/dashboard/currency-identity";

describe("CurrencyIdentity", () => {
  it("labels USD as US Dollar while keeping the flag decorative", () => {
    render(<CurrencyIdentity currency="USD" />);

    expect(screen.getByText("USD")).toBeInTheDocument();
    expect(screen.getByText("· US Dollar")).toBeInTheDocument();
    expect(document.querySelector("svg")?.parentElement).toHaveAttribute(
      "aria-hidden",
      "true",
    );
  });

  it("uses the Euro region identity for EUR", () => {
    render(<CurrencyIdentity currency="EUR" />);

    expect(screen.getByText("EUR")).toBeInTheDocument();
    expect(screen.getByText("· Euro")).toBeInTheDocument();
  });
});
