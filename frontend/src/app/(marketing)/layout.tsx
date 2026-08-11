import { MarketingShell } from "@/components/layout/marketing-shell";
import { AuthProvider } from "@/features/auth/auth-provider";

export default function MarketingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AuthProvider>
      <MarketingShell>{children}</MarketingShell>
    </AuthProvider>
  );
}
