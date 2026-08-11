import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AnimatedHeroCopy, heroCopyTiming } from "./animated-hero-copy";

function mockReducedMotion(matches: boolean) {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn().mockReturnValue({
      addEventListener: vi.fn(),
      matches,
      media: "(prefers-reduced-motion: reduce)",
      removeEventListener: vi.fn(),
    }),
    writable: true,
  });
}

describe("AnimatedHeroCopy", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockReducedMotion(false);
  });

  afterEach(() => vi.useRealTimers());

  it("types once after the configured delay and keeps the final headline", () => {
    render(<AnimatedHeroCopy />);
    const visual = screen.getByTestId("hero-headline-visual");

    expect(visual).toHaveTextContent("_");
    act(() => vi.advanceTimersByTime(heroCopyTiming.initialDelay - 1));
    expect(visual).toHaveTextContent("_");
    act(() => vi.advanceTimersByTime(1));
    expect(visual).toHaveTextContent("Y_");

    act(() => vi.advanceTimersByTime(8_000));
    expect(visual).toHaveTextContent("Your money, without mystery.");
    expect(screen.getByTestId("hero-headline-cursor")).toBeInTheDocument();
    act(() => vi.advanceTimersByTime(heroCopyTiming.cursorSettleDelay));
    expect(
      screen.queryByTestId("hero-headline-cursor"),
    ).not.toBeInTheDocument();
    act(() => vi.advanceTimersByTime(8_000));
    expect(visual).toHaveTextContent("Your money, without mystery.");
  });

  it("reveals supporting content on its own staged delay", () => {
    render(<AnimatedHeroCopy />);
    const supporting = screen.getByTestId("hero-supporting-copy");
    const actions = screen.getByTestId("hero-actions");

    expect(supporting).toHaveClass("opacity-0");
    expect(actions).toHaveClass("[&>*]:opacity-0");
    act(() => vi.advanceTimersByTime(heroCopyTiming.supportingDelay));
    expect(supporting).toHaveClass("opacity-100");
    expect(actions).toHaveClass("[&>*]:opacity-0");
    act(() =>
      vi.advanceTimersByTime(
        heroCopyTiming.actionsDelay - heroCopyTiming.supportingDelay,
      ),
    );
    expect(actions).toHaveClass("[&>*]:opacity-100");
  });

  it("supports deterministic lab-only review states", () => {
    const { rerender } = render(<AnimatedHeroCopy debugElapsedMs={0} />);
    expect(screen.getByTestId("hero-headline-visual")).toHaveTextContent("_");

    rerender(<AnimatedHeroCopy debugElapsedMs={600} />);
    expect(screen.getByTestId("hero-headline-visual").textContent).toContain(
      "Your",
    );
    expect(screen.getByTestId("hero-supporting-copy")).toHaveClass(
      "opacity-100",
    );
    expect(screen.getByTestId("hero-actions")).toHaveClass("[&>*]:opacity-0");

    rerender(<AnimatedHeroCopy debugElapsedMs={700} />);
    expect(screen.getByTestId("hero-actions")).toHaveClass("[&>*]:opacity-100");
  });

  it("renders the complete static state for reduced motion", () => {
    mockReducedMotion(true);
    render(<AnimatedHeroCopy />);
    act(() => vi.advanceTimersByTime(20));
    act(() => vi.advanceTimersByTime(20));

    expect(screen.getByTestId("hero-headline-visual")).toHaveTextContent(
      "Your money, without mystery.",
    );
    expect(
      screen.queryByTestId("hero-headline-cursor"),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("hero-supporting-copy")).toHaveClass(
      "opacity-100",
    );
    expect(screen.getByTestId("hero-actions")).toHaveClass("[&>*]:opacity-100");
  });

  it("cleans up pending typing work when unmounted", () => {
    const clearInterval = vi.spyOn(window, "clearInterval");
    const clearTimeout = vi.spyOn(window, "clearTimeout");
    const { unmount } = render(<AnimatedHeroCopy />);

    unmount();

    expect(clearInterval).toHaveBeenCalled();
    expect(clearTimeout).toHaveBeenCalledTimes(4);
  });
});
