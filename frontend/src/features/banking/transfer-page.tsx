"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Search } from "lucide-react";
import type { Route } from "next";
import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Field, fieldDescriptionIDs } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  createTransfer,
  resolveRecipient,
  type TransferIntent,
} from "@/features/banking/banking-api";
import { bankingErrorMessage } from "@/features/banking/banking-errors";
import {
  accountNumberSchema,
  majorAmountError,
  narrationSchema,
  normalizeAccountNumberInput,
  normalizeMajorAmountInput,
  parseMajorAmount,
} from "@/features/banking/banking-validation";
import { RecipientIdentity } from "@/features/banking/recipient-identity";
import { listOwnedAccounts } from "@/features/dashboard/dashboard-api";
import {
  accountLabel,
  formatMinorAmount,
} from "@/features/dashboard/financial-format";
import { isApiError } from "@/lib/api/api-error";
import type {
  Account,
  RecipientResolution,
  TransferResponse,
} from "@/lib/api/contracts";
import { createIdempotencyKey } from "@/lib/api/idempotency";
import { queryKeys } from "@/lib/query/query-keys";

type Stage = "prepare" | "review" | "result" | "uncertain";

export function TransferPage() {
  const queryClient = useQueryClient();
  const accounts = useQuery({
    queryFn: listOwnedAccounts,
    queryKey: queryKeys.accounts.all,
  });
  const senders =
    accounts.data?.filter((account) => account.status === "active") ?? [];
  const [senderID, setSenderID] = useState("");
  const [accountNumber, setAccountNumber] = useState("");
  const [recipient, setRecipient] = useState<RecipientResolution>();
  const [amount, setAmount] = useState("");
  const [narration, setNarration] = useState("");
  const [stage, setStage] = useState<Stage>("prepare");
  const [error, setError] = useState<string>();
  const [recipientError, setRecipientError] = useState<string>();
  const [amountError, setAmountError] = useState<string>();
  const [intent, setIntent] = useState<TransferIntent>();
  const [idempotencyKey, setIdempotencyKey] = useState<string>();
  const [result, setResult] = useState<TransferResponse>();
  const sender =
    senders.find((account) => account.id === senderID) ?? senders[0];

  const resolution = useMutation({
    mutationFn: () =>
      resolveRecipient(accountNumberSchema.parse(accountNumber)),
    onError: (cause) => {
      setRecipientError(bankingErrorMessage(cause));
      setError(undefined);
    },
    onSuccess: (resolved) => {
      setRecipient(resolved);
      setRecipientError(undefined);
      setError(undefined);
    },
  });

  const transfer = useMutation({
    mutationFn: ({
      key,
      transferIntent,
    }: {
      key: string;
      transferIntent: TransferIntent;
    }) => createTransfer(transferIntent, key),
    onError: (cause) => {
      setError(bankingErrorMessage(cause));
      setStage(
        isApiError(cause) && cause.code === "network_error"
          ? "uncertain"
          : "review",
      );
    },
    onSuccess: async (response) => {
      setResult(response);
      setStage("result");
      await queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all });
      await queryClient.invalidateQueries({
        queryKey: queryKeys.accounts.detail(response.from_account.id),
      });
    },
  });

  function prepareReview() {
    setError(undefined);
    if (!sender) {
      setError("Choose the account you want to send from.");
      return;
    }
    if (!recipient) {
      setRecipientError("Search for a recipient before continuing.");
      return;
    }
    const validationMessage = majorAmountError(amount);
    setAmountError(validationMessage);
    if (validationMessage) return;
    const minorAmount = parseMajorAmount(amount);
    if (!minorAmount) return;
    const parsedNarration = narrationSchema.safeParse(narration);
    if (!parsedNarration.success) {
      setError(parsedNarration.error.issues[0]?.message);
      return;
    }
    if (sender.currency !== recipient.currency) {
      setError(
        "Choose a sender account with the same currency as the recipient.",
      );
      return;
    }
    const nextIntent: TransferIntent = {
      amount: minorAmount,
      currency: sender.currency,
      from_account_id: sender.id,
      narration: parsedNarration.data,
      to_account_number: accountNumber.trim(),
    };
    setIntent(nextIntent);
    setIdempotencyKey(createIdempotencyKey());
    setStage("review");
  }

  function submitTransfer() {
    if (intent && idempotencyKey) {
      transfer.mutate({ key: idempotencyKey, transferIntent: intent });
    }
  }

  if (stage === "result" && result) {
    return <TransferResult recipient={recipient} result={result} />;
  }

  if (stage === "uncertain") {
    return (
      <div className="max-w-2xl">
        <TransferProgress stage="uncertain" />
        <p className="text-warning-700 text-sm font-bold uppercase">
          Checking result
        </p>
        <h1 className="mt-2 text-4xl font-semibold">
          We could not confirm the transfer result yet.
        </h1>
        <p className="text-ink-600 mt-4 leading-7">
          Do not create another transfer. Retry the original request with the
          same protected key to learn whether it was already posted.
        </p>
        {error ? <Message message={error} /> : null}
        <Button
          className="mt-7"
          loading={transfer.isPending}
          onClick={submitTransfer}
        >
          Check original transfer
        </Button>
      </div>
    );
  }

  if (stage === "review" && intent && sender && recipient) {
    return (
      <div className="max-w-2xl">
        <TransferProgress stage="review" />
        <p className="text-evergreen-700 text-sm font-bold uppercase">
          Review transfer
        </p>
        <h1 className="mt-2 text-4xl font-semibold">Check every detail.</h1>
        <p className="text-ink-600 mt-3">
          Account status, balance, currency, and transfer limits are checked
          again when you send.
        </p>
        <dl className="border-line-200 mt-8 grid gap-5 border-y py-6">
          <Review
            label="From"
            value={`${accountLabel(sender)} · ${sender.account_number}`}
          />
          <Review label="Recipient name" value={recipient.account_name} />
          <Review label="Recipient account" value={recipient.account_number} />
          <Review
            label="Amount"
            value={formatMinorAmount(intent.amount, intent.currency)}
          />
          <Review label="Fee" value={formatMinorAmount(0, intent.currency)} />
          <Review
            label="Total debit"
            value={formatMinorAmount(intent.amount, intent.currency)}
          />
          <Review label="Narration" value={intent.narration || "None"} />
        </dl>
        {error ? <Message message={error} /> : null}
        <div className="mt-7 flex flex-col gap-3 sm:flex-row">
          <Button loading={transfer.isPending} onClick={submitTransfer}>
            Send {formatMinorAmount(intent.amount, intent.currency)}
          </Button>
          <Button
            onClick={() => {
              setStage("prepare");
              setIntent(undefined);
              setIdempotencyKey(undefined);
              setError(undefined);
            }}
            variant="secondary"
          >
            Edit transfer
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <TransferProgress stage="prepare" />
      <h1 className="text-4xl font-semibold">Send money</h1>
      <p className="text-ink-600 mt-3 leading-7">
        Internal Monierave transfers currently have no fee.
      </p>
      {error ? <Message message={error} /> : null}
      {accounts.isPending ? (
        <p className="mt-8" role="status">
          Loading sender accounts…
        </p>
      ) : senders.length === 0 ? (
        <Message message="You need an active account before you can send money." />
      ) : (
        <div className="mt-8 grid items-start gap-8 lg:grid-cols-[minmax(0,2fr)_minmax(17rem,1fr)]">
          <div className="grid gap-6">
            <Field label="Send from" name="sender">
              <select
                className="border-line-300 min-h-11 w-full rounded-sm border bg-white px-3"
                id="sender"
                onChange={(event) => setSenderID(event.target.value)}
                value={sender?.id}
              >
                {senders.map((account) => (
                  <option key={account.id} value={account.id}>
                    {account.currency} · {account.account_number} ·{" "}
                    {formatMinorAmount(account.balance, account.currency)}
                  </option>
                ))}
              </select>
            </Field>
            <Field
              error={recipientError}
              hint="Searching confirms current receiving eligibility but does not guarantee the later transfer."
              label="Recipient account number"
              name="recipient"
            >
              <div className="flex gap-2">
                <Input
                  autoComplete="off"
                  aria-describedby={fieldDescriptionIDs({
                    error: recipientError,
                    hint: "search guidance",
                    name: "recipient",
                  })}
                  aria-invalid={Boolean(recipientError)}
                  id="recipient"
                  inputMode="numeric"
                  maxLength={10}
                  onChange={(event) => {
                    setAccountNumber(
                      normalizeAccountNumberInput(event.target.value),
                    );
                    setRecipient(undefined);
                    setRecipientError(undefined);
                  }}
                  pattern="[0-9]*"
                  placeholder="10-digit account number"
                  value={accountNumber}
                />
                <Button
                  loading={resolution.isPending}
                  onClick={() => {
                    const parsed = accountNumberSchema.safeParse(accountNumber);
                    if (!parsed.success) {
                      setRecipientError(parsed.error.issues[0]?.message);
                      return;
                    }
                    setRecipientError(undefined);
                    resolution.mutate();
                  }}
                  variant="secondary"
                >
                  <Search aria-hidden="true" className="size-4" />
                  Search
                </Button>
              </div>
            </Field>
            {recipient ? (
              <RecipientIdentity
                accountName={recipient.account_name}
                accountNumber={recipient.account_number}
                canReceive={recipient.can_receive}
                currency={recipient.currency}
              />
            ) : null}
            <Field
              error={amountError}
              label={`Amount${sender ? ` (${sender.currency})` : ""}`}
              name="amount"
            >
              <Input
                aria-describedby={fieldDescriptionIDs({
                  error: amountError,
                  name: "amount",
                })}
                aria-invalid={Boolean(amountError)}
                id="amount"
                inputMode="decimal"
                onChange={(event) => {
                  setAmount(normalizeMajorAmountInput(event.target.value));
                  setAmountError(undefined);
                }}
                pattern="[0-9]*[.]?[0-9]{0,2}"
                placeholder="0.00"
                value={amount}
              />
            </Field>
            <Field label="Narration" name="narration" optional>
              <Input
                id="narration"
                maxLength={255}
                onChange={(event) => setNarration(event.target.value)}
                value={narration}
              />
            </Field>
            <Button onClick={prepareReview}>Continue to review</Button>
          </div>
          <TransferSummary
            amount={amount}
            recipient={recipient}
            sender={sender}
          />
        </div>
      )}
    </div>
  );
}

function TransferResult({
  recipient,
  result,
}: {
  recipient: RecipientResolution | undefined;
  result: TransferResponse;
}) {
  return (
    <div className="max-w-xl text-center">
      <TransferProgress stage="result" />
      <CheckCircle2
        aria-hidden="true"
        className="text-success-700 mx-auto size-14"
      />
      <p className="text-success-700 mt-5 text-sm font-bold uppercase">
        {result.transaction.status}
      </p>
      <h1 className="mt-2 text-4xl font-semibold">Transfer posted.</h1>
      <p className="mt-4 text-3xl font-semibold" data-financial-number>
        {formatMinorAmount(
          result.transaction.amount,
          result.transaction.currency,
        )}
      </p>
      <p className="text-ink-600 mt-3">
        Fee: {formatMinorAmount(0, result.transaction.currency)}
      </p>
      <p className="text-ink-600 mt-1">
        Reference: {result.transaction.reference}
      </p>
      <RecipientIdentity
        accountName={recipient?.account_name ?? "Recipient"}
        accountNumber={result.to_account.account_number}
        className="mt-7 text-left"
        currency={result.to_account.currency}
      />
      <Button asChild className="mt-7">
        <Link
          href={`/app/transactions/${result.transaction.reference}` as Route}
        >
          View transaction
        </Link>
      </Button>
    </div>
  );
}

function TransferProgress({ stage }: { stage: Stage }) {
  const active =
    stage === "prepare"
      ? 1
      : stage === "review"
        ? 2
        : stage === "result"
          ? 3
          : 2;
  return (
    <ol
      aria-label="Transfer progress"
      className="border-line-200 mb-8 grid grid-cols-3 border-b pb-5"
    >
      {["Details", "Review", "Result"].map((label, index) => {
        const step = index + 1;
        const reached = step <= active;
        return (
          <li
            aria-current={step === active ? "step" : undefined}
            className={`flex items-center gap-2 text-sm font-semibold ${
              reached ? "text-ink-950" : "text-ink-600"
            }`}
            key={label}
          >
            <span
              className={`grid size-7 place-items-center rounded-full text-xs ${
                reached
                  ? "bg-[var(--product-accent)] text-[#111317]"
                  : "bg-paper-100 text-ink-600"
              }`}
            >
              {step}
            </span>
            <span className="hidden sm:inline">{label}</span>
          </li>
        );
      })}
    </ol>
  );
}

function TransferSummary({
  amount,
  recipient,
  sender,
}: {
  amount: string;
  recipient: RecipientResolution | undefined;
  sender: Account | undefined;
}) {
  const parsedAmount = parseMajorAmount(amount);
  return (
    <aside className="border-line-200 rounded-md border bg-white p-5 lg:sticky lg:top-24">
      <p className="text-evergreen-700 text-xs font-bold tracking-[0.12em] uppercase">
        Transfer summary
      </p>
      <h2 className="mt-2 text-lg font-semibold">Before review</h2>
      <dl className="mt-5 grid gap-4 text-sm">
        <div>
          <dt className="text-ink-600">From</dt>
          <dd className="mt-1 font-semibold">
            {sender
              ? `${sender.currency} · ${sender.account_number}`
              : "Choose an account"}
          </dd>
        </div>
        <div>
          <dt className="text-ink-600">Recipient</dt>
          <dd className="mt-1 font-semibold">
            {recipient
              ? `${recipient.account_name} · ${recipient.account_number}`
              : "Search for an account number"}
          </dd>
        </div>
        <div className="border-line-200 border-t pt-4">
          <dt className="text-ink-600">Estimated total debit</dt>
          <dd className="mt-1 text-xl font-semibold" data-financial-number>
            {sender && parsedAmount
              ? formatMinorAmount(parsedAmount, sender.currency)
              : "—"}
          </dd>
          <p className="text-ink-600 mt-1">Fee: 0</p>
        </div>
      </dl>
      <p className="text-ink-600 mt-5 text-xs leading-5">
        Nothing moves until you confirm the complete transfer on the next step.
      </p>
    </aside>
  );
}

function Review({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1 sm:grid-cols-[10rem_1fr]">
      <dt className="text-ink-600">{label}</dt>
      <dd className="font-semibold break-words">{value}</dd>
    </div>
  );
}

function Message({ message }: { message: string }) {
  return (
    <div
      className="border-warning-700 mt-6 rounded-sm border-l-4 bg-[#fff8e8] p-4"
      role="alert"
    >
      {message}
    </div>
  );
}
