import { z } from "zod";

const encoder = new TextEncoder();

export const usernameSchema = z
  .string()
  .min(3, "Username must contain at least 3 characters.")
  .max(32, "Username must contain no more than 32 characters.")
  .regex(/^[A-Za-z0-9]+$/, "Use letters and numbers only.");

export const fullNameSchema = z
  .string()
  .trim()
  .refine((value) => Array.from(value).length >= 1, "Enter your full name.")
  .refine(
    (value) => Array.from(value).length <= 100,
    "Full name must contain no more than 100 characters.",
  );

export const emailSchema = z
  .string()
  .trim()
  .max(254, "Email address must contain no more than 254 characters.")
  .email("Enter a valid email address.")
  .transform((value) => value.toLowerCase());

export const passwordSchema = z
  .string()
  .min(8, "Password must contain at least 8 characters.")
  .refine(
    (value) => encoder.encode(value).byteLength <= 72,
    "Password must contain no more than 72 bytes.",
  );

export const signupSchema = z.object({
  username: usernameSchema,
  full_name: fullNameSchema,
  email: emailSchema,
  password: passwordSchema,
});

export const loginSchema = z.object({
  username: usernameSchema,
  password: passwordSchema,
});

export const emailUpdateSchema = z.object({
  email: emailSchema,
});

export type SignupValues = z.input<typeof signupSchema>;
export type LoginValues = z.input<typeof loginSchema>;
export type EmailUpdateValues = z.input<typeof emailUpdateSchema>;

export function safeReturnPath(value: string | null | undefined) {
  if (
    !value ||
    !value.startsWith("/") ||
    value.startsWith("//") ||
    value.includes("\\")
  ) {
    return null;
  }

  try {
    const parsed = new URL(value, "https://monierave.invalid");
    return parsed.origin === "https://monierave.invalid"
      ? `${parsed.pathname}${parsed.search}${parsed.hash}`
      : null;
  } catch {
    return null;
  }
}
