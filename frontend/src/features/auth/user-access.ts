import type { User } from "@/lib/api/contracts";

const limitedApplicationPaths = ["/app/profile", "/app/security"] as const;

export function hasFinancialAccess(user: User | null | undefined) {
  return Boolean(user?.account_status === "active" && user.email_verified_at);
}

export function canAccessApplicationPath(
  user: User | null | undefined,
  pathname: string,
) {
  if (hasFinancialAccess(user) || pathname === "/app") {
    return true;
  }

  return limitedApplicationPaths.some(
    (path) => pathname === path || pathname.startsWith(`${path}/`),
  );
}
