import type { Metadata } from "next";

import { VerificationNeeded } from "@/features/auth/verification-needed";

export const metadata: Metadata = {
  title: "Verify your email",
  description: "Manage your pending Monierave registration.",
};

export default function VerificationNeededPage() {
  return <VerificationNeeded />;
}
