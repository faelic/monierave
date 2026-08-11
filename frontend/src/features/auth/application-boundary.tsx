"use client";

import type { Route } from "next";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { useAuth } from "@/features/auth/auth-provider";
import { canAccessApplicationPath } from "@/features/auth/user-access";

export function ApplicationBoundary({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const { restore, status, user } = useAuth();

  useEffect(() => {
    const currentPath = `${pathname}${
      searchParams.size ? `?${searchParams.toString()}` : ""
    }`;
    if (status === "anonymous") {
      router.replace(
        `/login?returnTo=${encodeURIComponent(currentPath)}` as Route,
      );
      return;
    }
    if (
      status === "authenticated" &&
      !canAccessApplicationPath(user, pathname)
    ) {
      router.replace("/app?access=verification-required" as Route);
    }
  }, [pathname, router, searchParams, status, user]);

  if (
    status === "restoring" ||
    status === "anonymous" ||
    (status === "authenticated" && !canAccessApplicationPath(user, pathname))
  ) {
    return <ApplicationSkeleton />;
  }

  if (status === "unavailable") {
    return (
      <main
        className="grid min-h-screen place-items-center px-5 py-12"
        id="main-content"
      >
        <div className="max-w-md text-center">
          <p className="text-evergreen-700 text-sm font-bold tracking-[0.14em] uppercase">
            Secure session check
          </p>
          <h1 className="mt-3 text-3xl font-semibold">
            We could not open your banking view.
          </h1>
          <p className="text-ink-600 mt-4 leading-7">
            No financial information has been loaded. Check your connection and
            retry the secure session check.
          </p>
          <Button className="mt-7" onClick={() => void restore()}>
            Retry session check
          </Button>
        </div>
      </main>
    );
  }

  return children;
}

function ApplicationSkeleton() {
  return (
    <main
      aria-busy="true"
      aria-label="Restoring your secure banking session"
      className="min-h-screen bg-white"
    >
      <div className="bg-evergreen-900 h-16 lg:fixed lg:inset-y-0 lg:h-auto lg:w-64" />
      <div className="mx-auto max-w-7xl px-5 py-10 lg:ml-64 lg:px-10">
        <div className="bg-paper-100 h-5 w-24 animate-pulse rounded motion-reduce:animate-none" />
        <div className="bg-paper-100 mt-5 h-12 w-64 animate-pulse rounded motion-reduce:animate-none" />
        <div className="bg-paper-100 mt-10 h-64 w-full animate-pulse rounded-md motion-reduce:animate-none" />
        <p className="sr-only">Restoring your secure banking session.</p>
      </div>
    </main>
  );
}
