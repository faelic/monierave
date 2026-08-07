import {
  ArrowRight,
  BadgeCheck,
  CircleAlert,
  Fingerprint,
  KeyRound,
  LockKeyhole,
  RefreshCw,
  ShieldCheck,
  TimerReset,
} from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { Button } from "@/components/ui/button";

export const metadata: Metadata = {
  title: "Security",
  description:
    "See how Monierave currently protects registration, sessions, and transfer requests, and understand the platform's security boundaries.",
};

const safeguards = [
  {
    icon: BadgeCheck,
    title: "Verified email before banking",
    description:
      "New registrations must confirm the email address before financial routes become available. Verification links expire after 24 hours.",
  },
  {
    icon: KeyRound,
    title: "Compromised-password screening",
    description:
      "New and changed passwords are checked through a privacy-preserving breach lookup. The complete password or hash is never sent to the provider.",
  },
  {
    icon: Fingerprint,
    title: "One active device session",
    description:
      "A successful login creates a new session and ends previous sessions for that user, limiting the account to one authenticated device at a time.",
  },
  {
    icon: RefreshCw,
    title: "Rotating refresh credentials",
    description:
      "Refresh credentials are stored as hashes, replaced after use, and bound to the device session. Reuse revokes the affected session.",
  },
  {
    icon: TimerReset,
    title: "Request limits",
    description:
      "Signup, login, and recipient-resolution requests have targeted rate limits to reduce automated abuse and account enumeration.",
  },
  {
    icon: LockKeyhole,
    title: "Duplicate-transfer protection",
    description:
      "Transfer requests require an idempotency key. Reusing a key with different instructions is rejected instead of creating an uncertain duplicate.",
  },
] as const;

const boundaries = [
  "Monierave does not currently provide multi-factor authentication or step-up transaction approval.",
  "Identity-document verification, external-bank transfers, virtual cards, and card authorizations are not currently available.",
  "The displayed balance is the current posted balance. Holds and pending card authorizations are not modeled.",
  "No security control replaces careful review of the sender, masked recipient, amount, currency, and fee before confirmation.",
] as const;

export default function SecurityPage() {
  return (
    <>
      <section className="marketing-hero border-line-200 border-b">
        <div className="mx-auto grid max-w-[90rem] gap-12 px-5 py-16 sm:px-8 md:py-24 lg:grid-cols-[1.1fr_0.9fr] lg:items-end xl:px-12">
          <div className="marketing-reveal">
            <p className="text-evergreen-700 text-sm font-bold tracking-[0.18em] uppercase">
              Security at Monierave
            </p>
            <h1 className="mt-5 max-w-3xl font-serif text-[clamp(3.25rem,7vw,6.5rem)] leading-[0.95] font-medium tracking-[-0.055em]">
              Protection,
              <span className="text-evergreen-700 block">
                explained plainly.
              </span>
            </h1>
          </div>
          <div className="marketing-reveal marketing-reveal-delay border-evergreen-700 border-l-2 pl-6">
            <p className="text-ink-700 max-w-xl text-lg leading-8">
              Security is a chain of deliberate controls, not a badge. Here is
              what the current platform enforces and where its boundaries
              remain.
            </p>
            <Button asChild className="mt-7" variant="secondary">
              <Link href="#safeguards">
                Review the safeguards
                <ArrowRight aria-hidden="true" className="size-4" />
              </Link>
            </Button>
          </div>
        </div>
      </section>

      <section
        aria-labelledby="safeguards-title"
        className="mx-auto max-w-[90rem] px-5 py-20 sm:px-8 md:py-28 xl:px-12"
        id="safeguards"
      >
        <div className="max-w-2xl">
          <ShieldCheck
            aria-hidden="true"
            className="text-evergreen-700 size-8"
          />
          <h2
            className="mt-6 font-serif text-4xl leading-tight font-medium tracking-[-0.035em] sm:text-5xl"
            id="safeguards-title"
          >
            Safeguards in the current product.
          </h2>
          <p className="text-ink-700 mt-5 text-lg leading-7">
            These controls are implemented across the API, database
            transactions, and authentication boundary.
          </p>
        </div>

        <div className="border-line-300 mt-14 grid border-t md:grid-cols-2 xl:grid-cols-3">
          {safeguards.map(({ description, icon: Icon, title }, index) => (
            <article
              className={[
                "border-line-200 border-b py-8 md:p-8",
                index % 2 === 0 ? "md:border-r" : "",
                index % 3 !== 2 ? "xl:border-r" : "xl:border-r-0",
              ].join(" ")}
              key={title}
            >
              <span className="bg-jade-100 text-evergreen-800 flex size-11 items-center justify-center rounded-md">
                <Icon aria-hidden="true" className="size-5" />
              </span>
              <h3 className="mt-6 text-xl font-bold">{title}</h3>
              <p className="text-ink-700 mt-3 leading-7">{description}</p>
            </article>
          ))}
        </div>
      </section>

      <section className="bg-paper-100 border-line-200 border-y">
        <div className="mx-auto grid max-w-[90rem] gap-12 px-5 py-20 sm:px-8 md:py-24 lg:grid-cols-[0.8fr_1.2fr] lg:gap-20 xl:px-12">
          <div>
            <CircleAlert
              aria-hidden="true"
              className="text-warning-700 size-8"
            />
            <h2 className="mt-6 max-w-md font-serif text-4xl leading-tight font-medium tracking-[-0.035em] sm:text-5xl">
              Clear boundaries are part of security.
            </h2>
            <p className="text-ink-700 mt-5 max-w-md text-lg leading-7">
              We do not describe planned capabilities as active protection.
            </p>
          </div>
          <ul className="border-line-300 border-t">
            {boundaries.map((boundary) => (
              <li
                className="border-line-200 flex gap-4 border-b py-6 leading-7"
                key={boundary}
              >
                <span
                  aria-hidden="true"
                  className="bg-warning-700 mt-2 size-2 shrink-0 rounded-full"
                />
                <span>{boundary}</span>
              </li>
            ))}
          </ul>
        </div>
      </section>

      <section
        aria-labelledby="your-role-title"
        className="mx-auto max-w-[90rem] px-5 py-20 sm:px-8 md:py-28 xl:px-12"
      >
        <div className="marketing-security-panel grid gap-10 rounded-lg px-6 py-10 sm:px-10 md:grid-cols-[1fr_auto] md:items-end md:px-14 md:py-14">
          <div>
            <h2
              className="max-w-2xl font-serif text-4xl leading-tight font-medium tracking-[-0.035em] text-white sm:text-5xl"
              id="your-role-title"
            >
              Your review is a security control too.
            </h2>
            <p className="mt-5 max-w-2xl text-lg leading-7 text-white/70">
              Confirm the account, masked recipient, amount, currency, fee, and
              total before sending. Monierave will never use a resolved
              recipient as proof that a later transfer is guaranteed.
            </p>
          </div>
          <Button
            asChild
            className="text-evergreen-900 hover:bg-paper-100 bg-white"
          >
            <a href="/signup">
              Create account
              <ArrowRight aria-hidden="true" className="size-4" />
            </a>
          </Button>
        </div>
      </section>
    </>
  );
}
