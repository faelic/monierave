import { ShieldCheck } from "lucide-react";

import { BrandMark } from "@/components/marketing/brand-mark";
import { SkipLink } from "@/components/ui/skip-link";
import { AdaptiveWalletVisual } from "@/features/wallet-hero/adaptive-wallet-visual";

export function AuthenticationShell({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[minmax(28rem,0.88fr)_minmax(0,1fr)]">
      <SkipLink />
      <main
        id="main-content"
        className="relative flex min-h-screen items-center justify-center bg-white px-5 py-24 sm:px-8 lg:order-2 lg:py-16"
      >
        <BrandMark className="absolute top-5 left-5 sm:top-7 sm:left-8 lg:top-8 lg:left-10" />
        <div className="w-full max-w-md">
          <AdaptiveWalletVisual
            className="mx-auto mb-8 aspect-[4/3] w-full lg:hidden"
            showControl={false}
          />
          {children}
        </div>
      </main>
      <aside
        aria-label="Monierave introduction"
        className="wallet-auth-panel text-evergreen-900 border-evergreen-900/10 relative hidden overflow-hidden border-r p-10 lg:order-1 lg:flex lg:flex-col lg:justify-between xl:p-12"
      >
        <div className="text-evergreen-800/75 relative z-10 flex items-center gap-2 text-sm font-semibold">
          <ShieldCheck aria-hidden="true" className="size-5" />
          Security is part of the design
        </div>
        <AdaptiveWalletVisual
          activationDelayMs={2500}
          className="wallet-auth-visual relative z-10 mx-auto aspect-[4/3] w-full max-w-2xl"
          fallbackClassName="size-full"
        />
        <div className="relative z-10">
          <p className="max-w-md font-serif text-4xl leading-tight font-medium">
            Your credentials belong to you, not your browser storage.
          </p>
          <p className="text-ink-700 mt-5 max-w-md text-base leading-7">
            Monierave keeps access short-lived and uses protected cookies to
            restore your session safely.
          </p>
        </div>
      </aside>
    </div>
  );
}
