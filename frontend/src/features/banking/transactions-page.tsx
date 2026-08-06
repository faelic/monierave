"use client";

import { useQuery } from "@tanstack/react-query";
import type { Route } from "next";
import Link from "next/link";

import { listOwnedAccounts } from "@/features/dashboard/dashboard-api";
import { maskOwnedAccountNumber } from "@/features/dashboard/financial-format";
import { queryKeys } from "@/lib/query/query-keys";

export function TransactionsPage() {
  const accounts = useQuery({
    queryFn: listOwnedAccounts,
    queryKey: queryKeys.accounts.all,
  });
  return (
    <div>
      <h1 className="text-4xl font-semibold">Transactions</h1>
      <p className="text-ink-600 mt-3">
        Select an account to review its complete transaction history.
      </p>
      {accounts.isPending ? (
        <p className="mt-8" role="status">
          Loading accounts…
        </p>
      ) : accounts.isError ? (
        <p className="mt-8" role="alert">
          We could not load your accounts.
        </p>
      ) : (
        <ul className="border-line-200 divide-line-200 mt-8 divide-y border-y">
          {accounts.data.map((account) => (
            <li key={account.id}>
              <Link
                className="flex min-h-20 items-center justify-between gap-4 py-3 no-underline"
                href={`/app/accounts/${account.id}` as Route}
              >
                <span>
                  <strong>{account.currency} account</strong>
                  <span className="text-ink-600 mt-1 block text-sm">
                    {maskOwnedAccountNumber(account.account_number)}
                  </span>
                </span>
                <span className="text-evergreen-800 font-semibold">
                  View history
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
