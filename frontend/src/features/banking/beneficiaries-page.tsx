"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  createBeneficiary,
  listBeneficiaries,
  removeBeneficiary,
  renameBeneficiary,
} from "@/features/banking/banking-api";
import { bankingErrorMessage } from "@/features/banking/banking-errors";
import {
  accountNumberSchema,
  nicknameSchema,
} from "@/features/banking/banking-validation";
import type { Beneficiary } from "@/lib/api/contracts";
import { queryKeys } from "@/lib/query/query-keys";

export function BeneficiariesPage() {
  const queryClient = useQueryClient();
  const beneficiaries = useQuery({
    queryFn: listBeneficiaries,
    queryKey: queryKeys.beneficiaries.all,
  });
  const [accountNumber, setAccountNumber] = useState("");
  const [nickname, setNickname] = useState("");
  const [editing, setEditing] = useState<Beneficiary>();
  const [removing, setRemoving] = useState<Beneficiary>();
  const [showCreate, setShowCreate] = useState(false);
  const [error, setError] = useState<string>();
  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.beneficiaries.all });
  const create = useMutation({
    mutationFn: () =>
      createBeneficiary({
        destination_account_number: accountNumberSchema.parse(accountNumber),
        nickname: nicknameSchema.parse(nickname),
      }),
    onError: (cause) => setError(bankingErrorMessage(cause)),
    onSuccess: async () => {
      setAccountNumber("");
      setNickname("");
      setError(undefined);
      setShowCreate(false);
      await invalidate();
    },
  });
  const rename = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      renameBeneficiary(id, nicknameSchema.parse(name)),
    onError: (cause) => setError(bankingErrorMessage(cause)),
    onSuccess: async () => {
      setEditing(undefined);
      await invalidate();
    },
  });
  const remove = useMutation({
    mutationFn: (id: string) => removeBeneficiary(id),
    onError: (cause) => setError(bankingErrorMessage(cause)),
    onSuccess: async () => {
      setRemoving(undefined);
      await invalidate();
    },
  });

  return (
    <div>
      <header className="border-line-200 flex flex-col gap-5 border-b pb-7 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-4xl font-semibold">Beneficiaries</h1>
          <p className="text-ink-600 mt-3 max-w-2xl">
            Save recipients for recognition and management. For security, saved
            numbers remain masked and a transfer still requires the full account
            number.
          </p>
        </div>
        <Button
          aria-expanded={showCreate}
          className="self-start sm:self-auto"
          onClick={() => setShowCreate((visible) => !visible)}
          variant={showCreate ? "secondary" : "primary"}
        >
          <Plus aria-hidden="true" className="size-4" />
          {showCreate ? "Close form" : "Add beneficiary"}
        </Button>
      </header>
      {error ? <Notice message={error} /> : null}
      {showCreate ? (
        <section className="border-line-200 mt-8 max-w-3xl rounded-md border bg-white p-5 sm:p-6">
          <p className="text-evergreen-700 text-xs font-bold tracking-[0.12em] uppercase">
            New saved recipient
          </p>
          <h2 className="mt-2 text-xl font-semibold">Add beneficiary</h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <Field label="10-digit account number" name="beneficiary-account">
              <Input
                id="beneficiary-account"
                inputMode="numeric"
                maxLength={10}
                onChange={(event) => setAccountNumber(event.target.value)}
                value={accountNumber}
              />
            </Field>
            <Field label="Nickname" name="beneficiary-nickname">
              <Input
                id="beneficiary-nickname"
                maxLength={50}
                onChange={(event) => setNickname(event.target.value)}
                value={nickname}
              />
            </Field>
          </div>
          <Button
            className="mt-4"
            loading={create.isPending}
            onClick={() => {
              const number = accountNumberSchema.safeParse(accountNumber);
              const name = nicknameSchema.safeParse(nickname);
              if (!number.success || !name.success) {
                setError(
                  number.error?.issues[0]?.message ??
                    name.error?.issues[0]?.message,
                );
                return;
              }
              create.mutate();
            }}
          >
            <Plus aria-hidden="true" className="size-4" />
            Save beneficiary
          </Button>
        </section>
      ) : null}

      <section className="mt-12">
        <h2 className="text-xl font-semibold">Saved beneficiaries</h2>
        {beneficiaries.isPending ? (
          <p className="mt-4" role="status">
            Loading beneficiaries…
          </p>
        ) : beneficiaries.isError ? (
          <Notice message={bankingErrorMessage(beneficiaries.error)} />
        ) : beneficiaries.data.length === 0 ? (
          <div className="border-line-200 mt-4 rounded-md border bg-white px-5 py-9 text-center">
            <p className="font-semibold">You have no saved beneficiaries.</p>
            <p className="text-ink-600 mt-2 text-sm">
              Save a recipient nickname so future account checks are easier to
              recognise.
            </p>
            <Button className="mt-5" onClick={() => setShowCreate(true)}>
              <Plus aria-hidden="true" className="size-4" />
              Add first beneficiary
            </Button>
          </div>
        ) : (
          <ul className="border-line-200 divide-line-200 mt-4 divide-y border-y">
            {beneficiaries.data.map((beneficiary) => (
              <li
                className="flex min-h-20 flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between"
                key={beneficiary.id}
              >
                {editing?.id === beneficiary.id ? (
                  <RenameForm
                    beneficiary={beneficiary}
                    loading={rename.isPending}
                    onCancel={() => setEditing(undefined)}
                    onSave={(name) =>
                      rename.mutate({ id: beneficiary.id, name })
                    }
                  />
                ) : removing?.id === beneficiary.id ? (
                  <div className="w-full">
                    <p className="font-semibold">
                      Remove {beneficiary.nickname} permanently?
                    </p>
                    <div className="mt-3 flex gap-2">
                      <Button
                        loading={remove.isPending}
                        onClick={() => remove.mutate(beneficiary.id)}
                        size="compact"
                        variant="danger"
                      >
                        Remove beneficiary
                      </Button>
                      <Button
                        onClick={() => setRemoving(undefined)}
                        size="compact"
                        variant="secondary"
                      >
                        Cancel
                      </Button>
                    </div>
                  </div>
                ) : (
                  <>
                    <div>
                      <p className="font-semibold">{beneficiary.nickname}</p>
                      <p className="text-ink-600 mt-1 text-sm">
                        {beneficiary.destination_account_number} ·{" "}
                        {beneficiary.currency} ·{" "}
                        {beneficiary.can_receive
                          ? "Can receive"
                          : "Cannot receive"}
                      </p>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        aria-label={`Rename ${beneficiary.nickname}`}
                        className="size-11 min-h-0 p-0"
                        onClick={() => setEditing(beneficiary)}
                        size="compact"
                        title="Rename beneficiary"
                        variant="secondary"
                      >
                        <Pencil aria-hidden="true" className="size-4" />
                      </Button>
                      <Button
                        aria-label={`Remove ${beneficiary.nickname}`}
                        className="text-danger-700 hover:border-danger-700 size-11 min-h-0 p-0 hover:bg-[#2a1210]"
                        onClick={() => setRemoving(beneficiary)}
                        size="compact"
                        title="Remove beneficiary"
                        variant="secondary"
                      >
                        <Trash2 aria-hidden="true" className="size-4" />
                      </Button>
                    </div>
                  </>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function RenameForm({
  beneficiary,
  loading,
  onCancel,
  onSave,
}: {
  beneficiary: Beneficiary;
  loading: boolean;
  onCancel: () => void;
  onSave: (name: string) => void;
}) {
  const [name, setName] = useState(beneficiary.nickname);
  return (
    <div className="flex w-full flex-col gap-2 sm:flex-row">
      <Input
        aria-label="New beneficiary nickname"
        maxLength={50}
        onChange={(event) => setName(event.target.value)}
        value={name}
      />
      <Button loading={loading} onClick={() => onSave(name)} size="compact">
        Save nickname
      </Button>
      <Button onClick={onCancel} size="compact" variant="secondary">
        Cancel
      </Button>
    </div>
  );
}

function Notice({ message }: { message: string }) {
  return (
    <div
      className="border-warning-700 mt-6 rounded-sm border-l-4 bg-[#fff8e8] p-4"
      role="alert"
    >
      {message}
    </div>
  );
}
