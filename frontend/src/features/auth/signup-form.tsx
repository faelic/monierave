"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";

import { FormErrorSummary } from "@/components/auth/form-error-summary";
import { PasswordInput } from "@/components/auth/password-input";
import { Button } from "@/components/ui/button";
import { Field, fieldDescriptionIDs } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { authErrorMessage } from "@/features/auth/auth-errors";
import { registerUser } from "@/features/auth/auth-api";
import { signupSchema, type SignupValues } from "@/features/auth/auth-schemas";

export function SignupForm() {
  const router = useRouter();
  const summaryRef = useRef<HTMLDivElement>(null);
  const [submitError, setSubmitError] = useState<string>();
  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
  } = useForm<SignupValues>({
    resolver: zodResolver(signupSchema),
    defaultValues: { email: "", full_name: "", password: "", username: "" },
  });

  useEffect(() => {
    if (submitError) {
      summaryRef.current?.focus();
    }
  }, [submitError]);

  async function onSubmit(values: SignupValues) {
    setSubmitError(undefined);
    try {
      const parsed = signupSchema.parse(values);
      await registerUser(parsed);
      router.push("/signup/check-email");
    } catch (error) {
      setSubmitError(authErrorMessage(error, "signup"));
    }
  }

  return (
    <div className="auth-form auth-form-enter">
      <h1 className="auth-form-heading">Create your Monierave account</h1>
      <p className="auth-form-switch mt-3 text-base">
        Already registered? <Link href="/login">Sign in.</Link>
      </p>

      <form
        className="auth-signup-form mt-8 grid gap-5"
        noValidate
        onSubmit={handleSubmit(onSubmit)}
      >
        <FormErrorSummary message={submitError} ref={summaryRef} />
        <Field
          error={errors.full_name?.message}
          label="Full name"
          name="full_name"
        >
          <Input
            autoComplete="name"
            aria-describedby={fieldDescriptionIDs({
              error: errors.full_name?.message,
              name: "full_name",
            })}
            aria-invalid={Boolean(errors.full_name)}
            id="full_name"
            placeholder="Enter your full name"
            {...register("full_name")}
          />
        </Field>
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
            placeholder="Choose a username"
            {...register("username")}
          />
        </Field>
        <Field error={errors.email?.message} label="Email address" name="email">
          <Input
            autoCapitalize="none"
            autoComplete="email"
            aria-describedby={fieldDescriptionIDs({
              error: errors.email?.message,
              name: "email",
            })}
            aria-invalid={Boolean(errors.email)}
            id="email"
            inputMode="email"
            placeholder="you@example.com"
            type="email"
            {...register("email")}
          />
        </Field>
        <Field
          error={errors.password?.message}
          hint="At least 8 characters."
          label="Password"
          name="password"
        >
          <PasswordInput
            autoComplete="new-password"
            aria-describedby={fieldDescriptionIDs({
              error: errors.password?.message,
              hint: "At least 8 characters.",
              name: "password",
            })}
            aria-invalid={Boolean(errors.password)}
            id="password"
            placeholder="Create a secure password"
            {...register("password")}
          />
        </Field>
        <Button
          className="auth-submit mt-1 w-full"
          loading={isSubmitting}
          type="submit"
        >
          Create profile
        </Button>
      </form>
    </div>
  );
}
