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

      {status === "unavailable" ? (
        <div className="border-warning-700 mt-6 rounded-sm border-l-4 bg-[#2a2113] px-4 py-3 text-sm text-white/80">
          We could not check your existing session. You can still sign in, or{" "}
          <button
            className="min-h-11 font-semibold text-white underline"
            onClick={() => void restore()}
            type="button"
          >
            retry the session check
          </button>
          .
        </div>
      ) : null}

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

      <p className="auth-form-note mt-6 text-sm">
        Protected by Monierave&apos;s single-device session controls.
      </p>
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
