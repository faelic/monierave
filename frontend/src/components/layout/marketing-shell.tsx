import { MarketingFooter } from "@/components/marketing/marketing-footer";
import { MarketingHeader } from "@/components/marketing/marketing-header";
import { SkipLink } from "@/components/ui/skip-link";

export function MarketingShell({ children }: { children: React.ReactNode }) {
  return (
    <>
      <SkipLink />
      <MarketingHeader />
      <main className="bg-[var(--marketing-canvas)]" id="main-content">
        {children}
      </main>
      <MarketingFooter />
    </>
  );
}
