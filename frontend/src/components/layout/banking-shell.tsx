"use client";

import * as Dialog from "@radix-ui/react-dialog";
import {
  Home,
  MailCheck,
  LogOut,
  Menu,
  ReceiptText,
  Send,
  ShieldCheck,
  Users,
  UserRound,
  WalletCards,
  X,
} from "lucide-react";
import type { Route } from "next";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useSyncExternalStore, type ReactNode } from "react";

import { BrandMark } from "@/components/marketing/brand-mark";
import { Button } from "@/components/ui/button";
import { SkipLink } from "@/components/ui/skip-link";
import { useAuth } from "@/features/auth/auth-provider";
import { hasFinancialAccess } from "@/features/auth/user-access";
import { cn } from "@/lib/utils/cn";

const financialNavigation = [
  { href: "/app" as Route, icon: Home, label: "Overview" },
  { href: "/app/accounts" as Route, icon: WalletCards, label: "Accounts" },
  { href: "/app/transfers/new" as Route, icon: Send, label: "Send money" },
  {
    href: "/app/transactions" as Route,
    icon: ReceiptText,
    label: "Transactions",
  },
  { href: "/app/beneficiaries" as Route, icon: Users, label: "Beneficiaries" },
];

const limitedNavigation = [
  { href: "/app" as Route, icon: Home, label: "Overview" },
  {
    href: "/verification-needed" as Route,
    icon: MailCheck,
    label: "Verify email",
  },
];

export function BankingShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const online = useSyncExternalStore(subscribeOnline, readOnline, () => true);
  const { logout, user } = useAuth();
  const financialAccess = hasFinancialAccess(user);
  const navigation = financialAccess ? financialNavigation : limitedNavigation;

  async function signOut() {
    try {
      await logout();
    } finally {
      router.replace("/login");
    }
  }

  return (
    <div className="banking-workspace min-h-screen bg-[var(--product-canvas)] lg:grid lg:grid-cols-[15rem_minmax(0,1fr)]">
      <SkipLink />
      <aside className="workspace-sidebar fixed inset-y-0 left-0 z-30 hidden w-60 flex-col border-r px-3 py-3 text-white lg:flex">
        <div className="workspace-brand flex min-h-11 items-center rounded-md px-2">
          <BrandMark inverse />
        </div>
        <p className="mt-5 px-2 text-[0.625rem] font-semibold tracking-[0.13em] text-[var(--product-text-faint)] uppercase">
          {financialAccess ? "Banking workspace" : "Profile workspace"}
        </p>
        <nav aria-label="Primary navigation" className="mt-2">
          <ul className="grid gap-1">
            {navigation.map((item) => {
              const active =
                pathname === item.href ||
                (item.href !== "/app" && pathname.startsWith(item.href));
              return (
                <li key={item.href}>
                  <Link
                    aria-current={active ? "page" : undefined}
                    className={cn(
                      "workspace-nav-link group flex min-h-9 items-center gap-2.5 rounded-md px-2.5 text-[0.8125rem] font-medium no-underline",
                      active
                        ? "workspace-nav-link-active text-white"
                        : "text-white/58 hover:text-white",
                    )}
                    href={item.href}
                  >
                    <item.icon
                      aria-hidden="true"
                      className="workspace-nav-icon size-4"
                    />
                    {item.label}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>
        <nav
          aria-label="Profile and support navigation"
          className="mt-5 border-t border-white/8 pt-4"
        >
          <ul className="grid gap-1">
            {[
              {
                href: "/app/profile" as Route,
                icon: UserRound,
                label: "Profile",
              },
              {
                href: "/app/security" as Route,
                icon: ShieldCheck,
                label: "Security",
              },
            ].map((item) => (
              <li key={item.href}>
                <Link
                  className="workspace-nav-link group flex min-h-9 items-center gap-2.5 rounded-md px-2.5 text-[0.8125rem] font-medium text-white/58 no-underline hover:text-white"
                  href={item.href}
                >
                  <item.icon
                    aria-hidden="true"
                    className="workspace-nav-icon size-4"
                  />
                  {item.label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
        <div className="workspace-user mt-auto rounded-md border p-2.5">
          <div className="flex items-center gap-2.5">
            <span className="workspace-avatar grid size-7 shrink-0 place-items-center rounded-md text-xs font-semibold">
              {initials(user?.full_name)}
            </span>
            <div className="min-w-0">
              <p className="truncate text-xs font-medium text-white/90">
                {user?.full_name}
              </p>
              <p className="truncate text-[0.6875rem] text-white/42">
                @{user?.username}
              </p>
            </div>
          </div>
          <button
            className="workspace-nav-link group mt-2 flex min-h-9 w-full items-center gap-2.5 rounded-md px-2.5 text-left text-xs font-medium text-white/55 hover:text-white"
            onClick={() => void signOut()}
            type="button"
          >
            <LogOut aria-hidden="true" className="workspace-nav-icon size-4" />
            Sign out
          </button>
        </div>
      </aside>

      <div className="min-w-0 lg:col-start-2">
        <header className="workspace-mobile-header sticky top-0 z-20 flex min-h-14 items-center justify-between border-b px-4 backdrop-blur-sm lg:hidden">
          <BrandMark inverse />
          <MobileMenu
            financialAccess={financialAccess}
            onSignOut={signOut}
            userName={user?.full_name}
          />
        </header>
        {!online ? (
          <div
            className="bg-warning-700 px-5 py-2 text-center text-sm font-semibold text-white"
            role="status"
          >
            You’re offline. Financial information may be out of date.
          </div>
        ) : null}
        {!financialAccess ? (
          <div className="workspace-verification-banner flex items-center justify-between gap-4 border-b px-5 py-2.5 text-xs sm:px-8 lg:px-8">
            <p className="font-semibold">
              Banking features are locked until your email is verified.
            </p>
            <Link
              className="shrink-0 font-semibold text-[var(--product-accent)] underline underline-offset-4"
              href="/verification-needed"
            >
              Verify email
            </Link>
          </div>
        ) : null}
        <main
          className="workspace-main min-w-0 px-5 pt-7 pb-28 sm:px-8 lg:px-8 lg:pt-8 lg:pb-12 xl:px-10"
          id="main-content"
        >
          {children}
        </main>
        <nav
          aria-label="Mobile primary navigation"
          className={cn(
            "workspace-mobile-nav fixed inset-x-0 bottom-0 z-20 grid min-h-16 border-t px-[max(0.25rem,env(safe-area-inset-left))] pb-[env(safe-area-inset-bottom)] lg:hidden",
            financialAccess ? "grid-cols-5" : "grid-cols-3",
          )}
        >
          {navigation.slice(0, financialAccess ? 4 : 2).map((item) => {
            const active =
              pathname === item.href ||
              (item.href !== "/app" && pathname.startsWith(item.href));
            return (
              <Link
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex min-h-16 flex-col items-center justify-center gap-1 text-[0.6875rem] font-medium no-underline",
                  active
                    ? "text-[var(--product-text)]"
                    : "text-[var(--product-text-muted)]",
                )}
                href={item.href}
                key={item.href}
              >
                <item.icon
                  aria-hidden="true"
                  className="workspace-nav-icon size-5"
                />
                {item.label === "Send money"
                  ? "Transfer"
                  : item.label === "Transactions"
                    ? "Activity"
                    : item.label}
              </Link>
            );
          })}
          <MobileMenu
            bottomNavigation
            financialAccess={financialAccess}
            onSignOut={signOut}
            userName={user?.full_name}
          />
        </nav>
      </div>
    </div>
  );
}

function MobileMenu({
  bottomNavigation = false,
  financialAccess = true,
  onSignOut,
  userName,
}: {
  bottomNavigation?: boolean;
  financialAccess?: boolean;
  onSignOut: () => Promise<void>;
  userName?: string | undefined;
}) {
  return (
    <Dialog.Root>
      <Dialog.Trigger asChild>
        <button
          aria-label={bottomNavigation ? undefined : "Open application menu"}
          className={cn(
            "text-evergreen-800 min-h-11 min-w-11 items-center justify-center",
            bottomNavigation
              ? "flex min-h-16 flex-col gap-1 text-xs font-semibold"
              : "flex rounded-sm border border-transparent",
          )}
          type="button"
        >
          <Menu aria-hidden="true" className="size-5" />
          {bottomNavigation ? "More" : null}
        </button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/70 backdrop-blur-[2px]" />
        <Dialog.Content className="banking-workspace-dialog fixed inset-x-0 bottom-0 z-50 rounded-t-lg border-t px-5 pt-5 pb-[max(1.5rem,env(safe-area-inset-bottom))] shadow-2xl">
          <div className="flex items-center justify-between">
            <div>
              <Dialog.Title className="text-xl font-semibold">
                Account menu
              </Dialog.Title>
              <Dialog.Description className="text-ink-600 mt-1 text-sm">
                Signed in as {userName ?? "Monierave customer"}
              </Dialog.Description>
            </div>
            <Dialog.Close asChild>
              <button
                aria-label="Close application menu"
                className="grid min-h-11 min-w-11 place-items-center"
                type="button"
              >
                <X aria-hidden="true" className="size-5" />
              </button>
            </Dialog.Close>
          </div>
          <div className="border-line-200 mt-6 grid gap-2 border-t pt-5">
            {financialAccess ? (
              <Dialog.Close asChild>
                <Link
                  className="hover:bg-paper-100 flex min-h-11 items-center gap-3 rounded-sm px-3 font-semibold no-underline"
                  href={"/app/beneficiaries" as Route}
                >
                  <Users aria-hidden="true" className="size-5" />
                  Beneficiaries
                </Link>
              </Dialog.Close>
            ) : (
              <Dialog.Close asChild>
                <Link
                  className="hover:bg-paper-100 flex min-h-11 items-center gap-3 rounded-sm px-3 font-semibold no-underline"
                  href={"/verification-needed" as Route}
                >
                  <MailCheck aria-hidden="true" className="size-5" />
                  Verify email
                </Link>
              </Dialog.Close>
            )}
            <Dialog.Close asChild>
              <Link
                className="hover:bg-paper-100 flex min-h-11 items-center gap-3 rounded-sm px-3 font-semibold no-underline"
                href={"/app/profile" as Route}
              >
                <UserRound aria-hidden="true" className="size-5" />
                Profile
              </Link>
            </Dialog.Close>
            <Dialog.Close asChild>
              <Link
                className="hover:bg-paper-100 flex min-h-11 items-center gap-3 rounded-sm px-3 font-semibold no-underline"
                href={"/app/security" as Route}
              >
                <ShieldCheck aria-hidden="true" className="size-5" />
                Security
              </Link>
            </Dialog.Close>
            <Button
              className="justify-start"
              onClick={() => void onSignOut()}
              variant="secondary"
            >
              <LogOut aria-hidden="true" className="size-5" />
              Sign out
            </Button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function subscribeOnline(notify: () => void) {
  window.addEventListener("online", notify);
  window.addEventListener("offline", notify);
  return () => {
    window.removeEventListener("online", notify);
    window.removeEventListener("offline", notify);
  };
}

function readOnline() {
  return navigator.onLine;
}

function initials(name?: string) {
  if (!name) return "M";
  return name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("");
}
