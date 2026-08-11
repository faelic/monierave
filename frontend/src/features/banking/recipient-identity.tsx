import { CheckCircle2 } from "lucide-react";

import type { Account } from "@/lib/api/contracts";
import { cn } from "@/lib/utils/cn";

export function RecipientIdentity({
  accountName,
  accountNumber,
  canReceive,
  className,
  currency,
}: {
  accountName: string;
  accountNumber: string;
  canReceive?: boolean;
  className?: string;
  currency: Account["currency"];
}) {
  return (
    <div
      className={cn(
        "border-line-200 flex items-start gap-3 rounded-md border bg-white p-4",
        className,
      )}
    >
      <span className="bg-jade-100 text-success-700 grid size-9 shrink-0 place-items-center rounded-full">
        <CheckCircle2 aria-hidden="true" className="size-5" />
      </span>
      <div className="min-w-0">
        <p className="font-semibold break-words">{accountName}</p>
        <p
          className="text-ink-600 mt-1 font-mono text-sm"
          data-financial-number
        >
          {accountNumber} · {currency}
        </p>
        {canReceive ? (
          <p className="text-success-700 mt-2 text-xs font-semibold">
            Can receive transfers
          </p>
        ) : null}
      </div>
    </div>
  );
}
