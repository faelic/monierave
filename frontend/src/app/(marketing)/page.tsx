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

export default function MarketingHomePage() {
  return (
    <>
      <section className="landing-hero">
        <div aria-hidden="true" className="landing-hero-light-field" />
        <div aria-hidden="true" className="landing-hero-light-occlusion" />
        <div className="landing-hero-frame mx-auto max-w-[90rem]">
          <SectionReveal className="h-full min-h-0" immediate>
            <RotatingGlobeHeroBody />
          </SectionReveal>
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
