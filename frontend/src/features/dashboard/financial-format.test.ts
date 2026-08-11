import { describe, expect, it } from "vitest";

import {
  accountLabel,
  formatMinorAmount,
  formatTransactionAmount,
} from "@/features/dashboard/financial-format";

describe("financial display formatting", () => {
  it("formats backend integer values as currency minor units", () => {
    expect(formatMinorAmount(125050, "USD", "en-US")).toBe("USD 1,250.50");
    expect(formatMinorAmount(-4250, "EUR", "en-US")).toBe("-EUR 42.50");
  });

  it("adds a semantic direction sign without changing the amount", () => {
    expect(
      formatTransactionAmount(
        { amount: 5000, currency: "USD", direction: "incoming" },
        "en-US",
      ),
    ).toBe("+USD 50.00");
    expect(
      formatTransactionAmount(
        { amount: 5000, currency: "USD", direction: "outgoing" },
        "en-US",
      ),
    ).toBe("−USD 50.00");
  });

  it("preserves account lifecycle context", () => {
    expect(accountLabel({ currency: "EUR", status: "closed" })).toBe(
      "Closed EUR account",
    );
  });
});
