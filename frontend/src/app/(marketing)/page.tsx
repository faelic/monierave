import type { Metadata } from "next";

import { IlluminatedClosingCta } from "@/components/marketing/illuminated-closing-cta";
import { OnboardingJourney } from "@/components/marketing/onboarding-journey";
import { MoneyMovementTheatre } from "@/components/marketing/money-movement-theatre";
import { ProductArchitecture } from "@/components/marketing/product-architecture";
import { SectionReveal } from "@/components/marketing/section-reveal";
import { RotatingGlobeHeroBody } from "@/features/rotating-globe-hero/rotating-globe-hero";

export const metadata: Metadata = {
  title: "Clear digital banking",
  description:
    "Understand your accounts and move money between Monierave users with clear review, status, and zero internal transfer fees.",
};

const trustMarks = [
  "Clear balances",
  "Verified recipients",
  "Secure transfers",
] as const;

export default function MarketingHomePage() {
  return (
    <>
      <section className="landing-hero">
        <div className="landing-hero-frame mx-auto max-w-[90rem]">
          <SectionReveal className="h-full min-h-0" immediate>
            <RotatingGlobeHeroBody />
          </SectionReveal>

          <div className="landing-trust-rail relative z-10 px-5 sm:px-8 lg:px-[4.625rem]">
            <p className="landing-trust-intro text-[0.6875rem]">
              Built for clear money movement
            </p>
            <ul className="landing-trust-list mt-4 grid grid-cols-2 items-center gap-x-7 gap-y-3 sm:grid-cols-3">
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

      <SectionReveal>
        <OnboardingJourney />
      </SectionReveal>

      <SectionReveal>
        <ProductArchitecture />
      </SectionReveal>

      <SectionReveal>
        <MoneyMovementTheatre />
      </SectionReveal>

      <SectionReveal>
        <IlluminatedClosingCta />
      </SectionReveal>
    </>
  );
}
