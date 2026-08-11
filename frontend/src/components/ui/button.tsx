import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils/cn";

const buttonVariants = cva(
  [
    "inline-flex min-h-11 items-center justify-center gap-2 rounded-sm px-4 py-2.5",
    "text-sm font-semibold transition-colors duration-150",
    "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-55",
    "motion-reduce:transition-none",
  ],
  {
    variants: {
      variant: {
        primary:
          "bg-evergreen-800 text-white shadow-[inset_0_0_0_1px_rgb(255_255_255_/_5%)] hover:bg-evergreen-700 active:bg-evergreen-900",
        secondary:
          "border-line-300 bg-white text-ink-950 hover:bg-paper-100 border",
        quiet: "text-evergreen-800 hover:bg-jade-100",
        danger:
          "bg-danger-700 text-white hover:bg-[#8e211b] active:bg-[#761b17]",
      },
      size: {
        default: "min-h-11",
        compact: "min-h-9 px-3 py-1.5",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "default",
    },
  },
);

type ButtonProps = ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
    loading?: boolean;
  };

export function Button({
  asChild = false,
  children,
  className,
  disabled,
  loading = false,
  size,
  type = "button",
  variant,
  ...props
}: ButtonProps) {
  const classes = cn(buttonVariants({ size, variant }), className);

  if (asChild) {
    return (
      <Slot className={classes} {...props}>
        {children}
      </Slot>
    );
  }

  return (
    <button
      className={classes}
      disabled={disabled || loading}
      type={type}
      {...props}
    >
      {loading ? (
        <span
          aria-hidden="true"
          className="size-4 animate-spin rounded-full border-2 border-current border-r-transparent motion-reduce:animate-none"
        />
      ) : null}
      <span>{children}</span>
      {loading ? <span className="sr-only">Loading</span> : null}
    </button>
  );
}

export { buttonVariants };
