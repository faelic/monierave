import { authenticatedRequest } from "@/features/auth/auth-api";
import type {
  Account,
  MoneyMovement,
  TransactionHistoryPage,
} from "@/lib/api/contracts";

export function listOwnedAccounts() {
  return authenticatedRequest<Account[]>("/accounts?page_id=1&page_size=100");
}

export function listRecentAccountTransactions(accountID: string) {
  return authenticatedRequest<TransactionHistoryPage>(
    `/accounts/${encodeURIComponent(accountID)}/transactions?page_size=5`,
  );
}

export function getAccountMoneyMovement(
  accountID: string,
  range: { from: string; interval: "day" | "week"; to: string },
) {
  const params = new URLSearchParams(range);
  return authenticatedRequest<MoneyMovement>(
    `/accounts/${encodeURIComponent(accountID)}/money-movement?${params}`,
  );
}
