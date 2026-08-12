"use client";

import { useQueries, useQuery } from "@tanstack/react-query";
import { ArrowDownLeft, ArrowUpRight, Search } from "lucide-react";
import type { Route } from "next";
import Link from "next/link";
import { useDeferredValue, useState } from "react";

import {
  EmptyState,
  ErrorState,
  LoadingState,
} from "@/components/ui/page-state";
import { StatusBadge } from "@/components/ui/status-badge";
import { listAccountTransactions } from "@/features/banking/banking-api";
import { bankingErrorMessage } from "@/features/banking/banking-errors";
import { listOwnedAccounts } from "@/features/dashboard/dashboard-api";
import { formatTransactionAmount } from "@/features/dashboard/financial-format";
import type { Account, BankingTransaction } from "@/lib/api/contracts";
import { queryKeys } from "@/lib/query/query-keys";

type DirectionFilter = "all" | BankingTransaction["direction"];
type StatusFilter = "all" | BankingTransaction["status"];

export function TransactionsPage() {
  const accounts = useQuery({
    queryFn: listOwnedAccounts,
    queryKey: queryKeys.accounts.all,
  });
  const activityQueries = useQueries({
    queries: (accounts.data ?? []).map((account) => ({
      queryFn: () => listAccountTransactions(account.id, { pageSize: 50 }),
      queryKey: queryKeys.accounts.transactions(account.id, {
        scope: "activity",
      }),
    })),
  });
  const [accountID, setAccountID] = useState("all");
  const [direction, setDirection] = useState<DirectionFilter>("all");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search.trim().toLowerCase());
  const accountByID = new Map(
    (accounts.data ?? []).map((account) => [account.id, account]),
  );
  const transactions = activityQueries
    .flatMap((query) => query.data?.transactions ?? [])
    .sort(
      (left, right) =>
        new Date(right.created_at).getTime() -
        new Date(left.created_at).getTime(),
    );
  const filteredTransactions = transactions.filter((transaction) => {
    if (accountID !== "all" && transaction.account_id !== accountID) {
      return false;
    }
    if (direction !== "all" && transaction.direction !== direction) {
      return false;
    }
    if (status !== "all" && transaction.status !== status) {
      return false;
    }
    if (!deferredSearch) return true;
    return [
      transaction.counterparty,
      transaction.narration,
      transaction.reference,
      transaction.type,
    ].some((value) => value.toLowerCase().includes(deferredSearch));
  });
  const activityPending = activityQueries.some((query) => query.isPending);
  const activityError = activityQueries.find((query) => query.isError)?.error;

  return (
    <div>
      <header className="border-line-200 border-b pb-7">
        <p className="text-evergreen-700 text-sm font-bold tracking-[0.14em] uppercase">
          Complete record
        </p>
        <h1 className="mt-2 text-4xl font-semibold tracking-[-0.035em]">
          Activity
        </h1>
        <p className="text-ink-600 mt-3 max-w-2xl leading-7">
          Review transaction outcomes across every currency account. Amounts
          remain in their original currencies and are never combined.
        </p>
      </header>

      {accounts.isPending ? (
        <LoadingState label="Loading your activity…" />
      ) : accounts.isError ? (
        <ErrorState
          description={bankingErrorMessage(accounts.error)}
          onRetry={() => void accounts.refetch()}
          title="We could not load your accounts."
        />
      ) : accounts.data.length === 0 ? (
        <EmptyState
          description="Open a currency account before reviewing transaction activity."
          title="No accounts to review"
        />
      ) : (
        <>
          <ActivityFilters
            accountID={accountID}
            accounts={accounts.data}
            direction={direction}
            onAccountChange={setAccountID}
            onDirectionChange={setDirection}
            onSearchChange={setSearch}
            onStatusChange={setStatus}
            search={search}
            status={status}
          />

          {activityPending ? (
            <LoadingState label="Loading transaction records…" />
          ) : activityError ? (
            <ErrorState
              description={bankingErrorMessage(activityError)}
              onRetry={() =>
                void Promise.all(
                  activityQueries.map((query) => query.refetch()),
                )
              }
              title="Some activity could not be loaded."
            />
          ) : filteredTransactions.length === 0 ? (
            <EmptyState
              description={
                transactions.length === 0
                  ? "Transactions will appear here as soon as they are recorded."
                  : "Adjust the account, status, direction, or search filters."
              }
              title={
                transactions.length === 0
                  ? "No transaction activity yet"
                  : "No activity matches these filters"
              }
            />
          ) : (
            <section aria-labelledby="activity-results" className="mt-7">
              <div className="flex items-center justify-between gap-4">
                <h2 className="text-lg font-semibold" id="activity-results">
                  Transaction records
                </h2>
                <p className="text-ink-600 text-sm">
                  {filteredTransactions.length} shown
                </p>
              </div>
              <ul className="border-line-200 divide-line-200 mt-3 divide-y overflow-hidden rounded-md border bg-white">
                {filteredTransactions.map((transaction) => (
                  <li key={`${transaction.account_id}-${transaction.id}`}>
                    <ActivityRow
                      account={accountByID.get(transaction.account_id)}
                      transaction={transaction}
                    />
                  </li>
                ))}
              </ul>
            </section>
          )}
        </>
      )}
    </div>
  );
}

function ActivityFilters({
  accountID,
  accounts,
  direction,
  onAccountChange,
  onDirectionChange,
  onSearchChange,
  onStatusChange,
  search,
  status,
}: {
  accountID: string;
  accounts: Account[];
  direction: DirectionFilter;
  onAccountChange: (value: string) => void;
  onDirectionChange: (value: DirectionFilter) => void;
  onSearchChange: (value: string) => void;
  onStatusChange: (value: StatusFilter) => void;
  search: string;
  status: StatusFilter;
}) {
  const selectClass =
    "border-line-300 min-h-12 w-full min-w-0 rounded-sm border bg-white px-3 text-sm";
  const labelClass = "grid min-w-0 gap-2";
  const labelTextClass =
    "text-ink-600 px-0.5 text-xs font-semibold tracking-[0.04em]";
  return (
    <section
      aria-label="Activity filters"
      className="border-line-200 mt-7 grid grid-cols-1 gap-3 rounded-md border bg-[var(--product-surface-subtle)] p-4 min-[24rem]:grid-cols-2 md:grid-cols-[minmax(13rem,1fr)_minmax(11rem,auto)_minmax(9rem,auto)_minmax(9rem,auto)] md:items-end"
    >
      <label className={`${labelClass} min-[24rem]:col-span-2 md:col-span-1`}>
        <span className={labelTextClass}>Search activity</span>
        <span className="relative block">
          <Search
            aria-hidden="true"
            className="text-ink-600 absolute top-1/2 left-3 size-4 -translate-y-1/2"
          />
          <input
            className="border-line-300 min-h-12 w-full rounded-sm border bg-white pr-3 pl-10"
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder="Reference or recipient"
            type="search"
            value={search}
          />
        </span>
      </label>
      <label className={`${labelClass} min-[24rem]:col-span-2 md:col-span-1`}>
        <span className={labelTextClass}>Account</span>
        <select
          className={selectClass}
          onChange={(event) => onAccountChange(event.target.value)}
          value={accountID}
        >
          <option value="all">All accounts</option>
          {accounts.map((account) => (
            <option key={account.id} value={account.id}>
              {account.currency} · {account.account_number}
            </option>
          ))}
        </select>
      </label>
      <label className={labelClass}>
        <span className={labelTextClass}>Direction</span>
        <select
          className={selectClass}
          onChange={(event) =>
            onDirectionChange(event.target.value as DirectionFilter)
          }
          value={direction}
        >
          <option value="all">All directions</option>
          <option value="incoming">Incoming</option>
          <option value="outgoing">Outgoing</option>
        </select>
      </label>
      <label className={labelClass}>
        <span className={labelTextClass}>Status</span>
        <select
          className={selectClass}
          onChange={(event) =>
            onStatusChange(event.target.value as StatusFilter)
          }
          value={status}
        >
          <option value="all">All statuses</option>
          <option value="pending">Pending</option>
          <option value="posted">Posted</option>
          <option value="failed">Failed</option>
          <option value="reversed">Reversed</option>
        </select>
      </label>
    </section>
  );
}

function ActivityRow({
  account,
  transaction,
}: {
  account: Account | undefined;
  transaction: BankingTransaction;
}) {
  const incoming = transaction.direction === "incoming";
  const tone =
    transaction.status === "posted"
      ? "positive"
      : transaction.status === "pending"
        ? "warning"
        : transaction.status === "failed"
          ? "danger"
          : "neutral";
  return (
    <Link
      className="grid min-h-24 grid-cols-[2.5rem_minmax(0,1fr)_auto] items-center gap-3 px-4 py-3 no-underline hover:bg-[var(--product-surface-subtle)] sm:px-5"
      href={`/app/transactions/${transaction.reference}` as Route}
    >
      <span
        className={`grid size-10 place-items-center rounded-full ${
          incoming
            ? "bg-jade-100 text-success-700"
            : "text-evergreen-800 bg-[var(--product-accent-soft)]"
        }`}
      >
        {incoming ? (
          <ArrowDownLeft aria-hidden="true" className="size-5" />
        ) : (
          <ArrowUpRight aria-hidden="true" className="size-5" />
        )}
      </span>
      <span className="min-w-0">
        <strong className="block truncate">
          {transaction.counterparty || transaction.type.replaceAll("_", " ")}
        </strong>
        <span className="text-ink-600 mt-1 flex flex-wrap items-center gap-2 text-sm">
          <span>{formatDate(transaction.created_at)}</span>
          {account ? <span>{account.currency} account</span> : null}
          <StatusBadge className="capitalize" tone={tone}>
            {transaction.status}
          </StatusBadge>
        </span>
      </span>
      <strong className="pl-2 text-right" data-financial-number>
        {formatTransactionAmount(transaction)}
      </strong>
    </Link>
  );
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Date unavailable";
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}
