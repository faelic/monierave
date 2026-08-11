"use client";

import type { CSSProperties, ReactNode } from "react";
import { useEffect, useRef, useState } from "react";

import styles from "./onboarding-journey.module.css";

const journeySteps = [
  {
    number: "01",
    title: "Create your profile",
    description: "Set up your secure Monierave profile and sign in.",
    motion: "account",
    visual: <AccountMotion />,
  },
  {
    number: "02",
    title: "Verify your email",
    description:
      "Confirm your email to unlock transfers and protected account features.",
    motion: "verification",
    visual: <VerificationMotion />,
  },
  {
    number: "03",
    title: "Make a transfer",
    description:
      "Choose an account, confirm the recipient and amount, then review before sending.",
    motion: "transfer",
    visual: <TransferMotion />,
  },
] as const;

function MotionFrame({ children }: { children: ReactNode }) {
  return (
    <div aria-hidden="true" className={styles.motionFrame}>
      {children}
    </div>
  );
}

function AccountMotion() {
  return (
    <MotionFrame>
      <svg className={styles.motionSvg} viewBox="0 0 120 120">
        <g className={styles.accountCorners}>
          <path d="M27 44V29h15" />
          <path d="M78 29h15v15" />
          <path d="M93 76v15H78" />
          <path d="M42 91H27V76" />
        </g>
        <rect
          className={styles.accountFrame}
          height="62"
          rx="9"
          width="66"
          x="27"
          y="29"
        />
        <g className={styles.accountIdentity}>
          <circle cx="60" cy="52" r="9" />
          <path d="M43 79c2.8-10.6 9.1-15.9 17-15.9S74.2 68.4 77 79" />
        </g>
        <line
          className={styles.accountScan}
          data-motion-part="profile-scan"
          x1="34"
          x2="86"
          y1="44"
          y2="44"
        />
        <circle
          className={styles.accountNode}
          cx="0"
          cy="0"
          data-motion-part="profile-node"
          r="2.5"
        />
      </svg>
    </MotionFrame>
  );
}

function VerificationMotion() {
  return (
    <MotionFrame>
      <svg className={styles.motionSvg} viewBox="0 0 120 120">
        <g
          className={styles.scannerRings}
          data-motion-part="verification-rings"
        >
          <circle cx="60" cy="60" r="38" />
          <circle cx="60" cy="60" r="28" />
          <circle cx="60" cy="60" r="18" />
          <path d="M60 18v8M60 94v8M18 60h8M94 60h8" />
        </g>
        <g
          className={styles.scannerSweep}
          data-motion-part="verification-sweep"
        >
          <path d="M60 60 88 39" />
          <circle cx="88" cy="39" r="2.6" />
        </g>
        <circle className={styles.scannerCore} cx="60" cy="60" r="12" />
        <path className={styles.verificationCheck} d="m53 60 5 5 10-11" />
      </svg>
    </MotionFrame>
  );
}

function TransferMotion() {
  return (
    <MotionFrame>
      <svg className={styles.motionSvg} viewBox="0 0 120 120">
        <g className={styles.transferAccounts}>
          <rect height="30" rx="6" width="28" x="14" y="48" />
          <path d="M22 58h12M22 66h8" />
          <rect height="30" rx="6" width="28" x="78" y="48" />
          <path d="M86 58h12M86 66h8" />
        </g>
        <path className={styles.transferRoute} d="M42 63C53 34 68 34 78 63" />
        <path className={styles.transferReturn} d="M78 69C67 91 53 91 42 69" />
        <circle className={styles.transferValue} cx="0" cy="0" r="4" />
        <path className={styles.transferCheck} d="m86 82 4 4 8-9" />
      </svg>
    </MotionFrame>
  );
}

export function OnboardingJourney() {
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
      { threshold: 0.28 },
    );

    observer.observe(section);
    return () => observer.disconnect();
  }, []);

  return (
    <section
      aria-labelledby="how-it-works-title"
      className={styles.section}
      data-journey-active={active ? "true" : "false"}
      id="how-it-works"
      ref={sectionRef}
    >
      <div className={styles.inner}>
        <div className={styles.headingBlock}>
          <p className={styles.eyebrow}>How Monierave works</p>
          <h2
            aria-label="From sign-up to your first transfer."
            className={styles.heading}
            id="how-it-works-title"
          >
            From sign-up to your
            <span> first transfer.</span>
          </h2>
          <p className={styles.introduction}>
            A clear path from creating your profile to moving money with every
            important detail reviewed.
          </p>
        </div>

        <div className={styles.stepsWrap}>
          <span aria-hidden="true" className={styles.progressTrack}>
            <span className={styles.progressSignal} />
          </span>
          <ol className={styles.steps}>
            {journeySteps.map((step, index) => (
              <li
                className={styles.step}
                data-step-motion={step.motion}
                key={step.number}
                style={{ "--step-index": index } as CSSProperties}
              >
                {step.visual}
                <div className={styles.stepCopy}>
                  <p className={styles.stepNumber}>{step.number}</p>
                  <h3 className={styles.stepTitle}>{step.title}</h3>
                  <p className={styles.stepDescription}>{step.description}</p>
                </div>
              </li>
            ))}
          </ol>
        </div>
      </div>
    </section>
  );
}
