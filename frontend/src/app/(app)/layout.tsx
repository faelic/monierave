import { Suspense } from "react";

import { BankingShell } from "@/components/layout/banking-shell";
import { ApplicationBoundary } from "@/features/auth/application-boundary";
import { AuthProvider } from "@/features/auth/auth-provider";

export default function ApplicationLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AuthProvider>
      <Suspense fallback={null}>
        <ApplicationBoundary>
          <BankingShell>{children}</BankingShell>
        </ApplicationBoundary>
      </Suspense>
    </AuthProvider>
  );
}
