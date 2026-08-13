"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import type { Route } from "next";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";

import { FormErrorSummary } from "@/components/auth/form-error-summary";
import { PasswordInput } from "@/components/auth/password-input";
import { Button } from "@/components/ui/button";
import { Field, fieldDescriptionIDs } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { authErrorMessage } from "@/features/auth/auth-errors";
import { useAuth } from "@/features/auth/auth-provider";
import {
  loginSchema,
  safeReturnPath,
  type LoginValues,
} from "@/features/auth/auth-schemas";
import type { User } from "@/lib/api/contracts";

export function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { login, restore, status, user } = useAuth();
  const summaryRef = useRef<HTMLDivElement>(null);
  const [submitError, setSubmitError] = useState<string>();
  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
  } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { password: "", username: "" },
  });

  const returnTo = safeReturnPath(searchParams.get("returnTo"));

  useEffect(() => {
    if (status === "authenticated" && user) {
      router.replace(destinationFor(user, returnTo) as Route);
    }
  }, [returnTo, router, status, user]);

  useEffect(() => {
    if (submitError) {
      summaryRef.current?.focus();
    }
  }, [submitError]);

  async function onSubmit(values: LoginValues) {
    setSubmitError(undefined);
    try {
      const currentUser = await login(loginSchema.parse(values));
      router.replace(destinationFor(currentUser, returnTo) as Route);
    } catch (error) {
      setSubmitError(authErrorMessage(error, "login"));
    }
  }

  if (status === "restoring") {
    return <SessionCheck />;
  }

  if (status === "unavailable") {
    return <AuthenticationUnavailable onRetry={() => void restore()} />;
  }

  return (
    <div className="auth-form auth-form-enter">
      <h1 className="auth-form-heading">Welcome back</h1>
      <p className="auth-form-switch mt-3 text-base">
        New to Monierave? <Link href="/signup">Create an account.</Link>
      </p>
      <p className="sr-only">
        Access your accounts, recent activity, and secure transfer tools from
        one place.
      </p>

      <form
        className="mt-8 grid gap-5"
        noValidate
        onSubmit={handleSubmit(onSubmit)}
      >
        <FormErrorSummary message={submitError} ref={summaryRef} />
        <Field
          error={errors.username?.message}
          label="Username"
          name="username"
        >
          <Input
            autoCapitalize="none"
            autoComplete="username"
            aria-describedby={fieldDescriptionIDs({
              error: errors.username?.message,
              name: "username",
            })}
            aria-invalid={Boolean(errors.username)}
            id="username"
            placeholder="Enter your username"
            {...register("username")}
          />
        </Field>
        <Field
          error={errors.password?.message}
          label="Password"
          name="password"
        >
          <PasswordInput
            autoComplete="current-password"
            aria-describedby={fieldDescriptionIDs({
              error: errors.password?.message,
              name: "password",
            })}
            aria-invalid={Boolean(errors.password)}
            id="password"
            placeholder="Enter your password"
            {...register("password")}
          />
        </Field>
        <Button
          className="auth-submit mt-1 w-full"
          loading={isSubmitting}
          type="submit"
        >
          Sign in
        </Button>
      </form>
    </div>
  );
}

function AuthenticationUnavailable({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="auth-form auth-form-enter text-center">
      <p className="text-evergreen-700 text-sm font-bold tracking-[0.14em] uppercase">
        Connection interrupted
      </p>
      <h1 className="auth-form-heading mt-3">
        We’re having trouble connecting to Monierave.
      </h1>
      <p className="auth-form-switch mx-auto mt-4 max-w-md text-base leading-7">
        Your account has not been changed. Check your connection and try again
        shortly.
      </p>
      <div className="mt-7 grid gap-3 sm:grid-cols-2">
        <Button onClick={onRetry}>Try again</Button>
        <Button
          asChild
          className="border-white/14 bg-white/[0.04] text-white/78 hover:bg-white/[0.08] hover:text-white"
          variant="secondary"
        >
          <Link href="/">Return home</Link>
        </Button>
      </div>
    </div>
  );
}

function destinationFor(user: User, returnTo: string | null) {
  if (user.account_status !== "active" || !user.email_verified_at) {
    return "/app";
  }
  return returnTo ?? "/app";
}

function SessionCheck() {
  return (
    <div
      aria-live="polite"
      className="grid gap-5 text-center text-white/60"
      role="status"
    >
      <div className="mx-auto h-4 w-28 animate-pulse rounded bg-white/8 motion-reduce:animate-none" />
      <div className="mx-auto h-14 w-4/5 animate-pulse rounded bg-white/8 motion-reduce:animate-none" />
      <div className="h-5 w-full animate-pulse rounded bg-white/8 motion-reduce:animate-none" />
      <p className="text-sm">Checking your secure session…</p>
    </div>
  );
}
