"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Copy, Plus, RefreshCw, WalletCards } from "lucide-react";
import type { Route } from "next";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  closeOwnedAccount,
  createFinancialAccount,
  getOwnedAccount,
  getOwnedTransaction,
  listAccountTransactions,
} from "@/features/banking/banking-api";
import { bankingErrorMessage } from "@/features/banking/banking-errors";
import { listOwnedAccounts } from "@/features/dashboard/dashboard-api";
import {
  accountLabel,
  formatMinorAmount,
  formatTransactionAmount,
} from "@/features/dashboard/financial-format";
import { isApiError } from "@/lib/api/api-error";
import type { Account, BankingTransaction } from "@/lib/api/contracts";
import { queryKeys } from "@/lib/query/query-keys";

export function AccountsPage() {
  const accounts = useQuery({
    queryFn: listOwnedAccounts,
    queryKey: queryKeys.accounts.all,
  });
  return (
    <div>
      <PageHeader
        action={
          <Button asChild>
            <Link href={"/app/accounts/new" as Route}>
              <Plus aria-hidden="true" className="size-4" />
              Open account
            </Link>
          </Button>
        }
        description="Review each currency account and its current posted balance."
        title="Accounts"
      />
      {accounts.isPending ? (
        <p className="mt-10" role="status">
          Loading accounts…
        </p>
      ) : accounts.isError ? (
        <ErrorPanel
          message={bankingErrorMessage(accounts.error)}
          onRetry={() => void accounts.refetch()}
        />
      ) : accounts.data.length === 0 ? (
        <div className="border-line-200 mt-8 border-y py-12 text-center">
          <WalletCards
            aria-hidden="true"
            className="text-evergreen-700 mx-auto size-8"
          />
          <h2 className="mt-4 text-xl font-semibold">
            Open your first currency account.
          </h2>
          <p className="text-ink-600 mt-2">
            New accounts begin active with a zero current posted balance.
          </p>
          <Button asChild className="mt-6">
            <Link href={"/app/accounts/new" as Route}>Open account</Link>
          </Button>
        </div>
      ) : (
        <ul className="mt-8 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {accounts.data.map((account) => (
            <li key={account.id}>
              <Link
                className="border-line-200 hover:border-evergreen-700 block min-h-44 rounded-md border bg-white p-4 no-underline transition-colors"
                href={`/app/accounts/${account.id}` as Route}
              >
                <div className="flex justify-between gap-3">
                  <div>
                    <h2 className="font-semibold">{accountLabel(account)}</h2>
                    <p className="text-ink-600 mt-1 font-mono text-sm">
                      {account.account_number}
                    </p>
                  </div>
                  <LifecycleStatus status={account.status} />
                </div>
                <p className="text-ink-600 mt-10 text-xs font-semibold tracking-wider uppercase">
                  Current posted balance
                </p>
                <p
                  className="mt-2 text-2xl font-semibold"
                  data-financial-number
                >
                  {formatMinorAmount(account.balance, account.currency)}
                </p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export function CreateAccountPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const accounts = useQuery({
    queryFn: listOwnedAccounts,
    queryKey: queryKeys.accounts.all,
  });
  const [currency, setCurrency] = useState<Account["currency"]>("USD");
  const [error, setError] = useState<string>();
  const existingAccount = accounts.data?.find(
    (account) => account.currency === currency,
  );
  const create = useMutation({
    mutationFn: () => createFinancialAccount(currency),
    onError: (cause) => {
      if (isApiError(cause) && cause.code === "account_already_exists") {
        setError(`You already have a ${currency} account.`);
        void queryClient.invalidateQueries({
          queryKey: queryKeys.accounts.all,
        });
        return;
      }
      setError(bankingErrorMessage(cause));
    },
    onSuccess: async (account) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all });
      router.push(`/app/accounts/${account.id}` as Route);
    },
  });
  return (
    <div className="max-w-xl">
      <BackLink href="/app/accounts" label="Accounts" />
      <h1 className="mt-6 text-4xl font-semibold">Open a currency account</h1>
      <p className="text-ink-600 mt-3 leading-7">
        Choose a currency. Monierave generates the account number internally,
        and the account starts active with a zero current posted balance.
      </p>
      {error ? <ErrorPanel message={error} /> : null}
      <fieldset className="mt-8">
        <legend className="font-semibold">Account currency</legend>
        <div className="mt-3 grid grid-cols-2 gap-3">
          {(["USD", "EUR"] as const).map((value) => {
            const ownedAccount = accounts.data?.find(
              (account) => account.currency === value,
            );
            return (
              <label
                className={`border-line-300 flex min-h-16 items-center gap-3 rounded-sm border bg-white px-4 ${
                  ownedAccount
                    ? "cursor-not-allowed opacity-65"
                    : "cursor-pointer"
                } ${currency === value ? "border-evergreen-700 border-2" : ""}`}
                key={value}
              >
                <input
                  checked={currency === value}
                  disabled={Boolean(ownedAccount)}
                  name="currency"
                  onChange={() => {
                    setCurrency(value);
                    setError(undefined);
                  }}
                  type="radio"
                  value={value}
                />
                <span>
                  <strong className="block">{value}</strong>
                  <span className="text-ink-600 mt-1 block text-xs">
                    {ownedAccount ? "Already open" : "Available"}
                  </span>
                </span>
              </label>
            );
          })}
        </div>
      </fieldset>
      <Button
        className="mt-8 w-full"
        disabled={accounts.isPending || Boolean(existingAccount)}
        loading={create.isPending}
        onClick={() => {
          setError(undefined);
          create.mutate();
        }}
      >
        {existingAccount
          ? `Already have a ${currency} account`
          : `Open ${currency} account`}
      </Button>
      {existingAccount ? (
        <Button asChild className="mt-3 w-full" variant="secondary">
          <Link href={`/app/accounts/${existingAccount.id}` as Route}>
            View {currency} account
          </Link>
        </Button>
      ) : null}
    </div>
  );
}

export function AccountDetailPage({ accountID }: { accountID: string }) {
  const account = useQuery({
    queryFn: () => getOwnedAccount(accountID),
    queryKey: queryKeys.accounts.detail(accountID),
  });
  const history = useQuery({
    enabled: account.isSuccess,
    queryFn: () => listAccountTransactions(accountID, { pageSize: 20 }),
    queryKey: queryKeys.accounts.transactions(accountID, {}),
  });
  const [additionalTransactions, setAdditionalTransactions] = useState<
    BankingTransaction[]
  >([]);
  const [nextCursor, setNextCursor] = useState<string>();
  const [loadedAdditionalPage, setLoadedAdditionalPage] = useState(false);
  const loadMore = useMutation({
    mutationFn: (cursor: string) =>
      listAccountTransactions(accountID, { cursor, pageSize: 20 }),
    onSuccess: (page) => {
      setAdditionalTransactions((current) => [
        ...current,
        ...page.transactions,
      ]);
      setNextCursor(page.next_cursor);
      setLoadedAdditionalPage(true);
    },
  });
  const [copied, setCopied] = useState(false);

  if (account.isPending) return <p role="status">Loading account…</p>;
  if (account.isError)
    return <ErrorPanel message="We could not find this account." />;
  return (
    <div>
      <BackLink href="/app/accounts" label="Accounts" />
      <div className="border-line-200 mt-6 flex flex-col gap-6 border-b pb-8 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-4xl font-semibold">
              {accountLabel(account.data)}
            </h1>
            <LifecycleStatus status={account.data.status} />
          </div>
          <div className="mt-3 flex items-center gap-2">
            <code className="text-ink-700">{account.data.account_number}</code>
            <button
              aria-label="Copy account number"
              className="grid min-h-11 min-w-11 place-items-center"
              onClick={async () => {
                await navigator.clipboard.writeText(
                  account.data.account_number,
                );
                setCopied(true);
              }}
              type="button"
            >
              <Copy aria-hidden="true" className="size-4" />
            </button>
            <span aria-live="polite" className="text-success-700 text-sm">
              {copied ? "Copied" : ""}
            </span>
          </div>
        </div>
        {account.data.status !== "closed" ? (
          <Button asChild variant="secondary">
            <Link href={`/app/accounts/${accountID}/close` as Route}>
              Close account
            </Link>
          </Button>
        ) : null}
      </div>
      <section className="mt-8">
        <p className="text-ink-600 text-xs font-semibold tracking-wider uppercase">
          Current posted balance
        </p>
        <p className="mt-2 text-4xl font-semibold" data-financial-number>
          {formatMinorAmount(account.data.balance, account.data.currency)}
        </p>
      </section>
      <section className="mt-12">
        <h2 className="text-xl font-semibold">Transaction history</h2>
        {history.isPending ? (
          <p className="mt-5" role="status">
            Loading transactions…
          </p>
        ) : history.isError ? (
          <ErrorPanel
            message={bankingErrorMessage(history.error)}
            onRetry={() => void history.refetch()}
          />
        ) : history.data.transactions.length === 0 ? (
          <p className="border-line-200 text-ink-600 mt-4 border-y py-8">
            No transactions have been recorded for this account.
          </p>
        ) : (
          <>
            <TransactionList
              transactions={[
                ...history.data.transactions,
                ...additionalTransactions,
              ]}
            />
            {(loadedAdditionalPage ? nextCursor : history.data.next_cursor) ? (
              <Button
                className="mt-5"
                loading={loadMore.isPending}
                onClick={() =>
                  loadMore.mutate(
                    (loadedAdditionalPage
                      ? nextCursor
                      : history.data.next_cursor)!,
                  )
                }
                variant="secondary"
              >
                Load more transactions
              </Button>
            ) : null}
            {loadMore.isError ? (
              <ErrorPanel
                message={bankingErrorMessage(loadMore.error)}
                onRetry={() => {
                  const cursor = loadedAdditionalPage
                    ? nextCursor
                    : history.data.next_cursor;
                  if (cursor) loadMore.mutate(cursor);
                }}
              />
            ) : null}
          </>
        )}
      </section>
    </div>
  );
}

export function CloseAccountPage({ accountID }: { accountID: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const account = useQuery({
    queryFn: () => getOwnedAccount(accountID),
    queryKey: queryKeys.accounts.detail(accountID),
  });
  const [error, setError] = useState<string>();
  const close = useMutation({
    mutationFn: () => closeOwnedAccount(accountID),
    onError: (cause) => setError(bankingErrorMessage(cause)),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all });
      await queryClient.invalidateQueries({
        queryKey: queryKeys.accounts.detail(accountID),
      });
      router.replace(`/app/accounts/${accountID}` as Route);
    },
  });
  if (account.isPending) return <p role="status">Loading account…</p>;
  if (account.isError)
    return <ErrorPanel message="We could not find this account." />;
  return (
    <div className="max-w-xl">
      <BackLink href={`/app/accounts/${accountID}`} label="Account details" />
      <p className="text-danger-700 mt-8 text-sm font-bold tracking-wider uppercase">
        Permanent action
      </p>
      <h1 className="mt-2 text-4xl font-semibold">
        Close {account.data.currency} account
      </h1>
      <p className="text-ink-600 mt-4 leading-7">
        Closing prevents future transfers but preserves the account number and
        financial history. The current posted balance must be zero.
      </p>
      <dl className="border-line-200 mt-7 grid gap-4 border-y py-5">
        <div className="flex justify-between gap-4">
          <dt>Account</dt>
          <dd>{account.data.account_number}</dd>
        </div>
        <div className="flex justify-between gap-4">
          <dt>Current posted balance</dt>
          <dd className="font-semibold">
            {formatMinorAmount(account.data.balance, account.data.currency)}
          </dd>
        </div>
      </dl>
      {error ? <ErrorPanel message={error} /> : null}
      <Button
        className="mt-8 w-full"
        loading={close.isPending}
        onClick={() => close.mutate()}
        variant="danger"
      >
        Close account permanently
      </Button>
    </div>
  );
}

export function TransactionDetailPage({ reference }: { reference: string }) {
  const transaction = useQuery({
    queryFn: () => getOwnedTransaction(reference),
    queryKey: queryKeys.transactions.detail(reference),
  });
  if (transaction.isPending) return <p role="status">Loading transaction…</p>;
  if (transaction.isError)
    return <ErrorPanel message="We could not find this transaction." />;
  const value = transaction.data;
  return (
    <div className="max-w-2xl">
      <BackLink href="/app/transactions" label="Activity" />
      <div className="mt-7 flex items-center justify-between gap-4">
        <h1 className="text-4xl font-semibold">Transaction details</h1>
        <StatusBadge className="capitalize">{value.status}</StatusBadge>
      </div>
      <p className="mt-8 text-4xl font-semibold" data-financial-number>
        {formatTransactionAmount(value)}
      </p>
      <dl className="border-line-200 mt-8 grid gap-5 border-y py-6">
        <Detail
          label={
            value.direction === "incoming"
              ? "Source account"
              : "Recipient account"
          }
          value={value.counterparty}
        />
        <Detail label="Narration" value={value.narration || "None"} />
        <Detail label="Type" value={value.type.replaceAll("_", " ")} />
        <Detail label="Reference" value={value.reference} />
        <Detail
          label="Created"
          value={new Date(value.created_at).toLocaleString()}
        />
      </dl>
    </div>
  );
}

function TransactionList({
  transactions,
}: {
  transactions: BankingTransaction[];
}) {
  return (
    <ul className="border-line-200 divide-line-200 mt-4 divide-y border-y">
      {transactions.map((transaction) => (
        <li key={transaction.id}>
          <Link
            className="grid min-h-20 grid-cols-[minmax(0,1fr)_auto] items-center gap-4 py-3 no-underline"
            href={`/app/transactions/${transaction.reference}` as Route}
          >
            <div className="min-w-0">
              <p className="truncate font-semibold">
                {transaction.counterparty ||
                  transaction.type.replaceAll("_", " ")}
              </p>
              <p className="text-ink-600 truncate text-sm">
                {new Date(transaction.created_at).toLocaleString()} ·{" "}
                <span className="capitalize">{transaction.status}</span>
              </p>
            </div>
            <strong data-financial-number>
              {formatTransactionAmount(transaction)}
            </strong>
          </Link>
        </li>
      ))}
    </ul>
  );
}

function PageHeader({
  action,
  description,
  title,
}: {
  action?: React.ReactNode;
  description: string;
  title: string;
}) {
  return (
    <header className="border-line-200 flex flex-col gap-5 border-b pb-7 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h1 className="text-4xl font-semibold">{title}</h1>
        <p className="text-ink-600 mt-3">{description}</p>
      </div>
      {action}
    </header>
  );
}

function LifecycleStatus({ status }: { status: Account["status"] }) {
  return (
    <StatusBadge
      className="capitalize"
      tone={
        status === "active"
          ? "positive"
          : status === "frozen"
            ? "warning"
            : "neutral"
      }
    >
      {status}
    </StatusBadge>
  );
}

function BackLink({ href, label }: { href: string; label: string }) {
  return (
    <Link
      className="text-evergreen-800 inline-flex min-h-11 items-center gap-2 font-semibold no-underline"
      href={href as Route}
    >
      <ArrowLeft aria-hidden="true" className="size-4" />
      {label}
    </Link>
  );
}

function ErrorPanel({
  message,
  onRetry,
}: {
  message: string;
  onRetry?: () => void;
}) {
  return (
    <div
      className="border-danger-700 mt-6 rounded-sm border-l-4 bg-[#fff5f3] px-4 py-4"
      role="alert"
    >
      <p>{message}</p>
      {onRetry ? (
        <Button
          className="mt-3"
          onClick={onRetry}
          size="compact"
          variant="secondary"
        >
          <RefreshCw aria-hidden="true" className="size-4" />
          Retry
        </Button>
      ) : null}
    </div>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1 sm:grid-cols-[12rem_1fr]">
      <dt className="text-ink-600">{label}</dt>
      <dd className="font-medium break-words capitalize">{value}</dd>
    </div>
  );
}
