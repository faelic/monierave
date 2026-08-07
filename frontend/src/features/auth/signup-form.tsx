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
    <div className="auth-form-enter">
      <p className="text-evergreen-700 text-sm font-bold tracking-[0.16em] uppercase">
        Start with clarity
      </p>
      <h1 className="mt-3 font-serif text-4xl leading-tight font-semibold tracking-[-0.035em] sm:text-5xl">
        Create your Monierave profile.
      </h1>
      <p className="text-ink-600 mt-4 text-base leading-7">
        Registration takes a minute. You’ll verify your email before banking
        features become available.
      </p>

      <form
        className="mt-8 grid gap-5"
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
            {...register("full_name")}
          />
        </Field>
        <Field
          error={errors.username?.message}
          hint="3–32 letters and numbers."
          label="Username"
          name="username"
        >
          <Input
            autoCapitalize="none"
            autoComplete="username"
            aria-describedby={fieldDescriptionIDs({
              error: errors.username?.message,
              hint: "3–32 letters and numbers.",
              name: "username",
            })}
            aria-invalid={Boolean(errors.username)}
            id="username"
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
            type="email"
            {...register("email")}
          />
        </Field>
        <Field
          error={errors.password?.message}
          hint="Use 8–72 bytes. Known breached passwords are rejected."
          label="Password"
          name="password"
        >
          <PasswordInput
            autoComplete="new-password"
            aria-describedby={fieldDescriptionIDs({
              error: errors.password?.message,
              hint: "Use 8–72 bytes. Known breached passwords are rejected.",
              name: "password",
            })}
            aria-invalid={Boolean(errors.password)}
            id="password"
            {...register("password")}
          />
        </Field>
        <Button className="mt-1 w-full" loading={isSubmitting} type="submit">
          Create registration
        </Button>
      </form>

      <p className="text-ink-600 mt-6 text-center text-sm">
        Already registered?{" "}
        <Link className="text-evergreen-800 font-semibold" href="/login">
          Sign in
        </Link>
      </p>
    </div>
  );
}
