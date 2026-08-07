import type { Metadata } from "next";

import { SignupForm } from "@/features/auth/signup-form";

export const metadata: Metadata = {
  title: "Create account",
  description: "Create your Monierave registration.",
};

export default function SignupPage() {
  return <SignupForm />;
}
