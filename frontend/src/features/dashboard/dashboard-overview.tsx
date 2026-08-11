"use client";

import { useQuery } from "@tanstack/react-query";
import {
  ArrowDownLeft,
  ArrowUpRight,
  Eye,
  EyeOff,
  LockKeyhole,
  MailCheck,
  Plus,
  RefreshCw,
  Send,
  ShieldCheck,
  UserRound,
  WalletCards,
} from "lucide-react";
import type { Route } from "next";
import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { useAuth } from "@/features/auth/auth-provider";
import { hasFinancialAccess } from "@/features/auth/user-access";
import {
  listOwnedAccounts,
  listRecentAccountTransactions,
} from "@/features/dashboard/dashboard-api";
import {
  accountLabel,
  formatMinorAmount,
  formatTransactionAmount,
} from "@/features/dashboard/financial-format";
import { isApiError } from "@/lib/api/api-error";
import type { Account, BankingTransaction } from "@/lib/api/contracts";
import { queryKeys } from "@/lib/query/query-keys";

export function DashboardOverview() {
  const { user } = useAuth();

  if (!hasFinancialAccess(user)) {
    return <LimitedDashboardOverview />;
  }

  return <FinancialDashboardOverview />;
}

function FinancialDashboardOverview() {
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

      <section
        aria-label="Quick actions"
        className="mt-7 grid gap-3 sm:grid-cols-3"
      >
        <QuickAction
          description="Review recipient and transfer details"
          href="/app/transfers/new"
          icon={Send}
          label="Send money"
        />
        <QuickAction
          description="Open a USD or EUR account"
          href="/app/accounts/new"
          icon={Plus}
          label="Open account"
        />
        <QuickAction
          description="See activity across every account"
          href="/app/transactions"
          icon={ArrowUpRight}
          label="View activity"
        />
      </section>

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
                {accountLabel(primaryAccount)} · {primaryAccount.account_number}
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

function LimitedDashboardOverview() {
  const { user } = useAuth();
  const disabled = user?.account_status === "disabled";
  const expiresAt = user?.registration_expires_at
    ? formatRegistrationDeadline(user.registration_expires_at)
    : null;

  return (
    <div>
      <header className="border-line-200 border-b pb-7">
        <p className="text-ink-600 text-sm">
          Welcome, {firstName(user?.full_name)}
        </p>
        <h1 className="mt-1 max-w-3xl text-4xl font-semibold tracking-[-0.035em] sm:text-5xl">
          {disabled
            ? "Recover your Monierave registration."
            : "Verify your email to activate banking."}
        </h1>
        <p className="text-ink-600 mt-4 max-w-2xl leading-7">
          Your secure session and profile are available. Financial information
          and money movement stay locked until your email address is confirmed.
        </p>
      </header>

      <section className="mt-7 grid gap-4 lg:grid-cols-[minmax(0,1.35fr)_minmax(18rem,0.65fr)]">
        <article className="border-line-200 rounded-md border bg-white p-6 sm:p-7">
          <div className="flex items-start gap-4">
            <span className="text-evergreen-800 grid size-11 shrink-0 place-items-center rounded-full bg-[var(--product-accent-soft)]">
              <MailCheck aria-hidden="true" className="size-5" />
            </span>
            <div>
              <p className="text-ink-600 text-xs font-bold tracking-[0.12em] uppercase">
                Account activation
              </p>
              <h2 className="mt-2 text-2xl font-semibold">
                {disabled
                  ? "Update your email to continue"
                  : "One step remains"}
              </h2>
              <p className="text-ink-600 mt-2 leading-7">
                {disabled
                  ? "The registration window ended, but you can recover it by confirming a current email address."
                  : "Use the verification message we sent to confirm that this email address belongs to you."}
              </p>
            </div>
          </div>

          <dl className="border-line-200 mt-6 grid gap-4 border-y py-5 sm:grid-cols-2">
            <div>
              <dt className="text-ink-600 text-xs font-bold tracking-[0.1em] uppercase">
                Current address
              </dt>
              <dd className="mt-1 font-semibold break-all">{user?.email}</dd>
            </div>
            <div>
              <dt className="text-ink-600 text-xs font-bold tracking-[0.1em] uppercase">
                Registration status
              </dt>
              <dd className="mt-1 font-semibold capitalize">
                {user?.account_status ?? "pending"}
                {!disabled && expiresAt ? ` · until ${expiresAt}` : ""}
              </dd>
            </div>
          </dl>

          <div className="mt-6 flex flex-col gap-3 sm:flex-row">
            <Button asChild>
              <Link href="/verification-needed">Continue verification</Link>
            </Button>
            <Button asChild variant="secondary">
              <Link href="/app/profile/edit">Review profile and email</Link>
            </Button>
          </div>
        </article>

        <aside className="border-line-200 rounded-md border bg-[var(--product-navigation)] p-6 text-white sm:p-7">
          <LockKeyhole
            aria-hidden="true"
            className="size-6 text-[var(--product-accent)]"
          />
          <h2 className="mt-5 text-xl font-semibold">
            Protected until verified
          </h2>
          <p className="mt-2 text-sm leading-6 text-white/65">
            These controls remain unavailable in both the interface and API.
          </p>
          <ul className="mt-5 grid gap-3 text-sm font-semibold text-white/85">
            <li>Accounts and posted balances</li>
            <li>Transfers and recipient lookup</li>
            <li>Beneficiaries</li>
            <li>Transactions and statements</li>
          </ul>
        </aside>
      </section>

      <section aria-labelledby="available-heading" className="mt-10">
        <h2 className="text-xl font-semibold" id="available-heading">
          Available now
        </h2>
        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <QuickAction
            description="Review your identity details or change your email address"
            href="/app/profile"
            icon={UserRound}
            label="Profile"
          />
          <QuickAction
            description="Review your active session and securely sign out everywhere"
            href="/app/security"
            icon={ShieldCheck}
            label="Session security"
          />
        </div>
      </section>
    </div>
  );
}

function formatRegistrationDeadline(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function QuickAction({
  description,
  href,
  icon: Icon,
  label,
}: {
  description: string;
  href: Route;
  icon: typeof Send;
  label: string;
}) {
  return (
    <Link
      className="border-line-200 hover:border-evergreen-700 group flex min-h-20 items-center gap-3.5 rounded-md border bg-white p-3.5 no-underline transition-colors"
      href={href}
    >
      <span className="text-evergreen-800 grid size-10 shrink-0 place-items-center rounded-full bg-[var(--product-accent-soft)]">
        <Icon aria-hidden="true" className="size-4" />
      </span>
      <span>
        <strong className="block">{label}</strong>
        <span className="text-ink-600 mt-1 block text-sm leading-5">
          {description}
        </span>
      </span>
    </Link>
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
      className={`border-line-200 flex min-h-44 flex-col rounded-md border bg-white p-4 ${
        primary ? "border-t-2 border-t-[var(--product-accent)]" : ""
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="font-semibold">{accountLabel(account)}</p>
          <p
            className="text-ink-600 mt-1 font-mono text-sm tracking-[0.08em]"
            data-financial-number
          >
            {account.account_number}
          </p>
        </div>
        <AccountStatus status={account.status} />
      </div>
      <div className="mt-auto pt-6">
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
          className="mt-2 text-2xl font-semibold tracking-[-0.025em]"
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
          className="border-line-200 h-44 animate-pulse rounded-md border bg-white p-4 motion-reduce:animate-none"
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
