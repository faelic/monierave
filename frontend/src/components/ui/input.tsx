import type { ComponentProps } from "react";

import { cn } from "@/lib/utils/cn";

export function Input({ className, ...props }: ComponentProps<"input">) {
  return (
    <input
      className={cn(
        [
          "border-line-300 text-ink-950 placeholder:text-ink-600 min-h-11 w-full",
          "rounded-sm border bg-white px-3 py-2 text-base shadow-none",
          "hover:border-ink-600 disabled:bg-paper-100 disabled:cursor-not-allowed disabled:opacity-70",
          "aria-invalid:border-danger-700 aria-invalid:border-2",
        ],
        className,
      )}
      {...props}
    />
  );
}
