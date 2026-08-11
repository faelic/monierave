import { isApiError } from "@/lib/api/api-error";

type AuthAction = "login" | "resend" | "signup" | "update-email";

export function authErrorMessage(error: unknown, action: AuthAction) {
  if (!isApiError(error)) {
    return "Something went wrong. Please try again.";
  }

  if (error.code === "network_error") {
    return "We could not reach Monierave. Check your connection and try again.";
  }
  if (error.code === "rate_limited" || error.status === 429) {
    return error.retryAfterSeconds
      ? `Too many attempts. Try again in ${formatWait(error.retryAfterSeconds)}.`
      : "Too many attempts. Please wait before trying again.";
  }

  const messages: Record<string, string> = {
    current_password_required:
      "Enter your current password to make this security-sensitive change.",
    email_address_undeliverable:
      "This address cannot receive Monierave mail. Enter a different email address.",
    email_already_exists: "An account already uses this email address.",
    email_already_verified: "Your email address is already verified.",
    invalid_credentials:
      "The username or password is incorrect. Check both and try again.",
    password_breach_check_unavailable:
      "We cannot safely check this password right now. Please try again shortly.",
    password_compromised:
      "This password appears in known data breaches. Choose a different password.",
    username_already_exists: "This username is already in use.",
    verification_cooldown:
      "A verification email was sent recently. Please wait before requesting another.",
    wait_before_requesting_another_verification_email:
      "A verification email was sent recently. Please wait before requesting another.",
  };

  if (messages[error.code]) {
    return messages[error.code];
  }

  if (action === "login" && error.status === 401) {
    return "The username or password is incorrect. Check both and try again.";
  }
  if (action === "resend") {
    return "We could not send another verification email. Please try again.";
  }
  if (action === "update-email") {
    return "We could not update your email address. Please try again.";
  }
  return "We could not create your registration. Please try again.";
}

function formatWait(seconds: number) {
  if (seconds < 60) {
    return `${Math.ceil(seconds)} seconds`;
  }
  const minutes = Math.ceil(seconds / 60);
  return `${minutes} minute${minutes === 1 ? "" : "s"}`;
}
