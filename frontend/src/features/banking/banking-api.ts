import { authenticatedRequest } from "@/features/auth/auth-api";
import type {
  Account,
  Beneficiary,
  BankingTransaction,
  RecipientResolution,
  TransactionHistoryPage,
  TransferResponse,
} from "@/lib/api/contracts";

export type TransactionFilters = {
  cursor?: string;
  direction?: string;
  from?: string;
  pageSize?: number;
  status?: string;
  to?: string;
  type?: string;
};

export type TransferIntent = {
  amount: number;
  currency: Account["currency"];
  from_account_id: string;
  narration: string;
  to_account_number: string;
};

export function createFinancialAccount(currency: Account["currency"]) {
  return authenticatedRequest<Account, { currency: Account["currency"] }>(
    "/accounts",
    { body: { currency }, method: "POST" },
  );
}

export function getOwnedAccount(accountID: string) {
  return authenticatedRequest<Account>(
    `/accounts/${encodeURIComponent(accountID)}`,
  );
}

export function closeOwnedAccount(accountID: string) {
  return authenticatedRequest<Account>(
    `/accounts/${encodeURIComponent(accountID)}/close`,
    { method: "POST" },
  );
}

export function listAccountTransactions(
  accountID: string,
  filters: TransactionFilters = {},
) {
  const params = new URLSearchParams();
  params.set("page_size", String(filters.pageSize ?? 20));
  for (const [key, value] of Object.entries(filters)) {
    if (key !== "pageSize" && value) {
      params.set(key === "pageSize" ? "page_size" : key, String(value));
    }
  }
  return authenticatedRequest<TransactionHistoryPage>(
    `/accounts/${encodeURIComponent(accountID)}/transactions?${params}`,
  );
}

export function getOwnedTransaction(reference: string) {
  return authenticatedRequest<BankingTransaction>(
    `/transactions/${encodeURIComponent(reference)}`,
  );
}

export function resolveRecipient(accountNumber: string) {
  return authenticatedRequest<RecipientResolution, { account_number: string }>(
    "/accounts/resolve",
    {
      body: { account_number: accountNumber },
      method: "POST",
    },
  );
}

export function createTransfer(intent: TransferIntent, idempotencyKey: string) {
  return authenticatedRequest<TransferResponse, TransferIntent>("/transfers", {
    body: intent,
    idempotencyKey,
    method: "POST",
  });
}

export function listBeneficiaries() {
  return authenticatedRequest<Beneficiary[]>(
    "/beneficiaries?page_id=1&page_size=100",
  );
}

export function createBeneficiary(input: {
  destination_account_number: string;
  nickname: string;
}) {
  return authenticatedRequest<Beneficiary, typeof input>("/beneficiaries", {
    body: input,
    method: "POST",
  });
}

export function renameBeneficiary(id: string, nickname: string) {
  return authenticatedRequest<Beneficiary, { nickname: string }>(
    `/beneficiaries/${encodeURIComponent(id)}`,
    { body: { nickname }, method: "PATCH" },
  );
}

export function removeBeneficiary(id: string) {
  return authenticatedRequest<void>(
    `/beneficiaries/${encodeURIComponent(id)}`,
    { method: "DELETE" },
  );
}
