import { CircleHelp, Mail, ShieldCheck } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { Button } from "@/components/ui/button";

export const metadata: Metadata = {
  title: "Help",
  description: "Find guidance for using Monierave safely.",
};

export default function HelpPage() {
  return (
    <main className="mx-auto max-w-5xl px-5 py-16 sm:px-8 lg:py-24">
      <p className="text-evergreen-700 text-sm font-bold tracking-wider uppercase">
        Help and support
      </p>
      <h1 className="mt-3 max-w-3xl font-serif text-5xl font-semibold tracking-[-0.04em]">
        Clear answers, without exposing your financial information.
      </h1>
      <p className="text-ink-600 mt-5 max-w-2xl text-lg leading-8">
        Monierave support will never ask for your password, verification token,
        refresh token, or complete access credentials.
      </p>
      <div className="mt-12 grid gap-5 md:grid-cols-2">
        <article className="border-line-200 rounded-md border bg-white p-6">
          <ShieldCheck
            aria-hidden="true"
            className="text-evergreen-700 size-7"
          />
          <h2 className="mt-4 text-xl font-semibold">Account and security</h2>
          <p className="text-ink-600 mt-2 leading-7">
            If you suspect unauthorized access, sign in and use “Sign out
            everywhere” from the Security page, then change your password.
          </p>
        </article>
        <article className="border-line-200 rounded-md border bg-white p-6">
          <CircleHelp
            aria-hidden="true"
            className="text-evergreen-700 size-7"
          />
          <h2 className="mt-4 text-xl font-semibold">Transfer questions</h2>
          <p className="text-ink-600 mt-2 leading-7">
            Keep the transaction reference available. Never resend a transfer
            while its result is uncertain; use the original retry action.
          </p>
        </article>
      </div>
      <section className="marketing-security-panel mt-12 rounded-lg p-7 text-white sm:p-10">
        <Mail aria-hidden="true" className="text-jade-100 size-7" />
        <h2 className="mt-4 font-serif text-3xl font-semibold">
          Development support channel
        </h2>
        <p className="mt-3 max-w-2xl text-white/75">
          A production support contact has not been configured yet. Do not send
          passwords, tokens, account numbers, or transaction details through
          informal channels.
        </p>
      </section>
      <Button asChild className="mt-8">
        <Link href="/login">Return to sign in</Link>
      </Button>
    </main>
  );
}
