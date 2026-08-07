"use client";

import * as Dialog from "@radix-ui/react-dialog";
import { Menu, X } from "lucide-react";
import type { Route } from "next";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { BrandMark } from "@/components/marketing/brand-mark";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils/cn";

const navigation: ReadonlyArray<{ href: Route; label: string }> = [
  { href: "/#product" as Route, label: "Product" },
  { href: "/#how-it-works" as Route, label: "How it works" },
  { href: "/security", label: "Security" },
  { href: "/help", label: "Help" },
];

function NavigationLink({
  href,
  label,
  mobile = false,
}: {
  href: Route;
  label: string;
  mobile?: boolean;
}) {
  const pathname = usePathname();
  const active = pathname === href;

  const link = (
    <Link
      aria-current={active ? "page" : undefined}
      className={cn(
        "relative flex min-h-11 items-center font-semibold no-underline",
        mobile
          ? "border-line-200 text-ink-950 border-b py-4 text-xl"
          : "px-1 text-xs text-white/58 transition-colors hover:text-white",
        active &&
          !mobile &&
          "text-white after:absolute after:right-1 after:bottom-0 after:left-1 after:h-px after:bg-white/55",
      )}
      href={href}
    >
      {label}
    </Link>
  );

  return mobile ? <Dialog.Close asChild>{link}</Dialog.Close> : link;
}

export function MarketingHeader() {
  return (
    <header className="bg-evergreen-900 sticky top-0 z-40 text-white">
      <div className="mx-auto flex h-19 max-w-[90rem] items-center justify-between px-5 sm:px-8 lg:px-[4.625rem]">
        <BrandMark inverse />

        <nav
          aria-label="Primary navigation"
          className="hidden items-center gap-7 md:flex lg:mr-auto lg:ml-20"
        >
          {navigation.map((item) => (
            <NavigationLink key={item.href} {...item} />
          ))}
        </nav>

        <div className="hidden items-center gap-2 md:flex">
          <Button
            asChild
            className="min-h-9 px-3 text-xs text-white/70 hover:bg-white/8 hover:text-white"
            size="compact"
            variant="quiet"
          >
            <a href="/login">Sign in</a>
          </Button>
          <Button
            asChild
            className="min-h-9 rounded-full bg-white px-4 text-xs text-black hover:bg-white/88"
            size="compact"
            variant="secondary"
          >
            <a href="/signup">Create account</a>
          </Button>
        </div>

        <Dialog.Root>
          <Dialog.Trigger asChild>
            <button
              aria-label="Open navigation"
              className="inline-flex size-11 items-center justify-center rounded-sm border border-white/20 bg-white/6 text-white md:hidden"
              type="button"
            >
              <Menu aria-hidden="true" className="size-5" />
            </button>
          </Dialog.Trigger>
          <Dialog.Portal>
            <Dialog.Overlay className="fixed inset-0 z-50 bg-black/25 backdrop-blur-[2px]" />
            <Dialog.Content className="bg-paper-50 border-line-200 fixed inset-y-0 right-0 z-50 w-[min(24rem,90vw)] overflow-y-auto border-l p-5 shadow-[var(--shadow-2)]">
              <div className="flex items-center justify-between">
                <Dialog.Title className="text-lg font-bold">
                  Navigation
                </Dialog.Title>
                <Dialog.Close asChild>
                  <button
                    aria-label="Close navigation"
                    className="border-line-300 inline-flex size-11 items-center justify-center rounded-sm border bg-white"
                    type="button"
                  >
                    <X aria-hidden="true" className="size-5" />
                  </button>
                </Dialog.Close>
              </div>
              <Dialog.Description className="text-ink-600 mt-2 text-sm">
                Explore Monierave or continue to your account.
              </Dialog.Description>
              <nav aria-label="Mobile navigation" className="mt-8">
                {navigation.map((item) => (
                  <NavigationLink key={item.href} {...item} mobile />
                ))}
              </nav>
              <div className="mt-8 grid gap-3">
                <Dialog.Close asChild>
                  <Button asChild variant="secondary">
                    <a href="/login">Sign in</a>
                  </Button>
                </Dialog.Close>
                <Dialog.Close asChild>
                  <Button asChild>
                    <a href="/signup">Create account</a>
                  </Button>
                </Dialog.Close>
              </div>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
      </div>
    </header>
  );
}
