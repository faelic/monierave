import type { Account, BankingTransaction } from "@/lib/api/contracts";

export function formatMinorAmount(
  amount: number,
  currency: Account["currency"],
  locale?: string,
) {
  const formatter = new Intl.NumberFormat(locale, {
    currency,
    currencyDisplay: "code",
    style: "currency",
  });
  const digits = formatter.resolvedOptions().maximumFractionDigits ?? 2;
  return formatter.format(amount / 10 ** digits);
}

export function formatTransactionAmount(
  transaction: Pick<BankingTransaction, "amount" | "currency" | "direction">,
  locale?: string,
) {
  const amount = Math.abs(transaction.amount);
  const prefix = transaction.direction === "incoming" ? "+" : "−";
  return `${prefix}${formatMinorAmount(amount, transaction.currency, locale)}`;
}

export function accountLabel(account: Pick<Account, "currency" | "status">) {
  const lifecycle = account.status === "closed" ? "Closed " : "";
  return `${lifecycle}${account.currency} account`;
}
