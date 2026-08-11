import { apiRequest, type ApiRequestOptions } from "@/lib/api/client";
import type {
  EmailStatus,
  LoginResponse,
  RenewAccessResponse,
  User,
} from "@/lib/api/contracts";

export type SignupInput = {
  username: string;
  password: string;
  full_name: string;
  email: string;
};

export type LoginInput = {
  username: string;
  password: string;
};

export type UpdateUserInput = {
  full_name?: string;
  email?: string;
  current_password?: string;
  password?: string;
};

type AccessCredential = {
  expiresAt: number;
  token: string;
};

let accessCredential: AccessCredential | null = null;
let refreshInFlight: Promise<AccessCredential> | null = null;
const refreshLockName = "monierave-refresh-token-rotation";
const refreshLeaseKey = "monierave-refresh-token-lease";
const refreshLeaseDurationMs = 30_000;
const authChannel =
  typeof window !== "undefined" && "BroadcastChannel" in window
    ? new BroadcastChannel("monierave-auth")
    : null;

authChannel?.addEventListener("message", (event: MessageEvent) => {
  if (event.data?.type === "credential") {
    setAccessCredentialInternal(event.data.token, event.data.expiresAt, false);
  } else if (event.data?.type === "logout") {
    accessCredential = null;
    notifySessionExpired();
  }
});

export const sessionExpiredEvent = "monierave:session-expired";

export function registerUser(input: SignupInput) {
  return apiRequest<User, SignupInput>("/users", {
    body: input,
    method: "POST",
  });
}

export async function loginUser(input: LoginInput) {
  const response = await apiRequest<LoginResponse, LoginInput>("/users/login", {
    body: input,
    method: "POST",
  });
  setAccessCredential(response.access_token, response.access_token_expires_at);
  return response.user;
}

export function clearAccessCredential(broadcast = true) {
  accessCredential = null;
  if (broadcast) {
    authChannel?.postMessage({ type: "logout" });
  }
}

export function setAccessCredential(token: string, expiresAt: string) {
  setAccessCredentialInternal(token, expiresAt, true);
}

function setAccessCredentialInternal(
  token: string,
  expiresAt: string,
  broadcast: boolean,
) {
  accessCredential = {
    expiresAt: new Date(expiresAt).getTime(),
    token,
  };
  if (broadcast) {
    authChannel?.postMessage({ type: "credential", token, expiresAt });
  }
}

function currentAccessToken() {
  if (
    accessCredential &&
    Number.isFinite(accessCredential.expiresAt) &&
    accessCredential.expiresAt - Date.now() > 5_000
  ) {
    return accessCredential.token;
  }
  return null;
}

export async function renewAccessToken(force = false) {
  if (!force) {
    const current = currentAccessToken();
    if (current) {
      return current;
    }
  }

  return withCrossTabRefreshLock(force);
}

async function withCrossTabRefreshLock(force: boolean) {
  const tokenBeforeLock = accessCredential?.token;
  if (typeof navigator !== "undefined" && navigator.locks) {
    return navigator.locks.request(refreshLockName, async () => {
      const current = currentAccessToken();
      if (current && (!force || current !== tokenBeforeLock)) return current;
      return performRefresh(force);
    });
  }
  const storage = availableBrowserStorage();
  if (storage) {
    return withStorageRefreshLease(storage, tokenBeforeLock, force);
  }
  return performRefresh(force);
}

async function withStorageRefreshLease(
  storage: Storage,
  tokenBeforeLock: string | undefined,
  force: boolean,
) {
  const owner = crypto.randomUUID();
  const deadline = Date.now() + refreshLeaseDurationMs;

  while (Date.now() < deadline) {
    const lease = readRefreshLease(storage);
    if (!lease || lease.expiresAt <= Date.now()) {
      try {
        storage.setItem(
          refreshLeaseKey,
          JSON.stringify({
            expiresAt: Date.now() + refreshLeaseDurationMs,
            owner,
          }),
        );
      } catch {
        return performRefresh(force);
      }
      await delay(25);
      if (readRefreshLease(storage)?.owner === owner) {
        try {
          const current = currentAccessToken();
          if (current && (!force || current !== tokenBeforeLock))
            return current;
          return await performRefresh(force);
        } finally {
          if (readRefreshLease(storage)?.owner === owner) {
            storage.removeItem(refreshLeaseKey);
          }
        }
      }
    }
    await delay(50);
    const current = currentAccessToken();
    if (current && (!force || current !== tokenBeforeLock)) return current;
  }

  throw new Error(
    "Timed out coordinating session refresh across browser tabs.",
  );
}

function availableBrowserStorage(): Storage | null {
  if (typeof window === "undefined") return null;
  try {
    const storage = window.localStorage;
    return typeof storage?.setItem === "function" &&
      typeof storage?.getItem === "function" &&
      typeof storage?.removeItem === "function"
      ? storage
      : null;
  } catch {
    return null;
  }
}

function readRefreshLease(
  storage: Storage,
): { expiresAt: number; owner: string } | null {
  try {
    const value = storage.getItem(refreshLeaseKey);
    if (!value) return null;
    const parsed = JSON.parse(value) as {
      expiresAt?: unknown;
      owner?: unknown;
    };
    if (
      typeof parsed.expiresAt !== "number" ||
      typeof parsed.owner !== "string"
    ) {
      return null;
    }
    return { expiresAt: parsed.expiresAt, owner: parsed.owner };
  } catch {
    return null;
  }
}

function delay(milliseconds: number) {
  return new Promise((resolve) => window.setTimeout(resolve, milliseconds));
}

async function performRefresh(force = false) {
  if (!force) {
    const current = currentAccessToken();
    if (current) return current;
  }
  if (!refreshInFlight) {
    refreshInFlight = apiRequest<RenewAccessResponse>("/tokens/renew_access", {
      method: "POST",
    })
      .then((response) => {
        setAccessCredential(
          response.access_token,
          response.access_token_expires_at,
        );
        return accessCredential as AccessCredential;
      })
      .catch((error: unknown) => {
        clearAccessCredential();
        throw error;
      })
      .finally(() => {
        refreshInFlight = null;
      });
  }

  return (await refreshInFlight).token;
}

export async function authenticatedRequest<TResponse, TBody = never>(
  path: `/${string}`,
  options: ApiRequestOptions<TBody> = {},
) {
  const requestToken = currentAccessToken() ?? (await renewAccessToken());

  try {
    return await apiRequest<TResponse, TBody>(path, {
      ...options,
      accessToken: requestToken,
    });
  } catch (error) {
    if (!isUnauthorized(error)) {
      throw error;
    }

    const newerToken = currentAccessToken();
    const retryToken =
      newerToken && newerToken !== requestToken
        ? newerToken
        : await renewAccessToken(true);

    try {
      return await apiRequest<TResponse, TBody>(path, {
        ...options,
        accessToken: retryToken,
      });
    } catch (retryError) {
      if (isUnauthorized(retryError)) {
        clearAccessCredential();
        notifySessionExpired();
      }
      throw retryError;
    }
  }
}

export function getCurrentUser() {
  return authenticatedRequest<User>("/users/me");
}

export function getEmailStatus() {
  return authenticatedRequest<EmailStatus>("/users/me/email-status");
}

export function updateCurrentUser(input: UpdateUserInput) {
  return authenticatedRequest<User, UpdateUserInput>("/users/me", {
    body: input,
    method: "PATCH",
  });
}

export function requestVerificationEmail() {
  return authenticatedRequest<{ job_id: string; message: string }>(
    "/users/me/resend-verification",
    { method: "POST" },
  );
}

export async function logoutCurrentSession() {
  try {
    await apiRequest<void>("/sessions/logout", { method: "POST" });
  } finally {
    clearAccessCredential();
  }
}

export async function logoutAllSessions() {
  try {
    await authenticatedRequest<void>("/sessions/logout-all", {
      method: "POST",
    });
  } finally {
    clearAccessCredential();
  }
}

function isUnauthorized(error: unknown) {
  return (
    typeof error === "object" &&
    error !== null &&
    "status" in error &&
    error.status === 401
  );
}

function notifySessionExpired() {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event(sessionExpiredEvent));
  }
}
