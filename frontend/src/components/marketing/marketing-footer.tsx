"use client";

import Link from "next/link";

import { BrandMark } from "@/components/marketing/brand-mark";
import { marketingAccountActions } from "@/components/marketing/marketing-account-actions";
import { useAuth } from "@/features/auth/auth-provider";

export function MarketingFooter() {
  const { status, user } = useAuth();
  const accountActions = marketingAccountActions(status, user);
  return (
    <footer className="bg-[var(--marketing-canvas)] text-white">
      <div className="mx-auto grid max-w-[90rem] gap-10 px-5 py-12 sm:px-8 md:grid-cols-[1.4fr_1fr] md:py-16 xl:px-12">
        <div>
          <BrandMark className="[&_svg]:text-jade-500 [&_span]:text-white" />
          <p className="mt-4 max-w-md text-sm leading-6 text-white/70">
            A calm, precise way to understand accounts and move money between
            Monierave users.
          </p>
        </div>
        <div className="grid grid-cols-2 gap-8 text-sm">
          <nav aria-label="Footer navigation">
            <p className="font-semibold text-white">Explore</p>
            <ul className="mt-4 grid gap-3 text-white/70">
              <li>
                <Link className="hover:text-white" href="/">
                  Home
                </Link>
              </li>
              <li>
                <Link className="hover:text-white" href="/#how-it-works">
                  How it works
                </Link>
              </li>
            </ul>
          </nav>
          <div>
            <p className="font-semibold text-white">Account</p>
            <ul className="mt-4 grid gap-3 text-white/70">
              {accountActions.loading ? (
                <li className="h-5 w-20 animate-pulse rounded bg-white/8 motion-reduce:animate-none">
                  <span className="sr-only">Restoring account session</span>
                </li>
              ) : (
                <>
                  {accountActions.secondary ? (
                    <li>
                      <Link
                        className="hover:text-white"
                        href={accountActions.secondary.href}
                      >
                        {accountActions.secondary.label}
                      </Link>
                    </li>
                  ) : null}
                  {accountActions.primary ? (
                    <li>
                      <Link
                        className="hover:text-white"
                        href={accountActions.primary.href}
                      >
                        {accountActions.primary.label}
                      </Link>
                    </li>
                  ) : null}
                </>
              )}
            </ul>
          </div>
        </div>
      </div>
      <div className="border-t border-white/15">
        <div className="mx-auto flex max-w-[90rem] flex-col gap-2 px-5 py-5 text-xs text-white/60 sm:px-8 md:flex-row md:items-center md:justify-between xl:px-12">
          <p>Monierave is a product in active development.</p>
          <p>&copy; {new Date().getFullYear()} Monierave.</p>
        </div>
      </div>
    </footer>
  );
}
