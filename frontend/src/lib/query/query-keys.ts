export const queryKeys = {
  currentUser: ["current-user"] as const,
  emailStatus: ["current-user", "email-status"] as const,
  accounts: {
    all: ["accounts"] as const,
    recentTransactions: (accountID: string) =>
      ["accounts", "recent-transactions", accountID] as const,
    detail: (accountID: string) => ["accounts", "detail", accountID] as const,
    statement: (accountID: string, filters: Record<string, string>) =>
      ["accounts", "statement", accountID, filters] as const,
    transactions: (accountID: string, filters: Record<string, string>) =>
      ["accounts", "transactions", accountID, filters] as const,
  },
  beneficiaries: {
    all: ["beneficiaries"] as const,
  },
  transactions: {
    detail: (reference: string) =>
      ["transactions", "detail", reference] as const,
  },
} as const;
