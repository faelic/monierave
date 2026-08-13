"use client";

import type { ReactNode } from "react";
import { useEffect, useState } from "react";

import { cn } from "@/lib/utils/cn";

export const heroCopyTiming = {
  headline: "Banking\nmade clear.",
  eyebrowDelay: 80,
  initialDelay: 300,
  characterDelay: 80,
  cursorSettleDelay: 1_000,
  supportingDelay: 480,
  actionsDelay: 620,
} as const;

export function AnimatedHeroCopy({
  actions,
  className,
  debugElapsedMs = null,
  supportingCopy = "Manage accounts, verify recipients, and move money with every important detail clearly reviewed.",
}: {
  actions?: ReactNode;
  className?: string;
  debugElapsedMs?: number | null;
  supportingCopy?: string;
}) {
  const [visibleCharacters, setVisibleCharacters] = useState(0);
  const [eyebrowVisible, setEyebrowVisible] = useState(false);
  const [supportingVisible, setSupportingVisible] = useState(false);
  const [actionsVisible, setActionsVisible] = useState(false);
  const [cursorVisible, setCursorVisible] = useState(true);
  const [reducedMotion, setReducedMotion] = useState(false);

  useEffect(() => {
    const query = window.matchMedia("(prefers-reduced-motion: reduce)");
    const evaluate = () => setReducedMotion(query.matches);
    const frame = requestAnimationFrame(evaluate);
    query.addEventListener("change", evaluate);
    return () => {
      cancelAnimationFrame(frame);
      query.removeEventListener("change", evaluate);
    };
  }, []);

  useEffect(() => {
    if (debugElapsedMs !== null) return;
    if (reducedMotion) {
      const frame = requestAnimationFrame(() => {
        setVisibleCharacters(heroCopyTiming.headline.length);
        setEyebrowVisible(true);
        setSupportingVisible(true);
        setActionsVisible(true);
      });
      return () => cancelAnimationFrame(frame);
    }

    let typingInterval = 0;
    const eyebrowStart = window.setTimeout(
      () => setEyebrowVisible(true),
      heroCopyTiming.eyebrowDelay,
    );
    const typingStart = window.setTimeout(() => {
      setVisibleCharacters(1);
      typingInterval = window.setInterval(() => {
        setVisibleCharacters((current) => {
          if (current >= heroCopyTiming.headline.length) {
            window.clearInterval(typingInterval);
            return current;
          }
          return current + 1;
        });
      }, heroCopyTiming.characterDelay);
    }, heroCopyTiming.initialDelay);
    const supportingStart = window.setTimeout(
      () => setSupportingVisible(true),
      heroCopyTiming.supportingDelay,
    );
    const actionsStart = window.setTimeout(
      () => setActionsVisible(true),
      heroCopyTiming.actionsDelay,
    );

    return () => {
      window.clearTimeout(eyebrowStart);
      window.clearTimeout(typingStart);
      window.clearTimeout(supportingStart);
      window.clearTimeout(actionsStart);
      window.clearInterval(typingInterval);
    };
  }, [debugElapsedMs, reducedMotion]);

  useEffect(() => {
    if (
      debugElapsedMs !== null ||
      reducedMotion ||
      visibleCharacters < heroCopyTiming.headline.length
    ) {
      return;
    }

    const cursorSettle = window.setTimeout(
      () => setCursorVisible(false),
      heroCopyTiming.cursorSettleDelay,
    );
    return () => window.clearTimeout(cursorSettle);
  }, [debugElapsedMs, reducedMotion, visibleCharacters]);

  const debugCharacters =
    debugElapsedMs === null || debugElapsedMs < heroCopyTiming.initialDelay
      ? 0
      : Math.min(
          heroCopyTiming.headline.length,
          1 +
            Math.floor(
              (debugElapsedMs - heroCopyTiming.initialDelay) /
                heroCopyTiming.characterDelay,
            ),
        );
  const effectiveCharacters = reducedMotion
    ? heroCopyTiming.headline.length
    : debugElapsedMs === null
      ? visibleCharacters
      : debugCharacters;
  const effectiveSupportingVisible = reducedMotion
    ? true
    : debugElapsedMs === null
      ? supportingVisible
      : debugElapsedMs >= heroCopyTiming.supportingDelay;
  const effectiveEyebrowVisible = reducedMotion
    ? true
    : debugElapsedMs === null
      ? eyebrowVisible
      : debugElapsedMs >= heroCopyTiming.eyebrowDelay;
  const effectiveActionsVisible = reducedMotion
    ? true
    : debugElapsedMs === null
      ? actionsVisible
      : debugElapsedMs >= heroCopyTiming.actionsDelay;
  const visibleHeadline = heroCopyTiming.headline.slice(0, effectiveCharacters);
  const visibleHeadlineParts = visibleHeadline.split("\n");
  const typingCompletionTime =
    heroCopyTiming.initialDelay +
    (heroCopyTiming.headline.length - 1) * heroCopyTiming.characterDelay;
  const effectiveCursorVisible =
    !reducedMotion &&
    (debugElapsedMs === null
      ? cursorVisible
      : debugElapsedMs <
        typingCompletionTime + heroCopyTiming.cursorSettleDelay);

  return (
    <div className={cn("w-full", className)}>
      <p
        className={cn(
          "mb-3 text-[clamp(0.62rem,0.8vw,0.78rem)] font-bold tracking-[0.18em] text-[var(--marketing-accent-highlight)] uppercase transition-[opacity,transform] duration-500 ease-[cubic-bezier(0.16,1,0.3,1)] motion-reduce:transition-none",
          effectiveEyebrowVisible
            ? "translate-y-0 opacity-100"
            : "translate-y-2 opacity-0",
        )}
        data-testid="hero-eyebrow"
      >
        Monierave banking
      </p>
      <h1 className="font-hero-mono relative max-w-[17ch] text-[clamp(2rem,8vw,2.5rem)] leading-[0.96] font-normal tracking-[-0.025em] text-white md:text-[clamp(3.25rem,4.2vw,4rem)]">
        <span className="sr-only">Banking made clear.</span>
        <span aria-hidden="true" className="relative block">
          <span className="invisible block whitespace-pre-line">
            {heroCopyTiming.headline}
          </span>
          <span
            className="absolute inset-0 block whitespace-pre-line"
            data-testid="hero-headline-visual"
          >
            {visibleHeadlineParts.map((line, index) => {
              const isLastVisibleLine =
                index === visibleHeadlineParts.length - 1;

              return (
                <span
                  className="relative block w-fit max-w-full max-md:mx-auto"
                  key={`${index}-${line}`}
                >
                  {index === 1 && line.startsWith("made clear") ? (
                    <>
                      made
                      <span className="inline-block w-[0.22em]"> </span>
                      {line.slice("made ".length)}
                    </>
                  ) : (
                    line
                  )}
                  {effectiveCursorVisible && isLastVisibleLine ? (
                    <span
                      className="organic-hero-cursor"
                      data-testid="hero-headline-cursor"
                    >
                      _
                    </span>
                  ) : null}
                  {!isLastVisibleLine ? "\n" : null}
                </span>
              );
            })}
          </span>
        </span>
      </h1>
      <div
        className={cn(
          "transition-[opacity,transform] duration-[520ms] ease-[cubic-bezier(0.16,1,0.3,1)] motion-reduce:transition-none",
          effectiveSupportingVisible
            ? "translate-y-0 opacity-100"
            : "translate-y-7 opacity-0 [will-change:transform,opacity]",
        )}
        data-testid="hero-supporting-copy"
      >
        <p className="mt-7 max-w-[31rem] text-[clamp(0.9rem,1.2vw,1.125rem)] leading-[1.55] text-white/68">
          {supportingCopy}
        </p>
      </div>
      <div
        className={cn(
          "mt-7 flex flex-wrap gap-2 sm:gap-3 [&>*]:transition-[opacity,transform] [&>*]:duration-[520ms] [&>*]:ease-[cubic-bezier(0.16,1,0.3,1)] [&>*]:motion-reduce:transition-none [&>*:nth-child(2)]:delay-[90ms]",
          effectiveActionsVisible
            ? "[&>*]:translate-y-0 [&>*]:opacity-100"
            : "[&>*]:translate-y-7 [&>*]:opacity-0 [&>*]:[will-change:transform,opacity]",
        )}
        data-testid="hero-actions"
      >
        {actions ?? (
          <>
            <span className="rounded-full bg-white px-5 py-3 text-sm font-bold text-black">
              Create account
            </span>
            <span className="rounded-full border border-white/20 px-5 py-3 text-sm font-bold text-white/75">
              See how it works
            </span>
          </>
        )}
      </div>
    </div>
  );
}
