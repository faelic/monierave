import type { Route } from "next";

import type { User } from "@/lib/api/contracts";

export type MarketingSessionStatus =
  "restoring" | "authenticated" | "anonymous" | "unavailable";

export type MarketingAccountAction = {
  href: Route;
  label: string;
};

export function marketingAccountActions(
  status: MarketingSessionStatus,
  user: User | null,
): {
  loading: boolean;
  primary?: MarketingAccountAction;
  secondary?: MarketingAccountAction;
} {
  if (status === "restoring") return { loading: true };
  if (status === "authenticated" && user) {
    const label =
      user.account_status === "pending"
        ? "Continue verification"
        : user.account_status === "disabled"
          ? "Review account"
          : "Open dashboard";
    return { loading: false, primary: { href: "/app" as Route, label } };
  }
  if (status === "unavailable") {
    return {
      loading: false,
      primary: { href: "/app" as Route, label: "Open app" },
    };
  }
  return {
    loading: false,
    primary: { href: "/signup" as Route, label: "Get started" },
    secondary: { href: "/login" as Route, label: "Sign in" },
  };
}
