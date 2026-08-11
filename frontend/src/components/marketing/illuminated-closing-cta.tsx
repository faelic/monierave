"use client";

import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

import { marketingAccountActions } from "@/components/marketing/marketing-account-actions";
import { useAuth } from "@/features/auth/auth-provider";

import styles from "./illuminated-closing-cta.module.css";

export function IlluminatedClosingCta() {
  const { status, user } = useAuth();
  const accountActions = marketingAccountActions(status, user);
  const sectionRef = useRef<HTMLElement>(null);
  const [active, setActive] = useState(false);

  useEffect(() => {
    const section = sectionRef.current;
    if (!section) return;

    const Observer = (
      window as Window & {
        IntersectionObserver?: typeof IntersectionObserver;
      }
    ).IntersectionObserver;

    if (!Observer) {
      const frame = requestAnimationFrame(() => setActive(true));
      return () => cancelAnimationFrame(frame);
    }

    const observer = new Observer(
      ([entry]) => {
        if (!entry?.isIntersecting) return;
        setActive(true);
        observer.disconnect();
      },
      { threshold: 0.24 },
    );

    observer.observe(section);
    return () => observer.disconnect();
  }, []);

  return (
    <section
      aria-labelledby="closing-cta-title"
      className={styles.section}
      data-closing-cta-active={active ? "true" : "false"}
      ref={sectionRef}
    >
      <div className={styles.atmosphere} />
      <div aria-hidden="true" className={styles.brandMark}>
        M
      </div>

      <div className={styles.copy}>
        <h2 className={styles.heading} id="closing-cta-title">
          Ready to get started?
        </h2>
        <p className={styles.description}>
          Create your profile, verify your email, and move money with clarity.
        </p>
        <div className={styles.actions}>
          {accountActions.loading ? (
            <span
              aria-label="Restoring account session"
              className={`${styles.primaryAction} animate-pulse opacity-50 motion-reduce:animate-none`}
              role="status"
            />
          ) : (
            <>
              {accountActions.primary ? (
                <Link
                  className={styles.primaryAction}
                  href={accountActions.primary.href}
                >
                  {accountActions.primary.label}
                  <ArrowRight aria-hidden="true" />
                </Link>
              ) : null}
              {accountActions.secondary ? (
                <Link
                  className={styles.secondaryAction}
                  href={accountActions.secondary.href}
                >
                  {accountActions.secondary.label}
                </Link>
              ) : null}
            </>
          )}
        </div>
      </div>
    </section>
  );
}
