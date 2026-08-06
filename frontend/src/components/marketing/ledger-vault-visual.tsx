import { ArrowRight, Check } from "lucide-react";

import { cn } from "@/lib/utils/cn";

export function LedgerVaultVisual({ className }: { className?: string }) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "vault-visual relative mx-auto aspect-square w-full max-w-[36rem]",
        className,
      )}
    >
      <div className="vault-orbit vault-orbit-outer" />
      <div className="vault-orbit vault-orbit-inner" />
      <span className="vault-node top-[47%] left-[8%]" />
      <span className="vault-node top-[18%] right-[12%]" />
      <span className="vault-node bottom-[8%] left-[52%]" />

      <div className="vault-card">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-xs font-semibold tracking-[0.16em] text-white/60 uppercase">
              Monierave
            </p>
            <p className="mt-1 text-lg font-semibold text-white">
              Internal transfer
            </p>
          </div>
          <div className="border-jade-500/50 bg-jade-500/15 flex size-10 items-center justify-center rounded-full border">
            <Check className="text-jade-500 size-5" />
          </div>
        </div>
        <div className="mt-14 grid grid-cols-[1fr_auto_1fr] items-center gap-3">
          <div>
            <p className="text-[0.65rem] tracking-[0.12em] text-white/45 uppercase">
              Review
            </p>
            <div className="mt-2 h-1.5 w-14 rounded-full bg-white/70" />
          </div>
          <ArrowRight className="text-jade-500 size-5" />
          <div>
            <p className="text-[0.65rem] tracking-[0.12em] text-white/45 uppercase">
              Posted
            </p>
            <div className="bg-jade-500 mt-2 h-1.5 w-14 rounded-full" />
          </div>
        </div>
        <div className="mt-8 flex items-end justify-between border-t border-white/15 pt-5">
          <div>
            <p className="text-xs text-white/50">Transfer fee</p>
            <p className="mt-1 text-2xl font-semibold text-white">0</p>
          </div>
          <p className="rounded-full border border-white/15 px-3 py-1.5 text-xs text-white/65">
            Same currency
          </p>
        </div>
      </div>
    </div>
  );
}
