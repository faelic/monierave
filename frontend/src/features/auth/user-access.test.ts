import { describe, expect, it } from "vitest";

import type { User } from "@/lib/api/contracts";

import { canAccessApplicationPath, hasFinancialAccess } from "./user-access";

const activeUser: User = {
  account_status: "active",
  created_at: "2026-08-11T08:00:00Z",
  email: "customer@example.com",
  email_bounced_at: null,
  email_deliverability_status: "deliverable",
  email_verified_at: "2026-08-11T08:05:00Z",
  full_name: "Monierave Customer",
  password_changed_at: "2026-08-11T08:00:00Z",
  registration_expires_at: null,
  username: "customer",
};

const pendingUser: User = {
  ...activeUser,
  account_status: "pending",
  email_deliverability_status: "pending",
  email_verified_at: null,
  registration_expires_at: "2026-08-18T08:00:00Z",
};

describe("user access policy", () => {
  it("grants financial access only to active verified users", () => {
    expect(hasFinancialAccess(activeUser)).toBe(true);
    expect(hasFinancialAccess(pendingUser)).toBe(false);
    expect(hasFinancialAccess({ ...activeUser, email_verified_at: null })).toBe(
      false,
    );
  });

  it("allows pending users to use overview, profile, and security", () => {
    expect(canAccessApplicationPath(pendingUser, "/app")).toBe(true);
    expect(canAccessApplicationPath(pendingUser, "/app/profile")).toBe(true);
    expect(canAccessApplicationPath(pendingUser, "/app/profile/edit")).toBe(
      true,
    );
    expect(canAccessApplicationPath(pendingUser, "/app/security")).toBe(true);
  });

  it("keeps financial pages unavailable to pending users", () => {
    expect(canAccessApplicationPath(pendingUser, "/app/accounts")).toBe(false);
    expect(canAccessApplicationPath(pendingUser, "/app/transfers/new")).toBe(
      false,
    );
    expect(canAccessApplicationPath(pendingUser, "/app/transactions")).toBe(
      false,
    );
    expect(canAccessApplicationPath(pendingUser, "/app/beneficiaries")).toBe(
      false,
    );
  });
});
