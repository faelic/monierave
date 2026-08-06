"use client";

import { useQuery } from "@tanstack/react-query";
import {
  ArrowDownLeft,
  ArrowUpRight,
  Eye,
  EyeOff,
  RefreshCw,
  WalletCards,
} from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { useAuth } from "@/features/auth/auth-provider";
import {
  listOwnedAccounts,
  listRecentAccountTransactions,
} from "@/features/dashboard/dashboard-api";
import {
  accountLabel,
  formatMinorAmount,
  formatTransactionAmount,
  maskOwnedAccountNumber,
} from "@/features/dashboard/financial-format";
import { isApiError } from "@/lib/api/api-error";
import type { Account, BankingTransaction } from "@/lib/api/contracts";
import { queryKeys } from "@/lib/query/query-keys";

export function DashboardOverview() {
  const { user } = useAuth();
  const [balancesVisible, setBalancesVisible] = useState(true);
  const accounts = useQuery({
    queryFn: listOwnedAccounts,
    queryKey: queryKeys.accounts.all,
  });
  const primaryAccount =
    accounts.data?.find((account) => account.status === "active") ??
    accounts.data?.[0];
  const recentActivity = useQuery({
    enabled: Boolean(primaryAccount),
    queryFn: () => listRecentAccountTransactions(primaryAccount!.id),
    queryKey: primaryAccount
      ? queryKeys.accounts.recentTransactions(primaryAccount.id)
      : ["accounts", "recent-transactions", "none"],
  });

  return (
    <div>
      <header className="border-line-200 flex flex-col gap-5 border-b pb-7 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-ink-600 text-sm">
            Welcome back, {firstName(user?.full_name)}
          </p>
          <h1 className="mt-1 text-4xl font-semibold tracking-[-0.035em]">
            Overview
          </h1>
          <p className="text-ink-600 mt-3 max-w-2xl leading-7">
            Review your current posted balances and the latest activity for your
            primary account.
          </p>
        </div>
        <Button
          aria-pressed={!balancesVisible}
          className="self-start sm:self-auto"
          onClick={() => setBalancesVisible((visible) => !visible)}
          variant="secondary"
        >
          {balancesVisible ? (
            <EyeOff aria-hidden="true" className="size-4" />
          ) : (
            <Eye aria-hidden="true" className="size-4" />
          )}
          {balancesVisible ? "Hide balances" : "Show balances"}
        </Button>
      </header>

      <section aria-labelledby="accounts-heading" className="mt-9">
        <div className="flex items-baseline justify-between gap-4">
          <h2 className="text-xl font-semibold" id="accounts-heading">
            Your accounts
          </h2>
          {accounts.data?.length ? (
            <p className="text-ink-600 text-sm">
              {accounts.data.length}{" "}
              {accounts.data.length === 1 ? "account" : "accounts"}
            </p>
          ) : null}
        </div>

        {accounts.isPending ? (
          <AccountGridSkeleton />
        ) : accounts.isError ? (
          <DashboardSectionError
            error={accounts.error}
            onRetry={() => void accounts.refetch()}
            title="We could not load your accounts."
          />
        ) : accounts.data.length === 0 ? (
          <AccountEmptyState onRefresh={() => void accounts.refetch()} />
        ) : (
          <ul className="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {accounts.data.map((account) => (
              <li key={account.id}>
                <AccountSummary
                  account={account}
                  balancesVisible={balancesVisible}
                  primary={account.id === primaryAccount?.id}
                />
              </li>
            ))}
          </ul>
        )}
      </section>

      {primaryAccount ? (
        <section aria-labelledby="activity-heading" className="mt-12">
          <div className="border-line-200 flex flex-col gap-2 border-b pb-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 className="text-xl font-semibold" id="activity-heading">
                Recent activity
              </h2>
              <p className="text-ink-600 mt-1 text-sm">
                {accountLabel(primaryAccount)} ·{" "}
                {maskOwnedAccountNumber(primaryAccount.account_number)}
              </p>
            </div>
            <p className="text-ink-600 text-sm">Latest five transactions</p>
          </div>

          {recentActivity.isPending ? (
            <TransactionListSkeleton />
          ) : recentActivity.isError ? (
            <DashboardSectionError
              error={recentActivity.error}
              onRetry={() => void recentActivity.refetch()}
              title="We could not load recent activity."
            />
          ) : recentActivity.data.transactions.length === 0 ? (
            <div className="border-line-200 mt-4 border-y py-9 text-center">
              <h3 className="font-semibold">No posted activity yet.</h3>
              <p className="text-ink-600 mt-2 text-sm">
                Transactions for this account will appear here after they are
                recorded.
              </p>
            </div>
          ) : (
            <ul className="border-line-200 divide-line-200 mt-4 divide-y border-y">
              {recentActivity.data.transactions.map((transaction) => (
                <li key={transaction.id}>
                  <TransactionRow transaction={transaction} />
                </li>
              ))}
            </ul>
          )}
        </section>
      ) : null}
    </div>
  );
}

function AccountSummary({
  account,
  balancesVisible,
  primary,
}: {
  account: Account;
  balancesVisible: boolean;
  primary: boolean;
}) {
  return (
    <article
      className={`border-line-200 flex min-h-56 flex-col rounded-md border bg-white p-5 ${
        primary ? "border-t-evergreen-700 border-t-4" : ""
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="font-semibold">{accountLabel(account)}</p>
          <p
            className="text-ink-600 mt-1 font-mono text-sm tracking-[0.08em]"
            data-financial-number
          >
            {maskOwnedAccountNumber(account.account_number)}
          </p>
        </div>
        <AccountStatus status={account.status} />
      </div>
      <div className="mt-auto pt-8">
        <p className="text-ink-600 text-xs font-semibold tracking-[0.1em] uppercase">
          Current posted balance
        </p>
        <p
          aria-label={
            balancesVisible
              ? `Current posted balance ${formatMinorAmount(
                  account.balance,
                  account.currency,
                )}`
              : "Current posted balance hidden"
          }
          className="mt-2 text-3xl font-semibold tracking-[-0.025em]"
          data-financial-number
        >
          {balancesVisible
            ? formatMinorAmount(account.balance, account.currency)
            : "••••••"}
        </p>
        {primary ? (
          <p className="text-evergreen-700 mt-3 text-xs font-semibold">
            Primary overview account
          </p>
        ) : null}
      </div>
    </article>
  );
}

function AccountStatus({ status }: { status: Account["status"] }) {
  const tone =
    status === "active"
      ? "positive"
      : status === "frozen"
        ? "warning"
        : "neutral";
  return (
    <StatusBadge className="capitalize" tone={tone}>
      {status}
    </StatusBadge>
  );
}

function TransactionRow({ transaction }: { transaction: BankingTransaction }) {
  const incoming = transaction.direction === "incoming";
  return (
    <article className="grid min-h-20 grid-cols-[2.5rem_minmax(0,1fr)_auto] items-center gap-3 py-3">
      <span
        className={`grid size-10 place-items-center rounded-sm ${
          incoming
            ? "bg-jade-100 text-success-700"
            : "bg-paper-100 text-evergreen-800"
        }`}
      >
        {incoming ? (
          <ArrowDownLeft aria-hidden="true" className="size-5" />
        ) : (
          <ArrowUpRight aria-hidden="true" className="size-5" />
        )}
      </span>
      <div className="min-w-0">
        <p className="truncate font-semibold">
          {transaction.counterparty || transactionLabel(transaction)}
        </p>
        <p className="text-ink-600 mt-0.5 truncate text-sm">
          <time dateTime={transaction.created_at}>
            {formatDateTime(transaction.created_at)}
          </time>
          {" · "}
          <span className="capitalize">{transaction.status}</span>
          {transaction.narration ? ` · ${transaction.narration}` : ""}
        </p>
      </div>
      <p
        aria-label={`${incoming ? "Incoming" : "Outgoing"} ${formatMinorAmount(
          Math.abs(transaction.amount),
          transaction.currency,
        )}`}
        className={`pl-2 text-right font-semibold ${
          incoming ? "text-success-700" : "text-ink-950"
        }`}
        data-financial-number
      >
        {formatTransactionAmount(transaction)}
      </p>
    </article>
  );
}

function AccountGridSkeleton() {
  return (
    <div
      aria-label="Loading accounts"
      className="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3"
      role="status"
    >
      {[0, 1].map((item) => (
        <div
          className="border-line-200 h-56 animate-pulse rounded-md border bg-white p-5 motion-reduce:animate-none"
          key={item}
        >
          <div className="bg-paper-100 h-5 w-28 rounded" />
          <div className="bg-paper-100 mt-3 h-4 w-24 rounded" />
          <div className="bg-paper-100 mt-20 h-8 w-44 rounded" />
        </div>
      ))}
      <span className="sr-only">Loading your accounts.</span>
    </div>
  );
}

function TransactionListSkeleton() {
  return (
    <div
      aria-label="Loading recent activity"
      className="border-line-200 mt-4 grid gap-4 border-y py-4"
      role="status"
    >
      {[0, 1, 2].map((item) => (
        <div
          className="flex animate-pulse gap-3 motion-reduce:animate-none"
          key={item}
        >
          <div className="bg-paper-100 size-10 rounded-sm" />
          <div className="flex-1">
            <div className="bg-paper-100 h-5 w-36 rounded" />
            <div className="bg-paper-100 mt-2 h-4 w-56 max-w-full rounded" />
          </div>
        </div>
      ))}
      <span className="sr-only">Loading recent account activity.</span>
    </div>
  );
}

function AccountEmptyState({ onRefresh }: { onRefresh: () => void }) {
  return (
    <div className="border-line-200 mt-4 border-y py-12 text-center">
      <span className="bg-jade-100 text-evergreen-800 mx-auto grid size-12 place-items-center rounded-sm">
        <WalletCards aria-hidden="true" className="size-6" />
      </span>
      <h3 className="mt-4 text-lg font-semibold">No financial accounts yet.</h3>
      <p className="text-ink-600 mx-auto mt-2 max-w-md text-sm leading-6">
        This overview will show each account’s current posted balance after an
        account has been created.
      </p>
      <Button className="mt-5" onClick={onRefresh} variant="secondary">
        <RefreshCw aria-hidden="true" className="size-4" />
        Refresh accounts
      </Button>
    </div>
  );
}

function DashboardSectionError({
  error,
  onRetry,
  title,
}: {
  error: Error;
  onRetry: () => void;
  title: string;
}) {
  const sessionExpired = isApiError(error) && error.status === 401;
  return (
    <div
      className="border-danger-700 mt-4 rounded-sm border-l-4 bg-[#fff5f3] px-5 py-5"
      role="alert"
    >
      <h3 className="font-semibold">{title}</h3>
      <p className="text-ink-600 mt-1 text-sm">
        {sessionExpired
          ? "Your secure session ended. We’re returning you to sign in."
          : "Your existing information has not been changed. Retry when your connection is stable."}
      </p>
      {!sessionExpired ? (
        <Button
          className="mt-4"
          onClick={onRetry}
          size="compact"
          variant="secondary"
        >
          Retry
        </Button>
      ) : null}
    </div>
  );
}

function firstName(fullName?: string) {
  return fullName?.trim().split(/\s+/)[0] || "there";
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Date unavailable";
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function transactionLabel(transaction: BankingTransaction) {
  return transaction.type.replaceAll("_", " ");
}
