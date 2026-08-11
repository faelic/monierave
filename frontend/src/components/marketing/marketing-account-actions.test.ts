import { describe, expect, it } from "vitest";

import { marketingAccountActions } from "./marketing-account-actions";
import type { User } from "@/lib/api/contracts";

const user: User = {
  account_status: "active",
  created_at: "2026-08-01T09:00:00Z",
  email: "favour@example.com",
  email_bounced_at: null,
  email_deliverability_status: "deliverable",
  email_verified_at: "2026-08-01T09:05:00Z",
  full_name: "Favour Ututu",
  password_changed_at: "2026-08-01T09:00:00Z",
  registration_expires_at: null,
  username: "favour22",
};

describe("marketing account actions", () => {
  it("does not flash anonymous actions while restoring a session", () => {
    expect(marketingAccountActions("restoring", null)).toEqual({
      loading: true,
    });
  });

  it("sends anonymous visitors to authentication", () => {
    expect(marketingAccountActions("anonymous", null)).toMatchObject({
      primary: { href: "/signup", label: "Get started" },
      secondary: { href: "/login", label: "Sign in" },
    });
  });

  it.each([
    ["active", "Open dashboard"],
    ["pending", "Continue verification"],
    ["disabled", "Review account"],
  ] as const)("uses the correct %s account action", (accountStatus, label) => {
    expect(
      marketingAccountActions("authenticated", {
        ...user,
        account_status: accountStatus,
      }).primary,
    ).toEqual({ href: "/app", label });
  });
});
