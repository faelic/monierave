import { ShieldCheck } from "lucide-react";

import { BrandMark } from "@/components/marketing/brand-mark";
import { SkipLink } from "@/components/ui/skip-link";
import { AdaptiveVaultVisual } from "@/features/three-vault/adaptive-vault-visual";

export function AuthenticationShell({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[minmax(0,1fr)_minmax(28rem,0.78fr)]">
      <SkipLink />
      <main
        id="main-content"
        className="relative flex min-h-screen items-center justify-center bg-white px-5 py-24 sm:px-8 lg:py-16"
      >
        <BrandMark className="absolute top-5 left-5 sm:top-7 sm:left-8 lg:top-8 lg:left-10" />
        <div className="w-full max-w-md">{children}</div>
      </main>
      <aside
        aria-label="Monierave introduction"
        className="auth-trust-panel bg-evergreen-900 text-paper-50 relative hidden overflow-hidden border-l border-white/10 p-12 lg:flex lg:flex-col lg:justify-between"
      >
        <div className="relative z-10 flex items-center gap-2 text-sm font-semibold text-white/75">
          <ShieldCheck aria-hidden="true" className="size-5" />
          Security is part of the design
        </div>
        <AdaptiveVaultVisual
          className="auth-vault relative z-10 mx-auto h-72 w-full max-w-md rounded-full"
          fallbackClassName="h-full w-auto max-w-full"
          theme="dark"
        />
        <div className="relative z-10">
          <p className="max-w-md font-serif text-4xl leading-tight font-medium">
            Your credentials belong to you, not your browser storage.
          </p>
          <p className="mt-5 max-w-md text-base leading-7 text-white/70">
            Monierave keeps access short-lived and uses protected cookies to
            restore your session safely.
          </p>
        </div>
      </aside>
    </div>
  );
}
