"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangle,
  Check,
  Clock3,
  LogOut,
  MailWarning,
  RefreshCw,
} from "lucide-react";
import type { Route } from "next";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";

import { FormErrorSummary } from "@/components/auth/form-error-summary";
import { Button } from "@/components/ui/button";
import { Field, fieldDescriptionIDs } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { authErrorMessage } from "@/features/auth/auth-errors";
import {
  getEmailStatus,
  requestVerificationEmail,
} from "@/features/auth/auth-api";
import { useAuth } from "@/features/auth/auth-provider";
import {
  emailUpdateSchema,
  type EmailUpdateValues,
} from "@/features/auth/auth-schemas";
import { queryKeys } from "@/lib/query/query-keys";

export function VerificationNeeded() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { logout, status, updateUser, user } = useAuth();
  const [notice, setNotice] = useState<string>();
  const summaryRef = useRef<HTMLDivElement>(null);

  const emailStatus = useQuery({
    enabled: status === "authenticated",
    queryFn: getEmailStatus,
    queryKey: queryKeys.emailStatus,
    retry: 1,
  });

  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
    reset,
  } = useForm<EmailUpdateValues>({
    resolver: zodResolver(emailUpdateSchema),
    values: { email: user?.email ?? "" },
  });

  const resend = useMutation({
    mutationFn: requestVerificationEmail,
    onError: (error) => {
      setNotice(authErrorMessage(error, "resend"));
    },
    onSuccess: async () => {
      setNotice("A fresh verification email is on its way.");
      await queryClient.invalidateQueries({ queryKey: queryKeys.emailStatus });
    },
  });

  useEffect(() => {
    if (status === "anonymous") {
      router.replace("/login?returnTo=%2Fverification-needed" as Route);
    }
    if (
      status === "authenticated" &&
      user?.account_status === "active" &&
      user.email_verified_at
    ) {
      router.replace("/app" as Route);
    }
  }, [router, status, user]);

  useEffect(() => {
    if (notice) {
      summaryRef.current?.focus();
    }
  }, [notice]);

  async function onEmailUpdate(values: EmailUpdateValues) {
    setNotice(undefined);
    try {
      const parsed = emailUpdateSchema.parse(values);
      const updatedUser = await updateUser({ email: parsed.email });
      reset({ email: updatedUser.email });
      setNotice(
        "Email updated. We queued a new verification message for this address.",
      );
      await queryClient.invalidateQueries({ queryKey: queryKeys.emailStatus });
    } catch (error) {
      setNotice(authErrorMessage(error, "update-email"));
    }
  }

  async function onLogout() {
    try {
      await logout();
    } finally {
      router.replace("/login" as Route);
    }
  }

  if (status === "restoring" || status === "anonymous") {
    return <VerificationSkeleton />;
  }

  if (status === "unavailable") {
    return (
      <section>
        <h1 className="font-serif text-4xl font-semibold">
          We could not restore your session.
        </h1>
        <p className="text-ink-600 mt-4 leading-7">
          Return to sign in and try again. Your registration has not been
          changed.
        </p>
        <Button
          className="mt-7 w-full"
          onClick={() => router.replace("/login" as Route)}
        >
          Return to sign in
        </Button>
      </section>
    );
  }

  const currentStatus = emailStatus.data;
  const disabled =
    user?.account_status === "disabled" ||
    currentStatus?.account_status === "disabled";

  return (
    <section className="auth-form-enter">
      <div
        className={`grid size-14 place-items-center rounded-full ${
          disabled
            ? "text-warning-700 bg-[#fff0dd]"
            : "bg-jade-100 text-evergreen-800"
        }`}
      >
        {disabled ? (
          <AlertTriangle aria-hidden="true" className="size-7" />
        ) : (
          <MailWarning aria-hidden="true" className="size-7" />
        )}
      </div>
      <p className="text-evergreen-700 mt-6 text-sm font-bold tracking-[0.16em] uppercase">
        {disabled ? "Registration needs attention" : "One step remains"}
      </p>
      <h1 className="mt-3 font-serif text-4xl leading-tight font-semibold tracking-[-0.035em]">
        {disabled ? "Update your email to recover." : "Verify your email."}
      </h1>
      <p className="text-ink-600 mt-4 leading-7">
        {disabled
          ? "The registration grace period ended before verification. Entering a valid email starts a new recovery period and sends a fresh link."
          : "Your profile is saved, but financial features stay unavailable until your email address is confirmed."}
      </p>

      <div className="border-line-200 bg-paper-50 mt-7 grid gap-4 rounded-md border p-5">
        <StatusRow
          icon={<Clock3 aria-hidden="true" className="size-5" />}
          label="Each verification link"
          value="Valid for 24 hours"
        />
        <StatusRow
          icon={<RefreshCw aria-hidden="true" className="size-5" />}
          label="Registration recovery"
          value={
            currentStatus?.registration_expires_at
              ? `Available until ${formatDateTime(
                  currentStatus.registration_expires_at,
                )}`
              : "Status unavailable"
          }
        />
        <StatusRow
          icon={<Check aria-hidden="true" className="size-5" />}
          label="Current address"
          value={currentStatus?.email ?? user?.email ?? "Loading…"}
        />
      </div>

      {currentStatus?.latest_job ? (
        <p className="text-ink-600 mt-3 text-sm">
          Latest email status:{" "}
          <strong className="text-ink-950 capitalize">
            {humanize(currentStatus.latest_job.delivery_status)}
          </strong>
        </p>
      ) : null}

      <FormErrorSummary message={notice} ref={summaryRef} />

      {emailStatus.isError ? (
        <div className="border-warning-700 mt-6 rounded-sm border-l-4 bg-[#fff8e8] px-4 py-3 text-sm">
          We could not load the latest delivery status. You can retry without
          changing your registration.
          <Button
            className="mt-3"
            onClick={() => void emailStatus.refetch()}
            size="compact"
            variant="secondary"
          >
            Retry status
          </Button>
        </div>
      ) : null}

      <div className="mt-7 grid gap-3 sm:grid-cols-2">
        <Button
          loading={resend.isPending}
          onClick={() => {
            setNotice(undefined);
            resend.mutate();
          }}
        >
          Resend verification
        </Button>
        <Button onClick={() => void onLogout()} variant="secondary">
          <LogOut aria-hidden="true" className="size-4" />
          Sign out
        </Button>
      </div>

      <div className="border-line-200 mt-8 border-t pt-7">
        <h2 className="font-serif text-2xl font-semibold">
          Need to use a different email?
        </h2>
        <p className="text-ink-600 mt-2 text-sm leading-6">
          Updating the address resets its delivery status and automatically
          queues a new verification message.
        </p>
        <form
          className="mt-5 grid gap-4"
          noValidate
          onSubmit={handleSubmit(onEmailUpdate)}
        >
          <Field
            error={errors.email?.message}
            label="Email address"
            name="recovery_email"
          >
            <Input
              autoCapitalize="none"
              autoComplete="email"
              aria-describedby={fieldDescriptionIDs({
                error: errors.email?.message,
                name: "recovery_email",
              })}
              aria-invalid={Boolean(errors.email)}
              id="recovery_email"
              inputMode="email"
              type="email"
              {...register("email")}
            />
          </Field>
          <Button
            className="sm:justify-self-start"
            loading={isSubmitting}
            type="submit"
            variant="secondary"
          >
            Update email
          </Button>
        </form>
      </div>

      {currentStatus?.restricted_features?.length ? (
        <div className="mt-8">
          <h2 className="text-sm font-bold tracking-[0.12em] uppercase">
            Unavailable until verified
          </h2>
          <ul className="text-ink-600 mt-3 grid gap-2 text-sm">
            {currentStatus.restricted_features.map((feature) => (
              <li className="flex gap-2" key={feature}>
                <span aria-hidden="true">•</span>
                {feature}
              </li>
            ))}
          </ul>
        </div>
      ) : (
        <p className="text-ink-600 mt-8 text-sm leading-6">
          Creating accounts, transfers, beneficiaries, and other financial
          actions remain unavailable until verification.
        </p>
      )}
    </section>
  );
}

function StatusRow({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="grid grid-cols-[1.5rem_1fr] gap-x-3">
      <span className="text-evergreen-700 mt-0.5">{icon}</span>
      <span className="text-ink-600 text-xs font-semibold tracking-wide uppercase">
        {label}
      </span>
      <strong className="col-start-2 mt-0.5 font-semibold break-words">
        {value}
      </strong>
    </div>
  );
}

function VerificationSkeleton() {
  return (
    <div aria-live="polite" className="grid gap-5" role="status">
      <div className="bg-paper-100 size-14 animate-pulse rounded-full motion-reduce:animate-none" />
      <div className="bg-paper-100 h-12 w-4/5 animate-pulse rounded motion-reduce:animate-none" />
      <div className="bg-paper-100 h-24 w-full animate-pulse rounded motion-reduce:animate-none" />
      <p className="text-ink-600 text-sm">Restoring verification status…</p>
    </div>
  );
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "the date shown in your latest email";
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function humanize(value: string) {
  return value.replaceAll("_", " ");
}
