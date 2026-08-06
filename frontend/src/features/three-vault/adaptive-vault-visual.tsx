"use client";

import dynamic from "next/dynamic";
import { Component, useEffect, useRef, useState, type ReactNode } from "react";

import { LedgerVaultVisual } from "@/components/marketing/ledger-vault-visual";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils/cn";

import {
  readVaultEnvironment,
  selectVaultCapability,
  vaultFallbackMessages,
  type VaultCapability,
} from "./vault-capabilities";

const VaultScene = dynamic(() => import("./vault-scene"), {
  loading: () => (
    <div className="grid size-full place-items-center" role="status">
      <span
        aria-label="Preparing optional 3D preview"
        className="border-jade-500 size-7 animate-spin rounded-full border-2 border-r-transparent motion-reduce:animate-none"
      />
    </div>
  ),
  ssr: false,
});

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

export function AdaptiveVaultVisual({
  className,
  fallbackClassName,
  showControl = true,
  showStatus = false,
  theme = "light",
}: {
  className?: string;
  fallbackClassName?: string;
  showControl?: boolean;
  showStatus?: boolean;
  theme?: "dark" | "light";
}) {
  const container = useRef<HTMLDivElement>(null);
  const [capability, setCapability] = useState<VaultCapability | null>(null);
  const [inViewport, setInViewport] = useState(false);
  const [preferStatic, setPreferStatic] = useState(false);
  const [sceneFailed, setSceneFailed] = useState(false);

  useEffect(() => {
    const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)");
    const wideViewport = window.matchMedia("(min-width: 64rem)");
    const evaluate = () => {
      setCapability(selectVaultCapability(readVaultEnvironment()));
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
    inViewport &&
    !preferStatic &&
    !sceneFailed;
  const fallbackMessage = sceneFailed
    ? "The 3D preview stopped safely, so the lightweight version is now active."
    : preferStatic
      ? "You selected the lightweight still version."
      : capability?.mode === "static"
        ? vaultFallbackMessages[capability.reason]
        : "Checking this device before loading the optional 3D preview.";

  return (
    <div
      className={cn("relative", className)}
      data-vault-mode={interactive ? "interactive" : "static"}
      ref={container}
    >
      <div className="size-full">
        {interactive ? (
          <SceneErrorBoundary onFailure={() => setSceneFailed(true)}>
            <VaultScene onSceneFailure={() => setSceneFailed(true)} />
          </SceneErrorBoundary>
        ) : (
          <div
            className="mx-auto grid size-full max-w-[44rem] place-items-center px-4"
            data-testid="vault-static-fallback"
          >
            {fallbackClassName ? (
              <LedgerVaultVisual className={fallbackClassName} />
            ) : (
              <LedgerVaultVisual />
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
          {preferStatic ? "Use 3D preview" : "Pause 3D preview"}
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
        {interactive ? "Interactive preview active." : fallbackMessage}
      </p>
    </div>
  );
}
