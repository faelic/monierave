import { describe, expect, it } from "vitest";

import {
  accountNumberSchema,
  nicknameSchema,
  parseMajorAmount,
} from "@/features/banking/banking-validation";

describe("banking validation", () => {
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
});
