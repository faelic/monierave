import { MailCheck } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { Button } from "@/components/ui/button";

export const metadata: Metadata = {
  title: "Check your email",
};

export default function CheckEmailPage() {
  return (
    <section className="auth-form-enter text-center">
      <div className="bg-jade-100 text-evergreen-800 mx-auto grid size-16 place-items-center rounded-full">
        <MailCheck aria-hidden="true" className="size-8" />
      </div>
      <p className="text-evergreen-700 mt-6 text-sm font-bold tracking-[0.16em] uppercase">
        Registration created
      </p>
      <h1 className="mt-3 font-serif text-4xl leading-tight font-semibold tracking-[-0.035em]">
        Check your email.
      </h1>
      <p className="text-ink-600 mx-auto mt-4 max-w-sm leading-7">
        Open the message from Monierave and use its secure confirmation button.
        The verification link is valid for 24 hours.
      </p>
      <div className="border-line-200 bg-paper-50 mt-7 rounded-md border p-4 text-left text-sm leading-6">
        Verification confirms your address, but it does not sign you in. After
        confirming, return here and sign in with your username and password.
      </div>
      <Button asChild className="mt-7 w-full">
        <Link href="/login">Continue to sign in</Link>
      </Button>
      <Link
        className="text-evergreen-800 mt-4 inline-flex min-h-11 items-center font-semibold"
        href="/"
      >
        Return home
      </Link>
    </section>
  );
}
