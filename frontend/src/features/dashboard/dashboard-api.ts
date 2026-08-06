import { authenticatedRequest } from "@/features/auth/auth-api";
import type { Account, TransactionHistoryPage } from "@/lib/api/contracts";

export function listOwnedAccounts() {
  return authenticatedRequest<Account[]>("/accounts?page_id=1&page_size=100");
}

export function listRecentAccountTransactions(accountID: string) {
  return authenticatedRequest<TransactionHistoryPage>(
    `/accounts/${encodeURIComponent(accountID)}/transactions?page_size=5`,
  );
}
