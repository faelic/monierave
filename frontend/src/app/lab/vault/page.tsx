import { FlaskConical, Gauge, PauseCircle, ShieldCheck } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { BrandMark } from "@/components/marketing/brand-mark";
import { SkipLink } from "@/components/ui/skip-link";
import { VaultLaboratory } from "@/features/three-vault/vault-laboratory";

export const metadata: Metadata = {
  title: "3D vault laboratory",
  description:
    "An isolated Monierave experiment for an optional, accessible security-vault visual.",
  robots: {
    follow: false,
    index: false,
  },
};

const safeguards = [
  {
    icon: Gauge,
    title: "Bounded rendering",
    detail:
      "Pixel density is capped at 1.5 and the scene uses procedural geometry without model or texture downloads.",
  },
  {
    icon: PauseCircle,
    title: "Respectful motion",
    detail:
      "Reduced motion, data saving, constrained hardware, hidden tabs, and WebGL loss all receive safe behavior.",
  },
  {
    icon: ShieldCheck,
    title: "No financial truth",
    detail:
      "The canvas is decorative. It never renders balances, account numbers, status, or security decisions.",
  },
] as const;

export default function VaultLaboratoryPage() {
  return (
    <div className="vault-lab-page min-h-screen text-white">
      <SkipLink />
      <header className="border-b border-white/10">
        <div className="mx-auto flex min-h-20 max-w-[90rem] items-center justify-between px-5 sm:px-8 xl:px-12">
          <BrandMark inverse />
          <Link
            className="text-sm font-semibold text-white/70 hover:text-white"
            href="/"
          >
            Exit laboratory
          </Link>
        </div>
      </header>

      <main
        className="mx-auto max-w-[90rem] px-5 py-12 sm:px-8 md:py-16 xl:px-12"
        id="main-content"
      >
        <div className="grid gap-8 border-b border-white/14 pb-12 lg:grid-cols-[1fr_0.6fr] lg:items-end">
          <div>
            <p className="text-jade-500 flex items-center gap-2 text-sm font-bold tracking-[0.16em] uppercase">
              <FlaskConical aria-hidden="true" className="size-4" />
              Isolated visual laboratory
            </p>
            <h1 className="mt-5 max-w-4xl font-serif text-[clamp(3rem,7vw,6.5rem)] leading-[0.95] font-medium tracking-[-0.055em]">
              Security, expressed without spectacle.
            </h1>
          </div>
          <p className="max-w-xl text-lg leading-8 text-white/68 lg:pb-2">
            This route evaluates one optional visual idea before it reaches the
            product. Forms, navigation, and banking tasks remain independent of
            this scene.
          </p>
        </div>

        <div className="py-12 md:py-16">
          <VaultLaboratory />
        </div>

        <section
          aria-labelledby="safeguards-title"
          className="border-t border-white/14 pt-12"
        >
          <h2 className="font-serif text-3xl font-medium" id="safeguards-title">
            Promotion safeguards
          </h2>
          <div className="mt-8 grid border-t border-white/14 md:grid-cols-3">
            {safeguards.map(({ detail, icon: Icon, title }) => (
              <article
                className="border-b border-white/14 py-7 md:border-r md:px-7 md:first:pl-0 md:last:border-r-0 md:last:pr-0"
                key={title}
              >
                <Icon aria-hidden="true" className="text-jade-500 size-6" />
                <h3 className="mt-5 text-lg font-bold">{title}</h3>
                <p className="mt-2 leading-7 text-white/65">{detail}</p>
              </article>
            ))}
          </div>
        </section>
      </main>
    </div>
  );
}
