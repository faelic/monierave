import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Button } from "@/components/ui/button";

describe("Button", () => {
  it("supports keyboard activation", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Continue</Button>);

    await user.tab();
    await user.keyboard("{Enter}");

    expect(onClick).toHaveBeenCalledOnce();
  });

  it("prevents duplicate activation while loading", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(
      <Button loading onClick={onClick}>
        Sending transfer
      </Button>,
    );

    const button = screen.getByRole("button", { name: /sending transfer/i });
    expect(button).toBeDisabled();
    await user.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });

  it("preserves link semantics when used for navigation", () => {
    render(
      <Button asChild>
        <a href="/security">Security approach</a>
      </Button>,
    );

    const link = screen.getByRole("link", { name: "Security approach" });
    expect(link).toHaveAttribute("href", "/security");
    expect(link).toHaveClass("bg-evergreen-800");
  });
});
