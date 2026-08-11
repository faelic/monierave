import type { Metadata } from "next";

import { SignupForm } from "@/features/auth/signup-form";

export const metadata: Metadata = {
  title: "Create profile",
  description: "Create your Monierave profile and verify your email.",
};

export default function SignupPage() {
  return <SignupForm />;
}
