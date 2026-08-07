import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { PasswordInput } from "@/components/auth/password-input";

describe("PasswordInput", () => {
  it("reveals and hides the value without moving keyboard focus", async () => {
    const user = userEvent.setup();
    render(<PasswordInput aria-label="Password" defaultValue="secret-value" />);

    const input = screen.getByLabelText("Password");
    const reveal = screen.getByRole("button", { name: "Show password" });
    expect(input).toHaveAttribute("type", "password");

    await user.click(reveal);
    expect(input).toHaveAttribute("type", "text");
    expect(reveal).toHaveFocus();

    await user.keyboard("{Enter}");
    expect(input).toHaveAttribute("type", "password");
    expect(reveal).toHaveFocus();
  });
});
