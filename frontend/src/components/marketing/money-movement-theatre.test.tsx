import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { MoneyMovementTheatre } from "./money-movement-theatre";

describe("MoneyMovementTheatre", () => {
  it("starts with posted balances and switches views without hiding their controls", () => {
    render(<MoneyMovementTheatre />);

    expect(screen.getByText("Current posted balances")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: /money movement/i }));
    expect(screen.getByText("See the value move")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: /transfer records/i }));
    expect(screen.getByText("One record, every state")).toBeInTheDocument();
  });

  it("shows truthful posted and failed record outcomes", () => {
    render(<MoneyMovementTheatre />);
    fireEvent.click(screen.getByRole("tab", { name: /transfer records/i }));

    expect(
      screen.getByText(/both ledger postings completed/i),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Failed" }));
    expect(screen.getByText(/no money moved/i)).toBeInTheDocument();
  });

  it("stops automatic cycling after manual selection", () => {
    vi.useFakeTimers();
    render(<MoneyMovementTheatre />);

    fireEvent.click(screen.getByRole("tab", { name: /money movement/i }));
    vi.advanceTimersByTime(20_000);

    expect(screen.getByText("See the value move")).toBeInTheDocument();
    vi.useRealTimers();
  });
});
