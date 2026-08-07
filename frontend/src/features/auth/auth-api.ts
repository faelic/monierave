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

export function clearAccessCredential() {
  accessCredential = null;
}

export function setAccessCredential(token: string, expiresAt: string) {
  accessCredential = {
    expiresAt: new Date(expiresAt).getTime(),
    token,
  };
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
