"use client";

import type { CSSProperties, ReactNode } from "react";
import { useEffect, useRef, useState } from "react";

import styles from "./product-architecture.module.css";

const finalStage = 6;

type ArchitectureNodeProps = {
  children: ReactNode;
  className?: string | undefined;
  currentStage: number;
  icon: ReactNode;
  stage: number;
  title: string;
};

function ArchitectureNode({
  children,
  className = "",
  currentStage,
  icon,
  stage,
  title,
}: ArchitectureNodeProps) {
  const active = currentStage >= stage;
  const current = currentStage === stage;

  return (
    <article
      className={`${styles.node} ${className}`}
      data-node-active={active ? "true" : "false"}
      data-node-current={current ? "true" : "false"}
      data-node-stage={stage}
    >
      <div aria-hidden="true" className={styles.nodeIcon}>
        {icon}
      </div>
      <div className={styles.nodeCopy}>
        <h3>{title}</h3>
        {children}
      </div>
      <span aria-hidden="true" className={styles.statusLight} />
    </article>
  );
}

function Connector({ active }: { active: boolean }) {
  return (
    <span
      aria-hidden="true"
      className={styles.connector}
      data-connector-active={active ? "true" : "false"}
    >
      <span />
    </span>
  );
}

function AccessIcon() {
  return (
    <svg viewBox="0 0 40 40">
      <rect height="17" rx="4" width="22" x="9" y="17" />
      <path d="M14 17v-4a6 6 0 0 1 12 0v4" />
      <path className={styles.accessGate} d="M20 23v5" />
    </svg>
  );
}

function DashboardIcon() {
  return (
    <svg viewBox="0 0 40 40">
      <rect height="26" rx="4" width="28" x="6" y="7" />
      <path d="M12 13h6v5h-6zM22 13h6v5h-6zM12 22h6v5h-6zM22 22h6v5h-6z" />
    </svg>
  );
}

function AccountsIcon() {
  return (
    <svg viewBox="0 0 40 40">
      <rect height="19" rx="4" width="27" x="8" y="14" />
      <path d="M5 27V11a4 4 0 0 1 4-4h22" />
      <path d="M14 21h8M14 26h14" />
    </svg>
  );
}

function TransferIcon() {
  return (
    <svg viewBox="0 0 40 40">
      <circle cx="8" cy="20" r="4" />
      <circle cx="32" cy="20" r="4" />
      <path d="M12 20c5-10 11-10 16 0" />
      <circle className={styles.transferDot} cx="0" cy="0" r="2.2" />
    </svg>
  );
}

function BeneficiaryIcon() {
  return (
    <svg viewBox="0 0 40 40">
      <circle cx="17" cy="15" r="6" />
      <path d="M7 31c1.7-7.2 5-10.8 10-10.8S25.3 23.8 27 31" />
      <circle className={styles.trustRing} cx="29" cy="24" r="7" />
      <path d="m26 24 2 2 4-5" />
    </svg>
  );
}

function ActivityIcon() {
  return (
    <svg viewBox="0 0 40 40">
      <path d="M11 10h21M11 20h21M11 30h21" />
      <circle cx="6" cy="10" r="2" />
      <circle cx="6" cy="20" r="2" />
      <circle cx="6" cy="30" r="2" />
    </svg>
  );
}

function SecurityIcon() {
  return (
    <svg viewBox="0 0 40 40">
      <path d="M20 5 31 9v9c0 8-4.8 13.7-11 17-6.2-3.3-11-9-11-17V9l11-4Z" />
      <path d="m15 20 3 3 7-8" />
    </svg>
  );
}

export function ProductArchitecture() {
  const sectionRef = useRef<HTMLElement>(null);
  const [stage, setStage] = useState(0);

  useEffect(() => {
    const section = sectionRef.current;
    if (!section) return;

    const motionQuery = window.matchMedia("(prefers-reduced-motion: reduce)");
    let frame = 0;

    const update = () => {
      frame = 0;
      if (motionQuery.matches || window.innerWidth < 768) {
        setStage(finalStage);
        return;
      }

      const rect = section.getBoundingClientRect();
      const stickyOffset = 80;
      const travel = Math.max(1, rect.height - window.innerHeight);
      const progress = Math.min(
        1,
        Math.max(0, (stickyOffset - rect.top) / travel),
      );
      const nextStage = Math.min(
        finalStage,
        Math.floor(progress * (finalStage + 1)),
      );
      setStage((current) => (current === nextStage ? current : nextStage));
    };

    const scheduleUpdate = () => {
      if (frame) return;
      frame = requestAnimationFrame(update);
    };

    frame = requestAnimationFrame(update);
    window.addEventListener("scroll", scheduleUpdate, { passive: true });
    window.addEventListener("resize", scheduleUpdate);
    motionQuery.addEventListener("change", scheduleUpdate);

    return () => {
      if (frame) cancelAnimationFrame(frame);
      window.removeEventListener("scroll", scheduleUpdate);
      window.removeEventListener("resize", scheduleUpdate);
      motionQuery.removeEventListener("change", scheduleUpdate);
    };
  }, []);

  return (
    <section
      aria-labelledby="product-architecture-title"
      className={styles.section}
      data-architecture-stage={stage}
      id="product-architecture"
      ref={sectionRef}
    >
      <div className={styles.stickyFrame}>
        <div className={styles.inner}>
          <div className={styles.copyColumn}>
            <p className={styles.eyebrow}>Your Monierave workspace</p>
            <h2 className={styles.heading} id="product-architecture-title">
              Everything important, connected.
            </h2>
            <p className={styles.introduction}>
              Your dashboard connects accounts, transfers, beneficiaries,
              activity, profile details, and security controls in one place.
            </p>

            <div className={styles.architectureSummary}>
              <div data-summary-active={stage >= 1 ? "true" : "false"}>
                <span>01</span>
                <p>See the whole picture from your dashboard.</p>
              </div>
              <div data-summary-active={stage >= 2 ? "true" : "false"}>
                <span>02</span>
                <p>Manage money through connected workflows.</p>
              </div>
              <div data-summary-active={stage >= 5 ? "true" : "false"}>
                <span>03</span>
                <p>Keep activity traceable and access controlled.</p>
              </div>
            </div>

            <div
              aria-hidden="true"
              className={styles.stageProgress}
              style={
                {
                  "--architecture-progress": stage / finalStage,
                } as CSSProperties
              }
            >
              <span />
            </div>
          </div>

          <div
            aria-label="Monierave product capability map"
            className={styles.diagram}
          >
            <ArchitectureNode
              className={styles.compactNode}
              currentStage={stage}
              icon={<AccessIcon />}
              stage={0}
              title="Verified access"
            >
              <p>Secure sign-in and confirmed email</p>
            </ArchitectureNode>

            <Connector active={stage >= 1} />

            <ArchitectureNode
              className={styles.dashboardNode}
              currentStage={stage}
              icon={<DashboardIcon />}
              stage={1}
              title="Dashboard overview"
            >
              <p>Accounts, posted balances, and recent activity</p>
            </ArchitectureNode>

            <Connector active={stage >= 2} />

            <div className={styles.workspacePool}>
              <p className={styles.poolLabel}>Core workspace</p>
              <div className={styles.capabilityGrid}>
                <ArchitectureNode
                  currentStage={stage}
                  icon={<AccountsIcon />}
                  stage={2}
                  title="Accounts"
                >
                  <p>View balances, status, and account details</p>
                </ArchitectureNode>
                <ArchitectureNode
                  currentStage={stage}
                  icon={<TransferIcon />}
                  stage={3}
                  title="Transfers"
                >
                  <p>Resolve, review, and send with clarity</p>
                </ArchitectureNode>
                <ArchitectureNode
                  className={styles.beneficiaryNode}
                  currentStage={stage}
                  icon={<BeneficiaryIcon />}
                  stage={4}
                  title="Beneficiaries"
                >
                  <p>Save and manage trusted recipients</p>
                </ArchitectureNode>
              </div>
            </div>

            <Connector active={stage >= 5} />

            <ArchitectureNode
              className={styles.activityNode}
              currentStage={stage}
              icon={<ActivityIcon />}
              stage={5}
              title="Posted activity"
            >
              <p>References, statuses, and transaction details</p>
            </ArchitectureNode>

            <Connector active={stage >= 6} />

            <ArchitectureNode
              className={styles.securityNode}
              currentStage={stage}
              icon={<SecurityIcon />}
              stage={6}
              title="Profile & security"
            >
              <p>Personal details, sessions, and safe logout</p>
            </ArchitectureNode>
          </div>
        </div>
      </div>
    </section>
  );
}
