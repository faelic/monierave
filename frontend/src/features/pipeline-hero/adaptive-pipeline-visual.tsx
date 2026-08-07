"use client";

import dynamic from "next/dynamic";
import { Component, useEffect, useRef, useState, type ReactNode } from "react";

import { cn } from "@/lib/utils/cn";

import { PipelineFallback } from "./pipeline-fallback";

const PipelineScene = dynamic(() => import("./pipeline-scene"), {
  ssr: false,
});

class PipelineErrorBoundary extends Component<
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

type NavigatorWithHints = Navigator & {
  connection?: { saveData?: boolean };
};

function supportsWebGL() {
  try {
    const canvas = document.createElement("canvas");
    return Boolean(canvas.getContext("webgl2") || canvas.getContext("webgl"));
  } catch {
    return false;
  }
}

export function AdaptivePipelineVisual({ className }: { className?: string }) {
  const root = useRef<HTMLDivElement>(null);
  const [eligible, setEligible] = useState(false);
  const [inViewport, setInViewport] = useState(false);
  const [ready, setReady] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)");
    const wideViewport = window.matchMedia("(min-width: 48rem)");
    const evaluate = () => {
      const connection = (navigator as NavigatorWithHints).connection;
      setEligible(
        wideViewport.matches &&
          !reducedMotion.matches &&
          !connection?.saveData &&
          supportsWebGL(),
      );
    };

    evaluate();
    reducedMotion.addEventListener("change", evaluate);
    wideViewport.addEventListener("change", evaluate);
    return () => {
      reducedMotion.removeEventListener("change", evaluate);
      wideViewport.removeEventListener("change", evaluate);
    };
  }, []);

  useEffect(() => {
    const element = root.current;
    if (!element) {
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => setInViewport(Boolean(entry?.isIntersecting)),
      { rootMargin: "160px" },
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const renderScene = eligible && inViewport && !failed;

  return (
    <div
      aria-label="A metallic network of Monierave payment rails"
      className={cn("relative", className)}
      data-pipeline-mode={renderScene ? "webgl" : "fallback"}
      ref={root}
      role="img"
    >
      <div className="absolute inset-0">
        <PipelineFallback
          className={cn(
            "transition-opacity duration-300 motion-reduce:transition-none",
            renderScene && ready ? "opacity-0" : "opacity-100",
          )}
        />
      </div>
      {renderScene ? (
        <PipelineErrorBoundary onFailure={() => setFailed(true)}>
          <div
            aria-hidden="true"
            className={cn(
              "absolute inset-0 transition-opacity duration-300 motion-reduce:transition-none",
              ready ? "opacity-100" : "opacity-0",
            )}
          >
            <PipelineScene onReady={() => setReady(true)} />
          </div>
        </PipelineErrorBoundary>
      ) : null}
    </div>
  );
}
