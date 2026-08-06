import type { ComponentProps, ReactNode } from "react";

import { cn } from "@/lib/utils/cn";

type FieldProps = ComponentProps<"div"> & {
  error?: string | undefined;
  hint?: string | undefined;
  label: string;
  name: string;
  optional?: boolean;
  children: ReactNode;
};

export function Field({
  children,
  className,
  error,
  hint,
  label,
  name,
  optional = false,
  ...props
}: FieldProps) {
  const hintID = hint ? `${name}-hint` : undefined;
  const errorID = error ? `${name}-error` : undefined;

  return (
    <div className={cn("grid gap-2", className)} {...props}>
      <div className="flex items-baseline justify-between gap-4">
        <label htmlFor={name} className="text-sm font-semibold">
          {label}
        </label>
        {optional ? (
          <span className="text-ink-600 text-xs">Optional</span>
        ) : null}
      </div>
      {hint ? (
        <p id={hintID} className="text-ink-600 text-sm">
          {hint}
        </p>
      ) : null}
      {children}
      {error ? (
        <p id={errorID} className="text-danger-700 text-sm font-medium">
          {error}
        </p>
      ) : null}
    </div>
  );
}

export function fieldDescriptionIDs({
  error,
  hint,
  name,
}: Pick<FieldProps, "error" | "hint" | "name">) {
  return [hint ? `${name}-hint` : null, error ? `${name}-error` : null]
    .filter(Boolean)
    .join(" ");
}
