import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { MoneyMovementChart } from "@/features/dashboard/money-movement-chart";
import type { MoneyMovement } from "@/lib/api/contracts";

const movement: MoneyMovement = {
  account_id: "account-id",
  buckets: [
    {
      incoming: 12_500,
      outgoing: 3_000,
      start: "2026-08-10T00:00:00Z",
    },
  ],
  currency: "USD",
  from: "2026-08-10T00:00:00Z",
  interval: "day",
  money_in: 12_500,
  money_out: 3_000,
  to: "2026-08-11T00:00:00Z",
};

describe("MoneyMovementChart", () => {
  it("exposes a readable chart and equivalent tabular values", () => {
    render(<MoneyMovementChart data={movement} />);

    expect(
      screen.getByRole("img", {
        name: /posted money in and money out for the selected USD account/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("USD 125.00")).toBeInTheDocument();
    expect(screen.getByText("USD 30.00")).toBeInTheDocument();
  });

  it("reveals exact movement details through keyboard focus", async () => {
    const user = userEvent.setup();
    render(<MoneyMovementChart data={movement} />);

    await user.tab();

    expect(screen.getByText("Net")).toBeInTheDocument();
    expect(screen.getByText("+USD 95.00")).toBeInTheDocument();
    expect(
      screen.getByText(
        /All movement in this period occurred on (10 Aug|Aug 10)\./,
      ),
    ).toBeInTheDocument();
  });

  it("shows a truthful empty state", () => {
    render(<MoneyMovementChart data={{ ...movement, buckets: [] }} />);

    expect(
      screen.getByText("No posted movement in this period."),
    ).toBeInTheDocument();
  });
});
