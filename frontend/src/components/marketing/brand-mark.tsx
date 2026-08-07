import Link from "next/link";

import { cn } from "@/lib/utils/cn";

export function BrandMark({
  className,
  inverse = false,
}: {
  className?: string;
  inverse?: boolean;
}) {
  return (
    <Link
      aria-label="Monierave home"
      className={cn(
        "group inline-flex min-h-9 items-center gap-2 no-underline",
        inverse ? "text-white" : "",
        className,
      )}
      href="/"
    >
      <svg
        aria-hidden="true"
        className={cn(
          "size-6",
          inverse ? "text-jade-100" : "text-evergreen-800",
        )}
        fill="none"
        viewBox="0 0 32 32"
      >
        <rect
          height="26"
          rx="8"
          stroke="currentColor"
          strokeWidth="2"
          width="26"
          x="3"
          y="3"
        />
        <path
          d="M9 21V11l7 6 7-6v10"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="2.25"
        />
        <path
          d="M11 25h10"
          stroke="currentColor"
          strokeLinecap="round"
          strokeWidth="2"
        />
      </svg>
      <span
        className={cn(
          "font-display text-[0.9375rem] font-bold tracking-[-0.045em]",
          inverse ? "text-white" : "text-evergreen-900",
        )}
      >
        Monierave
      </span>
    </Link>
  );
}
