import { forwardRef } from "react";

import { cn } from "@/lib/utils/cn";

export const FormErrorSummary = forwardRef<
  HTMLDivElement,
  {
    className?: string | undefined;
    message?: string | undefined;
    tone?: "error" | "success" | undefined;
  }
>(function FormErrorSummary({ className, message, tone = "error" }, ref) {
  if (!message) {
    return null;
  }

  return (
    <div
      ref={ref}
      className={cn(
        "rounded-sm border-l-4 px-4 py-3 text-sm font-medium",
        tone === "success"
          ? "border-evergreen-700 text-evergreen-800 bg-[#eef8f1]"
          : "border-danger-700 text-danger-700 bg-[#fff5f3]",
        className,
      )}
      data-form-summary
      data-tone={tone}
      role={tone === "error" ? "alert" : "status"}
      tabIndex={-1}
    >
      {message}
    </div>
  );
});
