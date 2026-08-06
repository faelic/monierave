import { z } from "zod";

export const accountNumberSchema = z
  .string()
  .trim()
  .regex(/^[0-9]{10}$/, "Enter a 10-digit account number.");

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

export function parseMajorAmount(value: string, fractionDigits = 2) {
  const normalized = value.trim();
  const pattern = new RegExp(
    `^(?:0|[1-9][0-9]*)(?:\\.([0-9]{1,${fractionDigits}}))?$`,
  );
  const match = normalized.match(pattern);
  if (!match) {
    return null;
  }
  const [whole = "0", fraction = ""] = normalized.split(".");
  const minor =
    BigInt(whole) * 10n ** BigInt(fractionDigits) +
    BigInt(fraction.padEnd(fractionDigits, "0"));
  if (minor <= 0n || minor > BigInt(Number.MAX_SAFE_INTEGER)) {
    return null;
  }
  return Number(minor);
}
