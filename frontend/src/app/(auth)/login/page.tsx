import type { Metadata } from "next";
import { Suspense } from "react";

import { LoginForm } from "@/features/auth/login-form";

export const metadata: Metadata = {
  title: "Sign in",
  description: "Sign in securely to Monierave.",
};

export default function LoginPage() {
  return (
    <Suspense fallback={<p aria-live="polite">Preparing secure sign in…</p>}>
      <LoginForm />
    </Suspense>
  );
}
