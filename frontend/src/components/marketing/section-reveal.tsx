"use client";

import type { CSSProperties, ReactNode } from "react";
import { useEffect, useRef, useState } from "react";

import { cn } from "@/lib/utils/cn";

import styles from "./section-reveal.module.css";

export function SectionReveal({
  children,
  className,
  delayMs = 0,
  immediate = false,
}: {
  children: ReactNode;
  className?: string;
  delayMs?: number;
  immediate?: boolean;
}) {
  const elementRef = useRef<HTMLDivElement>(null);
  const [ready, setReady] = useState(false);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const element = elementRef.current;
    if (!element) return;

    const reducedMotion =
      window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
    let readyFrame = 0;
    let revealFrame = 0;

    if (reducedMotion) {
      readyFrame = requestAnimationFrame(() => {
        setReady(true);
        setVisible(true);
      });
      return () => cancelAnimationFrame(readyFrame);
    }

    if (immediate) {
      readyFrame = requestAnimationFrame(() => {
        setReady(true);
        revealFrame = requestAnimationFrame(() => setVisible(true));
      });
      return () => {
        cancelAnimationFrame(readyFrame);
        if (revealFrame) cancelAnimationFrame(revealFrame);
      };
    }

    readyFrame = requestAnimationFrame(() => setReady(true));

    if (!("IntersectionObserver" in window)) {
      revealFrame = requestAnimationFrame(() => setVisible(true));
      return () => {
        cancelAnimationFrame(readyFrame);
        cancelAnimationFrame(revealFrame);
      };
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry?.isIntersecting) return;
        setVisible(true);
        observer.disconnect();
      },
      { rootMargin: "0px 0px 14%", threshold: 0.04 },
    );
    observer.observe(element);
    return () => {
      if (revealFrame) cancelAnimationFrame(revealFrame);
      if (readyFrame) cancelAnimationFrame(readyFrame);
      observer.disconnect();
    };
  }, [immediate]);

  return (
    <div
      className={cn(styles.shell, className)}
      data-section-reveal-shell
      ref={elementRef}
    >
      <div
        className={styles.reveal}
        data-reveal-ready={ready ? "true" : "false"}
        data-reveal-visible={visible ? "true" : "false"}
        style={{ "--section-reveal-delay": `${delayMs}ms` } as CSSProperties}
      >
        {children}
      </div>
    </div>
  );
}
