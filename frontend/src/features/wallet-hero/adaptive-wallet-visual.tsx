"use client";

import dynamic from "next/dynamic";
import { Component, useEffect, useRef, useState, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils/cn";

import { SceneFallback } from "./components/scene-fallback";
import {
  readWalletEnvironment,
  selectWalletCapability,
  walletFallbackMessages,
  type WalletCapability,
} from "./wallet-runtime-policy";

const WalletScene = dynamic(
  () => import("@/features/wallet-hero/monierave-wallet-scene"),
  {
    loading: () => <SceneFallback />,
    ssr: false,
  },
);

class SceneErrorBoundary extends Component<
  { children: ReactNode; onFailure: () => void },
  { failed: boolean }
> {
  state = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  componentDidCatch() {
    this.props.onFailure();
  }

  render() {
    return this.state.failed ? null : this.props.children;
  }
}

export function AdaptiveWalletVisual({
  className,
  fallbackClassName,
  showControl = true,
  showStatus = false,
  theme = "light",
  activationDelayMs = 0,
}: {
  className?: string;
  fallbackClassName?: string;
  showControl?: boolean;
  showStatus?: boolean;
  theme?: "dark" | "light";
  activationDelayMs?: number;
}) {
  const container = useRef<HTMLDivElement>(null);
  const [capability, setCapability] = useState<WalletCapability | null>(null);
  const [inViewport, setInViewport] = useState(false);
  const [preferStatic, setPreferStatic] = useState(false);
  const [sceneReady, setSceneReady] = useState(false);
  const [sceneFailed, setSceneFailed] = useState(false);
  const [activationReady, setActivationReady] = useState(
    activationDelayMs === 0,
  );

  useEffect(() => {
    if (activationDelayMs <= 0) {
      return;
    }

    let activationTimer = window.setTimeout(
      () => setActivationReady(true),
      activationDelayMs,
    );
    const deferForInteraction = () => {
      setActivationReady(false);
      window.clearTimeout(activationTimer);
      activationTimer = window.setTimeout(
        () => setActivationReady(true),
        activationDelayMs,
      );
    };

    window.addEventListener("input", deferForInteraction);
    window.addEventListener("keydown", deferForInteraction);
    window.addEventListener("pointerdown", deferForInteraction);

    return () => {
      window.clearTimeout(activationTimer);
      window.removeEventListener("input", deferForInteraction);
      window.removeEventListener("keydown", deferForInteraction);
      window.removeEventListener("pointerdown", deferForInteraction);
    };
  }, [activationDelayMs]);

  useEffect(() => {
    const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)");
    const wideViewport = window.matchMedia("(min-width: 64rem)");
    const evaluate = () => {
      setCapability(selectWalletCapability(readWalletEnvironment()));
    };
    const initialCheck = window.setTimeout(evaluate, 0);

    reducedMotion.addEventListener("change", evaluate);
    wideViewport.addEventListener("change", evaluate);

    return () => {
      window.clearTimeout(initialCheck);
      reducedMotion.removeEventListener("change", evaluate);
      wideViewport.removeEventListener("change", evaluate);
    };
  }, []);

  useEffect(() => {
    const element = container.current;
    if (!element) {
      return;
    }

    const Observer = (
      window as Window & {
        IntersectionObserver?: typeof IntersectionObserver;
      }
    ).IntersectionObserver;

    if (!Observer) {
      const visibilityCheck = window.setTimeout(() => setInViewport(true), 0);
      return () => window.clearTimeout(visibilityCheck);
    }

    const observer = new Observer(
      ([entry]) => setInViewport(Boolean(entry?.isIntersecting)),
      { rootMargin: "120px" },
    );
    observer.observe(element);

    return () => observer.disconnect();
  }, []);

  const interactive =
    capability?.mode === "interactive" &&
    activationReady &&
    inViewport &&
    !preferStatic &&
    !sceneFailed;
  const fallbackMessage = sceneFailed
    ? "The wallet animation stopped safely, so the still version is now active."
    : preferStatic
      ? "You selected the lightweight still version."
      : capability?.mode === "static"
        ? walletFallbackMessages[capability.reason]
        : "Checking this device before loading the optional wallet animation.";

  return (
    <div
      className={cn("relative", className)}
      data-wallet-mode={interactive ? "interactive" : "static"}
      ref={container}
    >
      <div className="size-full">
        {interactive ? (
          <SceneErrorBoundary onFailure={() => setSceneFailed(true)}>
            {!sceneReady ? (
              <div className="absolute inset-0">
                <SceneFallback />
              </div>
            ) : null}
            <div
              className={cn(
                "size-full transition-opacity duration-300 motion-reduce:transition-none",
                sceneReady ? "opacity-100" : "opacity-0",
              )}
            >
              <WalletScene
                onReady={() => setSceneReady(true)}
                onSceneFailure={() => setSceneFailed(true)}
              />
            </div>
          </SceneErrorBoundary>
        ) : (
          <div
            className="mx-auto grid size-full max-w-[44rem] place-items-center px-4"
            data-testid="wallet-static-fallback"
          >
            {fallbackClassName ? (
              <SceneFallback className={fallbackClassName} />
            ) : (
              <SceneFallback />
            )}
          </div>
        )}
      </div>

      {showControl &&
      capability?.mode === "interactive" &&
      inViewport &&
      !sceneFailed ? (
        <Button
          className={cn(
            "absolute right-3 bottom-3 shadow-lg",
            theme === "dark"
              ? "bg-evergreen-900/80 hover:bg-evergreen-800 border-white/25 text-white"
              : "bg-white/90",
          )}
          onClick={() => setPreferStatic((current) => !current)}
          size="compact"
          variant="secondary"
        >
          {preferStatic ? "Play wallet animation" : "Pause wallet animation"}
        </Button>
      ) : null}

      <p
        aria-live="polite"
        className={
          showStatus
            ? "bg-evergreen-900/80 absolute top-3 left-3 rounded-sm px-3 py-1.5 text-sm text-white/80 backdrop-blur-sm"
            : "sr-only"
        }
      >
        {interactive ? "Wallet animation active." : fallbackMessage}
      </p>
      <p className="sr-only">
        The visual represents a digital wallet opening to reveal three Monierave
        payment cards while two abstract transaction tokens move around it.
      </p>
    </div>
  );
}
