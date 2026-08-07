import { forwardRef } from "react";

export const FormErrorSummary = forwardRef<
  HTMLDivElement,
  { message?: string | undefined }
>(function FormErrorSummary({ message }, ref) {
  if (!message) {
    return null;
  }

  return (
    <div
      ref={ref}
      className="border-danger-700 text-danger-700 rounded-sm border-l-4 bg-[#fff5f3] px-4 py-3 text-sm font-medium"
      role="alert"
      tabIndex={-1}
    >
      {message}
    </div>
  );
});
