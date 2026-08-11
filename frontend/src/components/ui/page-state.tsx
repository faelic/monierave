import { AlertCircle, Inbox, LoaderCircle } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils/cn";

export function LoadingState({ label }: { label: string }) {
  return (
    <div
      aria-live="polite"
      className="border-line-200 text-ink-600 mt-6 flex min-h-32 items-center justify-center gap-3 rounded-md border bg-white"
      role="status"
    >
      <LoaderCircle
        aria-hidden="true"
        className="size-5 animate-spin motion-reduce:animate-none"
      />
      {label}
    </div>
  );
}

export function EmptyState({
  action,
  description,
  title,
}: {
  action?: ReactNode;
  description: string;
  title: string;
}) {
  return (
    <div className="border-line-200 mt-6 rounded-md border bg-white px-5 py-10 text-center">
      <span className="text-evergreen-800 mx-auto grid size-12 place-items-center rounded-full bg-[var(--product-accent-soft)]">
        <Inbox aria-hidden="true" className="size-5" />
      </span>
      <h2 className="mt-4 text-lg font-semibold">{title}</h2>
      <p className="text-ink-600 mx-auto mt-2 max-w-md text-sm leading-6">
        {description}
      </p>
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}

export function ErrorState({
  className,
  description,
  onRetry,
  title,
}: {
  className?: string;
  description?: string;
  onRetry?: () => void;
  title: string;
}) {
  return (
    <div
      className={cn(
        "border-danger-700 mt-6 rounded-md border-l-4 bg-[#fff5f3] px-5 py-5",
        className,
      )}
      role="alert"
    >
      <div className="flex gap-3">
        <AlertCircle
          aria-hidden="true"
          className="text-danger-700 mt-0.5 size-5 shrink-0"
        />
        <div>
          <h2 className="font-semibold">{title}</h2>
          {description ? (
            <p className="text-ink-600 mt-1 text-sm leading-6">{description}</p>
          ) : null}
          {onRetry ? (
            <Button
              className="mt-4"
              onClick={onRetry}
              size="compact"
              variant="secondary"
            >
              Try again
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  );
}
