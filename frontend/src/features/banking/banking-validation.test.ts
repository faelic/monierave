import { describe, expect, it } from "vitest";

import {
  accountNumberSchema,
  majorAmountError,
  nicknameSchema,
  normalizeAccountNumberInput,
  normalizeMajorAmountInput,
  parseMajorAmount,
} from "@/features/banking/banking-validation";

describe("banking input validation", () => {
  it("converts major units to integer minor units without floating point math", () => {
    expect(parseMajorAmount("10")).toBe(1000);
    expect(parseMajorAmount("10.5")).toBe(1050);
    expect(parseMajorAmount("10.05")).toBe(1005);
    expect(parseMajorAmount("0.01")).toBe(1);
  });

  it.each(["0", "-1", "1.001", "1e3", "1,000", "not-money"])(
    "rejects unsafe amount %s",
    (value) => {
      expect(parseMajorAmount(value)).toBeNull();
    },
  );

  it("enforces account-number and Unicode nickname boundaries", () => {
    expect(accountNumberSchema.safeParse("4839201756").success).toBe(true);
    expect(accountNumberSchema.safeParse("48392").success).toBe(false);
    expect(nicknameSchema.safeParse("É".repeat(50)).success).toBe(true);
    expect(nicknameSchema.safeParse("É".repeat(51)).success).toBe(false);
  });

  it("keeps account numbers numeric without losing leading zeroes", () => {
    expect(normalizeAccountNumberInput("01 23-ab456789")).toBe("0123456789");
  });

  it("normalizes amount input to one decimal separator and two places", () => {
    expect(normalizeMajorAmountInput("USD 1,2.345")).toBe("1.23");
  });

  it.each([
    ["", "Enter the amount you want to send."],
    ["0", "Enter an amount greater than 0."],
    ["10.999", "Use no more than 2 decimal places."],
    ["ten", "Enter numbers only, for example 25.00."],
    ["10.", "Enter a complete amount, for example 25.00."],
  ])("returns a specific error for %j", (value, message) => {
    expect(majorAmountError(value)).toBe(message);
  });

  it("converts a valid customer amount to integer minor units", () => {
    expect(majorAmountError("10.50")).toBeUndefined();
    expect(parseMajorAmount("10.50")).toBe(1050);
  });
});
