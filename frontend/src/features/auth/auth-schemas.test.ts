import { describe, expect, it } from "vitest";

import {
  emailUpdateSchema,
  fullNameSchema,
  passwordSchema,
  safeReturnPath,
  signupSchema,
} from "@/features/auth/auth-schemas";

describe("authentication validation", () => {
  it("enforces the backend password boundaries by UTF-8 bytes", () => {
    expect(passwordSchema.safeParse("12345678").success).toBe(true);
    expect(passwordSchema.safeParse("1234567").success).toBe(false);
    expect(passwordSchema.safeParse("a".repeat(72)).success).toBe(true);
    expect(passwordSchema.safeParse("a".repeat(73)).success).toBe(false);
    expect(passwordSchema.safeParse("£".repeat(36)).success).toBe(true);
    expect(passwordSchema.safeParse("£".repeat(37)).success).toBe(false);
  });

  it("keeps the upper-limit error user friendly", () => {
    const result = passwordSchema.safeParse("a".repeat(73));

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0]?.message).toBe(
        "This password is too long. Use fewer characters.",
      );
    }
  });

  it("trims profile fields and normalizes email addresses", () => {
    expect(fullNameSchema.parse("  Favour Ututu  ")).toBe("Favour Ututu");
    expect(
      signupSchema.parse({
        email: "  USER@Example.COM ",
        full_name: "  Favour Ututu ",
        password: "not-breached-locally",
        username: "favour22",
      }),
    ).toMatchObject({
      email: "user@example.com",
      full_name: "Favour Ututu",
    });
  });

  it("requires reauthentication for email changes", () => {
    expect(
      emailUpdateSchema.safeParse({
        current_password: "current-password",
        email: "new@example.com",
      }).success,
    ).toBe(true);
    expect(
      emailUpdateSchema.safeParse({
        current_password: "",
        email: "new@example.com",
      }).success,
    ).toBe(false);
  });
});

describe("safeReturnPath", () => {
  it.each([
    ["/app/accounts?currency=USD", "/app/accounts?currency=USD"],
    ["/app#activity", "/app#activity"],
    ["https://attacker.example", null],
    ["//attacker.example/path", null],
    ["/\\attacker.example", null],
    ["javascript:alert(1)", null],
    ["", null],
  ])("normalizes %s", (value, expected) => {
    expect(safeReturnPath(value)).toBe(expected);
  });
});
