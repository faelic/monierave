"use client";

import * as Dialog from "@radix-ui/react-dialog";
import {
  CircleHelp,
  Home,
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
import { cn } from "@/lib/utils/cn";

const navigation = [
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

export function BankingShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const online = useSyncExternalStore(subscribeOnline, readOnline, () => true);
  const { logout, user } = useAuth();

  async function signOut() {
    try {
      await logout();
    } finally {
      router.replace("/login");
    }
  }

  return (
    <div className="bg-paper-50 min-h-screen lg:grid lg:grid-cols-[16rem_minmax(0,1fr)]">
      <SkipLink />
      <aside className="bg-evergreen-900 text-paper-50 fixed inset-y-0 left-0 z-30 hidden w-64 border-r border-white/10 px-5 py-6 lg:flex lg:flex-col">
        <BrandMark inverse />
        <nav aria-label="Primary navigation" className="mt-12">
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
                      "flex min-h-11 items-center gap-3 rounded-sm px-3 text-sm font-semibold no-underline",
                      active
                        ? "bg-white/12 text-white"
                        : "text-white/70 hover:bg-white/8 hover:text-white",
                    )}
                    href={item.href}
                  >
                    <item.icon aria-hidden="true" className="size-5" />
                    {item.label}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>
        <nav
          aria-label="Profile and support navigation"
          className="mt-7 border-t border-white/10 pt-5"
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
              { href: "/help" as Route, icon: CircleHelp, label: "Help" },
            ].map((item) => (
              <li key={item.href}>
                <Link
                  className="flex min-h-11 items-center gap-3 rounded-sm px-3 text-sm font-semibold text-white/65 no-underline hover:bg-white/8 hover:text-white"
                  href={item.href}
                >
                  <item.icon aria-hidden="true" className="size-5" />
                  {item.label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
        <div className="mt-auto border-t border-white/10 pt-5">
          <p className="truncate font-semibold">{user?.full_name}</p>
          <p className="truncate text-sm text-white/55">@{user?.username}</p>
          <button
            className="mt-4 flex min-h-11 w-full items-center gap-3 rounded-sm px-3 text-left text-sm font-semibold text-white/75 hover:bg-white/8 hover:text-white"
            onClick={() => void signOut()}
            type="button"
          >
            <LogOut aria-hidden="true" className="size-5" />
            Sign out
          </button>
        </div>
      </aside>

      <div className="min-w-0 lg:col-start-2">
        <header className="border-line-200 sticky top-0 z-20 flex min-h-16 items-center justify-between border-b bg-white/95 px-5 backdrop-blur-sm lg:hidden">
          <BrandMark />
          <MobileMenu onSignOut={signOut} userName={user?.full_name} />
        </header>
        {!online ? (
          <div
            className="bg-warning-700 px-5 py-2 text-center text-sm font-semibold text-white"
            role="status"
          >
            You’re offline. Financial information may be out of date.
          </div>
        ) : null}
        <main
          className="mx-auto max-w-7xl min-w-0 px-5 pt-8 pb-28 sm:px-8 lg:px-10 lg:pt-10 lg:pb-12"
          id="main-content"
        >
          {children}
        </main>
        <nav
          aria-label="Mobile primary navigation"
          className="border-line-200 fixed inset-x-0 bottom-0 z-20 grid min-h-16 grid-cols-5 border-t bg-white px-[max(0.25rem,env(safe-area-inset-left))] pb-[env(safe-area-inset-bottom)] lg:hidden"
        >
          {navigation.slice(0, 4).map((item) => {
            const active =
              pathname === item.href ||
              (item.href !== "/app" && pathname.startsWith(item.href));
            return (
              <Link
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex min-h-16 flex-col items-center justify-center gap-1 text-[0.6875rem] font-semibold no-underline",
                  active ? "text-evergreen-800" : "text-ink-600",
                )}
                href={item.href}
                key={item.href}
              >
                <item.icon aria-hidden="true" className="size-5" />
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
  onSignOut,
  userName,
}: {
  bottomNavigation?: boolean;
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
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/35" />
        <Dialog.Content className="bg-paper-50 fixed inset-x-0 bottom-0 z-50 rounded-t-lg px-5 pt-5 pb-[max(1.5rem,env(safe-area-inset-bottom))] shadow-2xl">
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
            <Dialog.Close asChild>
              <Link
                className="hover:bg-paper-100 flex min-h-11 items-center gap-3 rounded-sm px-3 font-semibold no-underline"
                href={"/app/beneficiaries" as Route}
              >
                <Users aria-hidden="true" className="size-5" />
                Beneficiaries
              </Link>
            </Dialog.Close>
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
            <Dialog.Close asChild>
              <Link
                className="hover:bg-paper-100 flex min-h-11 items-center gap-3 rounded-sm px-3 font-semibold no-underline"
                href={"/help" as Route}
              >
                <CircleHelp aria-hidden="true" className="size-5" />
                Help
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
