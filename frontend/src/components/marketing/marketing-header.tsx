"use client";

import * as Dialog from "@radix-ui/react-dialog";
import { Menu, X } from "lucide-react";
import type { Route } from "next";
import Link from "next/link";
import { useEffect, useState, type MouseEvent } from "react";

import { BrandMark } from "@/components/marketing/brand-mark";
import { marketingAccountActions } from "@/components/marketing/marketing-account-actions";
import { useAuth } from "@/features/auth/auth-provider";
import { cn } from "@/lib/utils/cn";

const navigation: ReadonlyArray<{ href: Route; label: string }> = [
  { href: "/#how-it-works" as Route, label: "How it works" },
  { href: "/#product-architecture" as Route, label: "Workspace" },
  { href: "/#money-movement" as Route, label: "Money movement" },
];

function NavigationLink({
  href,
  label,
  active,
  mobile = false,
  onNavigate,
}: {
  href: Route;
  label: string;
  active: boolean;
  mobile?: boolean;
  onNavigate: (event: MouseEvent<HTMLAnchorElement>, href: Route) => void;
}) {
  const link = (
    <Link
      aria-current={active ? "location" : undefined}
      className={cn(
        "relative flex min-h-11 items-center font-semibold no-underline",
        mobile
          ? "border-b border-white/12 py-4 text-xl text-[#fafcfe]"
          : "px-1 text-[0.9375rem] font-medium text-[#fafcfe]/70 transition-colors hover:text-[#fafcfe]",
        active &&
          !mobile &&
          "text-[#fafcfe] after:absolute after:right-1 after:bottom-0 after:left-1 after:h-px after:bg-white/55",
      )}
      href={href}
      onClick={(event) => onNavigate(event, href)}
    >
      {label}
    </Link>
  );

  return mobile ? <Dialog.Close asChild>{link}</Dialog.Close> : link;
}

export function MarketingHeader() {
  const { status, user } = useAuth();
  const accountActions = marketingAccountActions(status, user);
  const [activeSection, setActiveSection] = useState("");
  const [mobileNavigationOpen, setMobileNavigationOpen] = useState(false);

  useEffect(() => {
    let frame = 0;
    const updateActiveSection = () => {
      frame = 0;
      let current = "";
      for (const item of navigation) {
        const hash = item.href.split("#")[1];
        if (!hash) continue;
        const section = document.getElementById(hash);
        if (section && section.getBoundingClientRect().top <= 176) {
          current = hash;
        }
      }
      setActiveSection((active) => (active === current ? active : current));
    };
    const scheduleUpdate = () => {
      if (frame) return;
      frame = requestAnimationFrame(updateActiveSection);
    };

    const initialSection = window.location.hash.slice(1);
    frame = requestAnimationFrame(() => {
      if (initialSection) {
        document.getElementById(initialSection)?.scrollIntoView();
        window.history.replaceState(
          window.history.state,
          "",
          `${window.location.pathname}${window.location.search}`,
        );
      }
      updateActiveSection();
    });
    window.addEventListener("scroll", scheduleUpdate, { passive: true });
    window.addEventListener("resize", scheduleUpdate);
    window.addEventListener("hashchange", scheduleUpdate);
    return () => {
      if (frame) cancelAnimationFrame(frame);
      window.removeEventListener("scroll", scheduleUpdate);
      window.removeEventListener("resize", scheduleUpdate);
      window.removeEventListener("hashchange", scheduleUpdate);
    };
  }, []);

  function navigateToSection(
    event: MouseEvent<HTMLAnchorElement>,
    href: Route,
  ) {
    if (
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey ||
      window.location.pathname !== "/"
    ) {
      return;
    }

    const sectionID = href.split("#")[1];
    if (!sectionID) return;
    const section = document.getElementById(sectionID);
    if (!section) return;

    event.preventDefault();
    setActiveSection(sectionID);
    setMobileNavigationOpen(false);
    section.scrollIntoView({
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
        ? "auto"
        : "smooth",
      block: "start",
    });
    window.history.replaceState(
      window.history.state,
      "",
      `${window.location.pathname}${window.location.search}`,
    );
  }

  return (
    <header
      className="marketing-header-enter sticky top-0 z-40 border-b border-[var(--marketing-border)] bg-[var(--marketing-navigation)] text-[#fafcfe]"
      style={{ fontFamily: '"DM Sans", sans-serif' }}
    >
      <div className="mx-auto flex h-20 max-w-[90rem] items-center justify-between px-4 sm:px-6 lg:px-12">
        <BrandMark
          className="[&_span]:text-[1.125rem] [&_span]:font-semibold [&_span]:tracking-[-0.02em] [&_svg]:size-[1.875rem]"
          inverse
        />

        <nav
          aria-label="Primary navigation"
          className="hidden items-center gap-[2.125rem] lg:mr-auto lg:ml-[4.375rem] lg:flex"
        >
          {navigation.map((item) => (
            <NavigationLink
              active={item.href.endsWith(`#${activeSection}`)}
              key={item.href}
              onNavigate={navigateToSection}
              {...item}
            />
          ))}
        </nav>

        <div className="hidden min-w-[14.5rem] items-center justify-end gap-[1.125rem] lg:flex">
          {accountActions.loading ? (
            <span
              aria-label="Restoring account session"
              className="h-12 w-36 animate-pulse rounded-full bg-white/8 motion-reduce:animate-none"
              role="status"
            />
          ) : (
            <>
              {accountActions.secondary ? (
                <Link
                  className="text-[0.9375rem] font-medium text-[#fafcfe]/70 no-underline transition-colors hover:text-[#fafcfe]"
                  href={accountActions.secondary.href}
                >
                  {accountActions.secondary.label}
                </Link>
              ) : null}
              {accountActions.primary ? (
                <Link
                  className="inline-flex min-h-12 items-center justify-center rounded-full bg-[#fafcfe] px-6 text-[0.9375rem] font-semibold text-[#111317] no-underline transition-colors hover:bg-white"
                  href={accountActions.primary.href}
                >
                  {accountActions.primary.label}
                </Link>
              ) : null}
            </>
          )}
        </div>

        <Dialog.Root
          onOpenChange={setMobileNavigationOpen}
          open={mobileNavigationOpen}
        >
          <Dialog.Trigger asChild>
            <button
              aria-label="Open navigation"
              className="inline-flex size-11 items-center justify-center rounded-lg border border-white/20 bg-white/6 text-[#fafcfe] lg:hidden"
              type="button"
            >
              <Menu aria-hidden="true" className="size-5" />
            </button>
          </Dialog.Trigger>
          <Dialog.Portal>
            <Dialog.Overlay className="fixed inset-0 z-50 bg-black/48 backdrop-blur-[3px]" />
            <Dialog.Content className="fixed inset-y-0 right-0 z-50 w-[min(24rem,90vw)] overflow-y-auto bg-[#14161a] p-5 text-[#fafcfe] shadow-[-24px_0_64px_rgb(0_0_0_/_34%)]">
              <div className="flex items-center justify-between">
                <Dialog.Title className="font-serif text-[1.375rem] font-bold">
                  Navigation
                </Dialog.Title>
                <Dialog.Close asChild>
                  <button
                    aria-label="Close navigation"
                    className="inline-flex size-11 items-center justify-center rounded-lg border border-white/20 bg-white/5 text-[#fafcfe]"
                    type="button"
                  >
                    <X aria-hidden="true" className="size-5" />
                  </button>
                </Dialog.Close>
              </div>
              <Dialog.Description className="mt-2 text-sm text-[#dee2e6]/78">
                Explore Monierave or continue to your account.
              </Dialog.Description>
              <nav aria-label="Mobile navigation" className="mt-8">
                {navigation.map((item) => (
                  <NavigationLink
                    active={item.href.endsWith(`#${activeSection}`)}
                    key={item.href}
                    onNavigate={navigateToSection}
                    {...item}
                    mobile
                  />
                ))}
              </nav>
              <div className="mt-8 grid gap-3">
                {accountActions.loading ? (
                  <span
                    className="h-12 animate-pulse rounded-full bg-white/8 motion-reduce:animate-none"
                    role="status"
                  >
                    <span className="sr-only">Restoring account session</span>
                  </span>
                ) : (
                  <>
                    {accountActions.secondary ? (
                      <Dialog.Close asChild>
                        <Link
                          className="inline-flex min-h-12 items-center justify-center rounded-full border border-white/12 bg-transparent px-5 text-center text-[0.9375rem] font-semibold text-[#fafcfe] no-underline"
                          href={accountActions.secondary.href}
                        >
                          {accountActions.secondary.label}
                        </Link>
                      </Dialog.Close>
                    ) : null}
                    {accountActions.primary ? (
                      <Dialog.Close asChild>
                        <Link
                          className="inline-flex min-h-12 items-center justify-center rounded-full bg-[#fafcfe] px-5 text-center text-[0.9375rem] font-semibold text-[#111317] no-underline"
                          href={accountActions.primary.href}
                        >
                          {accountActions.primary.label}
                        </Link>
                      </Dialog.Close>
                    ) : null}
                  </>
                )}
              </div>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
      </div>
    </header>
  );
}
