import { z } from "zod";

export const accountNumberSchema = z
  .string()
  .trim()
  .min(1, "Enter the recipient's account number.")
  .regex(/^[0-9]{10}$/, "Account numbers contain 10 digits.");

export const nicknameSchema = z
  .string()
  .trim()
  .min(1, "Enter a nickname.")
  .refine(
    (value) => Array.from(value).length <= 50,
    "Nickname must contain no more than 50 characters.",
  );

export const narrationSchema = z
  .string()
  .trim()
  .max(255, "Narration must contain no more than 255 characters.");

export function normalizeAccountNumberInput(value: string) {
  return value.replace(/[^0-9]/g, "").slice(0, 10);
}

export function normalizeMajorAmountInput(value: string, fractionDigits = 2) {
  const numeric = value.replace(/,/g, ".").replace(/[^0-9.]/g, "");
  const [whole = "", ...fractionParts] = numeric.split(".");
  if (fractionParts.length === 0) return whole;
  return `${whole}.${fractionParts.join("").slice(0, fractionDigits)}`;
}

export function majorAmountError(value: string, fractionDigits = 2) {
  const normalized = value.trim();
  if (!normalized) return "Enter the amount you want to send.";
  if (!/^[0-9.]+$/.test(normalized) || normalized.split(".").length > 2) {
    return "Enter numbers only, for example 25.00.";
  }
  const [, fraction = ""] = normalized.split(".");
  if (fraction.length > fractionDigits) {
    return `Use no more than ${fractionDigits} decimal places.`;
  }
  if (!/^\d+(?:\.\d+)?$/.test(normalized)) {
    return "Enter a complete amount, for example 25.00.";
  }
  const minor = parseMajorAmount(normalized, fractionDigits);
  if (minor === null) {
    const numericValue = Number(normalized);
    return Number.isFinite(numericValue) && numericValue <= 0
      ? "Enter an amount greater than 0."
      : "This amount is too large.";
  }
  return undefined;
}

export function parseMajorAmount(value: string, fractionDigits = 2) {
  const normalized = value.trim();
  const pattern = new RegExp(`^[0-9]+(?:\\.([0-9]{1,${fractionDigits}}))?$`);
  const match = normalized.match(pattern);
  if (!match) return null;
  const [whole = "0", fraction = ""] = normalized.split(".");
  const minor =
    BigInt(whole) * 10n ** BigInt(fractionDigits) +
    BigInt(fraction.padEnd(fractionDigits, "0"));
  if (minor <= 0n || minor > BigInt(Number.MAX_SAFE_INTEGER)) return null;
  return Number(minor);
}
