import { AuthenticationShell } from "@/components/layout/authentication-shell";
import { AuthProvider } from "@/features/auth/auth-provider";

export default function AuthenticationLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AuthProvider>
      <AuthenticationShell>{children}</AuthenticationShell>
    </AuthProvider>
  );
}
