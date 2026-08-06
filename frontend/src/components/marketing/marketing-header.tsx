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
  { href: "/", label: "Home" },
  { href: "/security", label: "Security" },
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
          : "text-ink-700 hover:text-evergreen-800 px-1 text-sm",
        active &&
          !mobile &&
          "text-evergreen-900 after:bg-jade-500 after:absolute after:right-1 after:bottom-0 after:left-1 after:h-0.5",
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
    <header className="border-line-200 bg-paper-50/95 sticky top-0 z-40 border-b backdrop-blur-md">
      <div className="mx-auto flex h-18 max-w-[90rem] items-center justify-between px-5 sm:h-20 sm:px-8 xl:px-12">
        <BrandMark />

        <nav
          aria-label="Primary navigation"
          className="hidden items-center gap-8 md:flex"
        >
          {navigation.map((item) => (
            <NavigationLink key={item.href} {...item} />
          ))}
        </nav>

        <div className="hidden items-center gap-2 md:flex">
          <Button asChild variant="quiet">
            <a href="/login">Sign in</a>
          </Button>
          <Button asChild>
            <a href="/signup">Create account</a>
          </Button>
        </div>

        <Dialog.Root>
          <Dialog.Trigger asChild>
            <button
              aria-label="Open navigation"
              className="border-line-300 text-evergreen-900 inline-flex size-11 items-center justify-center rounded-sm border bg-white md:hidden"
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
