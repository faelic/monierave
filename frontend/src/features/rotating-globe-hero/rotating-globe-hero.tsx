"use client";

import * as Dialog from "@radix-ui/react-dialog";
import { ArrowRight, Menu, X } from "lucide-react";
import Link from "next/link";

import { AnimatedHeroCopy } from "./animated-hero-copy";
import { CelestialMoneyGlobe } from "./celestial-money-globe";
import styles from "./rotating-globe-hero.module.css";
import { useReducedMotionPreference } from "./use-reduced-motion-preference";

const navigation = [
  { href: "/#how-it-works", label: "How it works" },
  { href: "/#product-architecture", label: "Workspace" },
  { href: "/#money-movement", label: "Money movement" },
] as const;

export function RotatingGlobeHero({
  paused = false,
  reviewTime = null,
}: {
  paused?: boolean;
  reviewTime?: number | null;
}) {
  const reducedMotion = useReducedMotionPreference();

  return (
    <section
      className={styles.candidate}
      data-rotating-globe-motion={reducedMotion ? "static" : "active"}
      data-rotating-globe-stage
    >
      <header className={styles.navigation}>
        <Link className={styles.brand} href="/">
          <span aria-hidden="true" className={styles.brandMark}>
            M
          </span>
          <span>Monierave</span>
        </Link>

        <nav
          aria-label="Candidate primary navigation"
          className={styles.desktopNavigation}
        >
          {navigation.map(({ href, label }) => (
            <Link className={styles.navigationLink} href={href} key={href}>
              {label}
            </Link>
          ))}
        </nav>

        <div className={styles.navigationActions}>
          <Link className={styles.signInLink} href="/login">
            Sign in
          </Link>
          <Link className={styles.createAccountLink} href="/signup">
            Get started
          </Link>
        </div>

        <Dialog.Root>
          <Dialog.Trigger asChild>
            <button
              aria-label="Open candidate navigation"
              className={styles.mobileMenuTrigger}
              type="button"
            >
              <Menu aria-hidden="true" size={20} />
            </button>
          </Dialog.Trigger>
          <Dialog.Portal>
            <Dialog.Overlay className={styles.mobileDialogOverlay} />
            <Dialog.Content className={styles.mobileDialog}>
              <div className={styles.mobileDialogHeader}>
                <Dialog.Title className={styles.mobileDialogTitle}>
                  Navigation
                </Dialog.Title>
                <Dialog.Close asChild>
                  <button
                    aria-label="Close candidate navigation"
                    className={styles.mobileClose}
                    type="button"
                  >
                    <X aria-hidden="true" size={20} />
                  </button>
                </Dialog.Close>
              </div>
              <Dialog.Description className="sr-only">
                Explore Monierave or continue to your account.
              </Dialog.Description>
              <nav
                aria-label="Candidate mobile navigation"
                className={styles.mobileNavigation}
              >
                {navigation.map(({ href, label }) => (
                  <Dialog.Close asChild key={href}>
                    <Link className={styles.mobileNavigationLink} href={href}>
                      {label}
                    </Link>
                  </Dialog.Close>
                ))}
              </nav>
              <div className={styles.mobileActions}>
                <Dialog.Close asChild>
                  <Link className={styles.signInLink} href="/login">
                    Sign in
                  </Link>
                </Dialog.Close>
                <Dialog.Close asChild>
                  <Link className={styles.createAccountLink} href="/signup">
                    Get started
                  </Link>
                </Dialog.Close>
              </div>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
      </header>

      <Link className={styles.announcement} href="/signup">
        Move and manage money with clarity · Get started →
      </Link>

      <RotatingGlobeHeroBody
        paused={paused}
        reducedMotion={reducedMotion}
        reviewTime={reviewTime}
      />
    </section>
  );
}

export function RotatingGlobeHeroBody({
  paused = false,
  reducedMotion,
  reviewTime = null,
}: {
  paused?: boolean;
  reducedMotion?: boolean;
  reviewTime?: number | null;
}) {
  const prefersReducedMotion = useReducedMotionPreference();
  const motionReduced = reducedMotion ?? prefersReducedMotion;

  return (
    <div
      className={styles.productionBody}
      data-rotating-globe-motion={motionReduced ? "static" : "active"}
      data-rotating-globe-stage
    >
      <div className={styles.heroBody}>
        <div className={styles.heroInner}>
          <div className={styles.heroCopy}>
            <AnimatedHeroCopy
              actions={
                <>
                  <Link className={styles.primaryAction} href="/signup">
                    Get started
                    <ArrowRight aria-hidden="true" size={16} />
                  </Link>
                  <Link className={styles.secondaryAction} href="#how-it-works">
                    See how it works
                  </Link>
                </>
              }
            />
          </div>

          <div className={styles.globeFrame}>
            <CelestialMoneyGlobe
              className={styles.globe}
              paused={paused}
              reducedMotion={motionReduced}
              reviewTime={reviewTime}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
