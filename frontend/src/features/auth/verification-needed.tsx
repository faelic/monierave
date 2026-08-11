"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangle,
  Check,
  ChevronDown,
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
import { PasswordInput } from "@/components/auth/password-input";
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

import styles from "./verification-needed.module.css";

type VerificationNotice = {
  message: string;
  tone: "error" | "success";
};

export function VerificationNeeded() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { logout, refreshUser, status, updateUser, user } = useAuth();
  const [notice, setNotice] = useState<VerificationNotice>();
  const summaryRef = useRef<HTMLDivElement>(null);
  const verificationHandled = useRef(false);

  const emailStatus = useQuery({
    enabled: status === "authenticated",
    queryFn: getEmailStatus,
    queryKey: queryKeys.emailStatus,
    refetchInterval: (query) => (query.state.data?.verified_at ? false : 5_000),
    retry: 1,
  });

  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
  } = useForm<EmailUpdateValues>({
    resolver: zodResolver(emailUpdateSchema),
    values: { current_password: "", email: user?.email ?? "" },
  });

  const resend = useMutation({
    mutationFn: requestVerificationEmail,
    onError: (error) => {
      setNotice({
        message:
          authErrorMessage(error, "resend") ??
          "We could not resend the verification email. Please try again.",
        tone: "error",
      });
    },
    onSuccess: async () => {
      setNotice({
        message: "A fresh verification email is on its way.",
        tone: "success",
      });
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

  useEffect(() => {
    if (
      !emailStatus.data?.verified_at ||
      verificationHandled.current ||
      status !== "authenticated"
    ) {
      return;
    }

    verificationHandled.current = true;
    void refreshUser()
      .then(() => router.replace("/app" as Route))
      .catch(() => {
        verificationHandled.current = false;
        setNotice({
          message:
            "Your email is verified. Refresh the page to continue into Monierave.",
          tone: "success",
        });
      });
  }, [emailStatus.data?.verified_at, refreshUser, router, status]);

  async function onEmailUpdate(values: EmailUpdateValues) {
    setNotice(undefined);
    try {
      const parsed = emailUpdateSchema.parse(values);
      await updateUser({
        current_password: parsed.current_password,
        email: parsed.email,
      });
      router.replace("/signup/check-email" as Route);
    } catch (error) {
      setNotice({
        message:
          authErrorMessage(error, "update-email") ??
          "We could not update the email address. Please try again.",
        tone: "error",
      });
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
      <section className="auth-form">
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
    <section className={`auth-form auth-form-enter ${styles.page}`}>
      <div className={styles.intro}>
        <div
          className={`${styles.icon} ${disabled ? styles.iconAttention : ""}`}
        >
          {disabled ? (
            <AlertTriangle aria-hidden="true" className="size-6" />
          ) : (
            <MailWarning aria-hidden="true" className="size-6" />
          )}
        </div>
        <p className={styles.eyebrow}>
          {disabled ? "Registration needs attention" : "One step remains"}
        </p>
        <h1 className={styles.title}>
          {disabled ? "Update your email to recover." : "Verify your email."}
        </h1>
        <p className={styles.description}>
          {disabled
            ? "The registration grace period ended before verification. Entering a valid email starts a new recovery period and sends a fresh link."
            : "Your profile is saved. Confirm your email to unlock accounts, beneficiaries, and transfers."}
        </p>
      </div>

      <div className={styles.statusPanel}>
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
        <p className={styles.delivery}>
          <span>Latest verification email</span>
          <strong className={styles.deliveryBadge}>
            {humanize(currentStatus.latest_job.delivery_status)}
          </strong>
        </p>
      ) : null}

      <FormErrorSummary
        className={styles.notice}
        message={notice?.message}
        ref={summaryRef}
        tone={notice?.tone}
      />

      {emailStatus.isError ? (
        <div className={styles.warning} role="alert">
          <span>
            We could not refresh the delivery status. Your registration is
            unchanged.
          </span>
          <Button
            className={styles.retryButton}
            onClick={() => void emailStatus.refetch()}
            size="compact"
            variant="secondary"
          >
            Retry status
          </Button>
        </div>
      ) : null}

      <div className={styles.actions}>
        <Button
          className={styles.primaryAction}
          loading={resend.isPending}
          onClick={() => {
            setNotice(undefined);
            resend.mutate();
          }}
        >
          Resend verification
        </Button>
        <Button
          className={styles.secondaryAction}
          onClick={() => void onLogout()}
          variant="secondary"
        >
          <LogOut aria-hidden="true" className="size-4" />
          Sign out
        </Button>
      </div>

      <details className={styles.recovery} open={disabled || undefined}>
        <summary className={styles.recoverySummary}>
          <span className={styles.summaryText}>
            Use a different email
            <span className={styles.summaryHint}>
              Update the address and send a fresh link
            </span>
          </span>
          <ChevronDown aria-hidden="true" className={styles.chevron} />
        </summary>
        <div className={styles.recoveryBody}>
          <p className={styles.recoveryCopy}>
            Changing your email resets its delivery status and starts a new
            verification attempt.
          </p>
          <form
            className={styles.recoveryForm}
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
            <Field
              error={errors.current_password?.message}
              hint="Changing your email signs you out on every device."
              label="Current password"
              name="recovery_current_password"
            >
              <PasswordInput
                autoComplete="current-password"
                aria-describedby={fieldDescriptionIDs({
                  error: errors.current_password?.message,
                  name: "recovery_current_password",
                })}
                aria-invalid={Boolean(errors.current_password)}
                id="recovery_current_password"
                {...register("current_password")}
              />
            </Field>
            <Button
              className={styles.updateButton}
              loading={isSubmitting}
              type="submit"
              variant="secondary"
            >
              Update email
            </Button>
          </form>
        </div>
      </details>

      {currentStatus?.restricted_features?.length ? (
        <p className={styles.restrictionNote}>
          Until verification,{" "}
          {formatFeatureList(currentStatus.restricted_features)}.
        </p>
      ) : (
        <p className={styles.restrictionNote}>
          Accounts, transfers, and beneficiaries remain unavailable until
          verification.
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
    <div className={styles.statusRow}>
      <span className={styles.statusIcon}>{icon}</span>
      <span className={styles.statusLabel}>{label}</span>
      <strong className={styles.statusValue}>{value}</strong>
    </div>
  );
}

function formatFeatureList(features: string[]) {
  const normalized = features.map((feature) =>
    feature
      .replace(/^Create and manage /i, "")
      .replace(/^Send and receive /i, ""),
  );
  if (normalized.length === 1) {
    return `${normalized[0]?.toLowerCase() ?? "financial features"} remain unavailable`;
  }
  if (normalized.length === 2) {
    return `${normalized[0]?.toLowerCase()} and ${normalized[1]?.toLowerCase()} remain unavailable`;
  }
  const finalFeature = normalized.at(-1)?.toLowerCase();
  return `${normalized
    .slice(0, -1)
    .map((feature) => feature.toLowerCase())
    .join(", ")}, and ${finalFeature} remain unavailable`;
}

function VerificationSkeleton() {
  return (
    <div aria-live="polite" className="auth-form grid gap-5" role="status">
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
