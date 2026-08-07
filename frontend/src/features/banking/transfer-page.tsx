"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Search } from "lucide-react";
import type { Route } from "next";
import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  createTransfer,
  resolveRecipient,
  type TransferIntent,
} from "@/features/banking/banking-api";
import { bankingErrorMessage } from "@/features/banking/banking-errors";
import {
  accountNumberSchema,
  narrationSchema,
  parseMajorAmount,
} from "@/features/banking/banking-validation";
import { listOwnedAccounts } from "@/features/dashboard/dashboard-api";
import {
  accountLabel,
  formatMinorAmount,
  maskOwnedAccountNumber,
} from "@/features/dashboard/financial-format";
import { isApiError } from "@/lib/api/api-error";
import type {
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
  const [intent, setIntent] = useState<TransferIntent>();
  const [idempotencyKey, setIdempotencyKey] = useState<string>();
  const [result, setResult] = useState<TransferResponse>();
  const sender =
    senders.find((account) => account.id === senderID) ?? senders[0];

  const resolution = useMutation({
    mutationFn: () =>
      resolveRecipient(accountNumberSchema.parse(accountNumber)),
    onError: (cause) => setError(bankingErrorMessage(cause)),
    onSuccess: (resolved) => {
      setRecipient(resolved);
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
    if (!sender || !recipient) {
      setError("Choose a sender and resolve a recipient first.");
      return;
    }
    const minorAmount = parseMajorAmount(amount);
    if (!minorAmount) {
      setError("Enter a positive amount with no more than two decimal places.");
      return;
    }
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
    return <TransferResult result={result} />;
  }

  if (stage === "uncertain") {
    return (
      <div className="max-w-xl">
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
            value={`${accountLabel(sender)} · ${maskOwnedAccountNumber(sender.account_number)}`}
          />
          <Review
            label="Recipient"
            value={`${recipient.account_name} · ${recipient.account_number}`}
          />
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
    <div className="max-w-2xl">
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
        <div className="mt-8 grid gap-6">
          <Field label="Send from" name="sender">
            <select
              className="border-line-300 min-h-11 rounded-sm border bg-white px-3"
              id="sender"
              onChange={(event) => setSenderID(event.target.value)}
              value={sender?.id}
            >
              {senders.map((account) => (
                <option key={account.id} value={account.id}>
                  {account.currency} ·{" "}
                  {maskOwnedAccountNumber(account.account_number)} ·{" "}
                  {formatMinorAmount(account.balance, account.currency)}
                </option>
              ))}
            </select>
          </Field>
          <Field
            hint="Resolution confirms current receiving eligibility but does not guarantee the later transfer."
            label="Recipient account number"
            name="recipient"
          >
            <div className="flex gap-2">
              <Input
                autoComplete="off"
                id="recipient"
                inputMode="numeric"
                maxLength={10}
                onChange={(event) => {
                  setAccountNumber(event.target.value);
                  setRecipient(undefined);
                }}
                value={accountNumber}
              />
              <Button
                loading={resolution.isPending}
                onClick={() => {
                  const parsed = accountNumberSchema.safeParse(accountNumber);
                  if (!parsed.success) {
                    setError(parsed.error.issues[0]?.message);
                    return;
                  }
                  resolution.mutate();
                }}
                variant="secondary"
              >
                <Search aria-hidden="true" className="size-4" />
                Resolve
              </Button>
            </div>
          </Field>
          {recipient ? (
            <div className="border-success-700 bg-jade-100 rounded-sm border-l-4 p-4">
              <p className="font-semibold">{recipient.account_name}</p>
              <p className="text-ink-700 mt-1">
                {recipient.account_number} · {recipient.currency} · Can receive
              </p>
            </div>
          ) : null}
          <Field
            label={`Amount${sender ? ` (${sender.currency})` : ""}`}
            name="amount"
          >
            <Input
              id="amount"
              inputMode="decimal"
              onChange={(event) => setAmount(event.target.value)}
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
      )}
    </div>
  );
}

function TransferResult({ result }: { result: TransferResponse }) {
  return (
    <div className="max-w-xl text-center">
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
