import { isApiError } from "@/lib/api/api-error";

const messages: Record<string, string> = {
  account_already_exists:
    "An account is already open in this currency. Choose another currency or view your existing account.",
  account_balance_not_zero:
    "This account must have a zero posted balance before it can be closed.",
  account_closed: "A closed account cannot complete this action.",
  account_frozen: "This account is frozen and cannot send money.",
  beneficiary_already_exists: "This recipient is already saved.",
  currency_mismatch:
    "The sender and recipient accounts use different currencies.",
  daily_transfer_limit_exceeded:
    "This transfer would exceed your daily transfer limit.",
  idempotency_conflict:
    "This transfer key belongs to a different request. Review the transfer again.",
  insufficient_funds:
    "This account does not have enough posted funds for the transfer.",
  per_transfer_limit_exceeded:
    "This amount exceeds the limit for one transfer.",
  recipient_not_found:
    "We could not find a recipient that can receive this transfer.",
  same_account: "The sender and recipient accounts must be different.",
};

export function bankingErrorMessage(error: unknown) {
  if (!isApiError(error)) {
    return "We could not complete this request. Please try again.";
  }
  if (error.code === "network_error") {
    return "We could not confirm the result. Check your connection before trying again.";
  }
  if (error.status === 429) {
    return "Too many requests. Please wait before trying again.";
  }
  return (
    messages[error.code] ??
    "We could not complete this request. Your existing information has not changed."
  );
}
