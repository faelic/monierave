import {
  ArrowRight,
  BookOpen,
  Landmark,
  ListChecks,
  Send,
  ShieldCheck,
} from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import { StaticReferenceArtwork } from "@/features/pipeline-hero/static-reference-artwork";

export const metadata: Metadata = {
  title: "Clear digital banking",
  description:
    "Understand your accounts and move money between Monierave users with clear review, status, and zero internal transfer fees.",
};

const capabilities = [
  {
    icon: Landmark,
    title: "Accounts with an identity",
    description:
      "Each account has a customer-facing 10-digit number, a clear currency, and a lifecycle status you can understand.",
  },
  {
    icon: Send,
    title: "Transfers you review",
    description:
      "See the sender, masked recipient, amount, currency, fee, and total before you confirm an internal transfer.",
  },
  {
    icon: ListChecks,
    title: "Activity with a status",
    description:
      "Posted transaction history keeps references, narration, direction, and outcome connected to the account involved.",
  },
] as const;

const steps = [
  {
    number: "01",
    title: "Choose the account",
    description:
      "Start with the currency, account status, and current posted balance in view.",
  },
  {
    number: "02",
    title: "Confirm the recipient",
    description:
      "Resolve a 10-digit account number to a privacy-safe, masked identity.",
  },
  {
    number: "03",
    title: "Review, then send",
    description:
      "Check the full transfer intent and submit it once with a unique request key.",
  },
] as const;

const trustMarks = [
  "Account identity",
  "Posted ledger",
  "Verified users",
  "Zero-fee internal",
  "Secure sessions",
] as const;

export default function MarketingHomePage() {
  return (
    <>
      <section className="landing-hero">
        <div className="landing-hero-frame mx-auto max-w-[90rem]">
          <div className="relative grid min-h-[34rem] px-5 pt-14 pb-8 sm:px-8 md:grid-cols-[42%_58%] md:items-center md:pt-10 lg:px-[4.625rem] lg:pt-0">
            <div className="relative z-10 max-w-[35.75rem] md:-translate-y-8">
              <h1 className="font-display text-[clamp(3.25rem,8.4vw,4.6rem)] leading-[0.97] font-extrabold tracking-[-0.055em] text-[#f3f3f1]">
                <span className="block">Your money,</span>
                <span className="block">without</span>
                <span className="block">mystery.</span>
              </h1>
              <p className="mt-5 max-w-[28rem] text-[0.9375rem] leading-[1.48] text-white/52 sm:text-base">
                Monierave brings accounts, recipients, and same-currency
                transfers into one calm place, with every important detail
                reviewed before money moves.
              </p>
              <Button
                asChild
                className="mt-6 min-h-10 rounded-full border-0 bg-white px-5 text-xs text-black hover:bg-white/88"
                size="compact"
                variant="secondary"
              >
                <a href="/signup">
                  Get started
                  <ArrowRight aria-hidden="true" className="size-3.5" />
                </a>
              </Button>
            </div>

            <StaticReferenceArtwork className="mt-8 aspect-[735/290] min-h-[20rem] md:absolute md:inset-0 md:mt-0 md:size-full" />
          </div>

          <div className="relative z-10 px-5 pb-5 sm:px-8 lg:px-[4.625rem] lg:pb-7">
            <p className="text-[0.6875rem] text-white/30">
              The controls a modern money platform should make visible:
            </p>
            <ul className="mt-4 grid grid-cols-2 items-center gap-x-7 gap-y-3 sm:grid-cols-3 lg:grid-cols-5">
              {trustMarks.map((mark, index) => (
                <li
                  className="landing-trust-mark flex items-center gap-2 text-[0.67rem] font-bold"
                  key={mark}
                >
                  <span
                    aria-hidden="true"
                    className="grid size-4 place-items-center rounded-[0.2rem] border border-current text-[0.5rem]"
                  >
                    {String(index + 1).padStart(2, "0")}
                  </span>
                  {mark}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      <section
        aria-label="Monierave product facts"
        className="border-line-200 border-y bg-white"
      >
        <div className="divide-line-200 mx-auto grid max-w-[90rem] divide-y px-5 sm:px-8 md:grid-cols-3 md:divide-x md:divide-y-0 xl:px-12">
          <div className="py-7 md:pr-8">
            <p className="font-serif text-3xl font-medium">10 digits</p>
            <p className="text-ink-600 mt-1 text-sm">
              Customer-facing account numbers
            </p>
          </div>
          <div className="py-7 md:px-8">
            <p className="font-serif text-3xl font-medium">Zero fee</p>
            <p className="text-ink-600 mt-1 text-sm">
              Current internal transfer model
            </p>
          </div>
          <div className="py-7 md:pl-8">
            <p className="font-serif text-3xl font-medium">Posted</p>
            <p className="text-ink-600 mt-1 text-sm">
              Truthful balance and activity language
            </p>
          </div>
        </div>
      </section>

      <section
        aria-labelledby="capabilities-title"
        className="mx-auto max-w-[90rem] px-5 py-20 sm:px-8 md:py-28 xl:px-12"
        id="product"
      >
        <div className="grid gap-10 lg:grid-cols-[0.72fr_1.28fr] lg:gap-20">
          <div>
            <p className="text-evergreen-700 text-sm font-bold tracking-[0.16em] uppercase">
              Built for comprehension
            </p>
            <h2
              className="mt-4 max-w-md font-serif text-4xl leading-tight font-medium tracking-[-0.035em] sm:text-5xl"
              id="capabilities-title"
            >
              Less decoration. More certainty.
            </h2>
            <p className="text-ink-700 mt-5 max-w-md text-lg leading-7">
              The interface gives financial facts room to breathe without hiding
              them inside dashboards full of noise.
            </p>
          </div>
          <div className="border-line-300 border-t">
            {capabilities.map(({ description, icon: Icon, title }) => (
              <article
                className="border-line-200 grid gap-4 border-b py-8 sm:grid-cols-[3.5rem_1fr] sm:gap-6"
                key={title}
              >
                <span className="bg-jade-100 text-evergreen-800 flex size-12 items-center justify-center rounded-md">
                  <Icon aria-hidden="true" className="size-5" />
                </span>
                <div>
                  <h3 className="text-xl font-bold">{title}</h3>
                  <p className="text-ink-700 mt-2 max-w-xl leading-7">
                    {description}
                  </p>
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section
        aria-labelledby="how-it-works-title"
        className="bg-evergreen-900 text-white"
        id="how-it-works"
      >
        <div className="mx-auto max-w-[90rem] px-5 py-20 sm:px-8 md:py-28 xl:px-12">
          <div className="max-w-2xl">
            <p className="text-jade-500 text-sm font-bold tracking-[0.16em] uppercase">
              A deliberate transfer path
            </p>
            <h2
              className="mt-4 font-serif text-4xl leading-tight font-medium tracking-[-0.035em] sm:text-5xl"
              id="how-it-works-title"
            >
              Three steps. No ambiguous moment.
            </h2>
          </div>
          <ol className="mt-14 grid border-t border-white/20 md:grid-cols-3">
            {steps.map((step) => (
              <li
                className="border-b border-white/20 py-8 md:border-r md:border-b-0 md:px-8 md:first:pl-0 md:last:border-r-0 md:last:pr-0"
                key={step.number}
              >
                <p className="text-jade-500 font-mono text-sm">{step.number}</p>
                <h3 className="mt-7 text-xl font-bold">{step.title}</h3>
                <p className="mt-3 max-w-sm leading-7 text-white/70">
                  {step.description}
                </p>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <section
        aria-labelledby="language-title"
        className="mx-auto grid max-w-[90rem] gap-12 px-5 py-20 sm:px-8 md:py-28 lg:grid-cols-2 lg:gap-20 xl:px-12"
      >
        <div className="lg:sticky lg:top-28 lg:self-start">
          <BookOpen aria-hidden="true" className="text-evergreen-700 size-8" />
          <h2
            className="mt-6 max-w-lg font-serif text-4xl leading-tight font-medium tracking-[-0.035em] sm:text-5xl"
            id="language-title"
          >
            Financial clarity has a vocabulary.
          </h2>
          <p className="text-ink-700 mt-5 max-w-lg text-lg leading-7">
            Monierave says exactly what the platform knows today, without
            implying financial capabilities that are not yet modeled.
          </p>
        </div>
        <dl className="border-line-300 border-t">
          <div className="border-line-200 border-b py-8">
            <dt className="text-lg font-bold">Current posted balance</dt>
            <dd className="text-ink-700 mt-2 leading-7">
              The balance after completed ledger postings. It is not called
              available balance because holds and card authorizations are not
              currently modeled.
            </dd>
          </div>
          <div className="border-line-200 border-b py-8">
            <dt className="text-lg font-bold">Internal transfer</dt>
            <dd className="text-ink-700 mt-2 leading-7">
              Money moving between Monierave accounts in the same currency. The
              current fee is zero.
            </dd>
          </div>
          <div className="border-line-200 border-b py-8">
            <dt className="text-lg font-bold">Posted status</dt>
            <dd className="text-ink-700 mt-2 leading-7">
              The transfer is recorded. Processing, failed, reversed, and
              uncertain outcomes are shown as distinct states.
            </dd>
          </div>
        </dl>
      </section>

      <section className="px-5 pb-20 sm:px-8 md:pb-28 xl:px-12">
        <div className="marketing-security-panel mx-auto grid max-w-[90rem] gap-10 overflow-hidden rounded-lg px-6 py-10 sm:px-10 md:grid-cols-[1fr_auto] md:items-end md:px-14 md:py-14">
          <div>
            <ShieldCheck aria-hidden="true" className="text-jade-500 size-8" />
            <h2 className="mt-6 max-w-2xl font-serif text-4xl leading-tight font-medium tracking-[-0.035em] text-white sm:text-5xl">
              Security should be visible in the workflow, not hidden in a
              slogan.
            </h2>
            <p className="mt-5 max-w-2xl text-lg leading-7 text-white/70">
              Learn how verification, session controls, request limits, and
              duplicate-submit protection shape the current product.
            </p>
          </div>
          <Button
            asChild
            className="text-evergreen-900 hover:bg-paper-100 bg-white"
          >
            <Link href="/security">
              Our security approach
              <ArrowRight aria-hidden="true" className="size-4" />
            </Link>
          </Button>
        </div>
      </section>
    </>
  );
}
