import { HttpResponse, http } from "msw";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  authenticatedRequest,
  clearAccessCredential,
  loginUser,
  logoutCurrentSession,
  renewAccessToken,
  sessionExpiredEvent,
} from "@/features/auth/auth-api";
import type { User } from "@/lib/api/contracts";
import { mockServer } from "@/test/mocks/server";

const user: User = {
  account_status: "active",
  created_at: "2026-08-01T00:00:00Z",
  email: "favour@example.com",
  email_bounced_at: null,
  email_deliverability_status: "deliverable",
  email_verified_at: "2026-08-01T00:10:00Z",
  full_name: "Favour Ututu",
  password_changed_at: "2026-08-01T00:00:00Z",
  registration_expires_at: null,
  username: "favour",
};

afterEach(() => clearAccessCredential());

describe("auth API session handling", () => {
  it("keeps login access tokens in memory for authenticated requests", async () => {
    mockServer.use(
      http.post("http://localhost:8080/users/login", () =>
        HttpResponse.json({
          access_token: "login-token",
          access_token_expires_at: "2099-01-01T00:00:00Z",
          user,
        }),
      ),
      http.get("http://localhost:8080/protected", ({ request }) => {
        expect(request.headers.get("Authorization")).toBe("Bearer login-token");
        return HttpResponse.json({ ok: true });
      }),
    );

    await loginUser({ password: "password123", username: "favour" });
    await expect(
      authenticatedRequest<{ ok: boolean }>("/protected"),
    ).resolves.toEqual({ ok: true });
  });

  it("deduplicates concurrent refresh attempts", async () => {
    let refreshCount = 0;
    mockServer.use(
      http.post("http://localhost:8080/tokens/renew_access", async () => {
        refreshCount += 1;
        await new Promise((resolve) => setTimeout(resolve, 20));
        return HttpResponse.json({
          access_token: "restored-token",
          access_token_expires_at: "2099-01-01T00:00:00Z",
        });
      }),
    );

    const tokens = await Promise.all([
      renewAccessToken(),
      renewAccessToken(),
      renewAccessToken(),
    ]);

    expect(tokens).toEqual([
      "restored-token",
      "restored-token",
      "restored-token",
    ]);
    expect(refreshCount).toBe(1);
  });

  it("refreshes once after a protected request rejects an old token", async () => {
    let refreshCount = 0;
    let protectedCount = 0;
    mockServer.use(
      http.post("http://localhost:8080/users/login", () =>
        HttpResponse.json({
          access_token: "old-token",
          access_token_expires_at: "2099-01-01T00:00:00Z",
          user,
        }),
      ),
      http.post("http://localhost:8080/tokens/renew_access", () => {
        refreshCount += 1;
        return HttpResponse.json({
          access_token: "rotated-token",
          access_token_expires_at: "2099-01-01T00:00:00Z",
        });
      }),
      http.get("http://localhost:8080/protected", ({ request }) => {
        protectedCount += 1;
        if (request.headers.get("Authorization") === "Bearer old-token") {
          return HttpResponse.json(
            { code: "unauthorized", message: "unauthorized" },
            { status: 401 },
          );
        }
        expect(request.headers.get("Authorization")).toBe(
          "Bearer rotated-token",
        );
        return HttpResponse.json({ ok: true });
      }),
    );

    await loginUser({ password: "password123", username: "favour" });
    await expect(
      authenticatedRequest<{ ok: boolean }>("/protected"),
    ).resolves.toEqual({ ok: true });
    expect(refreshCount).toBe(1);
    expect(protectedCount).toBe(2);
  });

  it("clears local credentials even when logout cannot reach the server", async () => {
    mockServer.use(
      http.post("http://localhost:8080/users/login", () =>
        HttpResponse.json({
          access_token: "login-token",
          access_token_expires_at: "2099-01-01T00:00:00Z",
          user,
        }),
      ),
      http.post("http://localhost:8080/sessions/logout", () =>
        HttpResponse.error(),
      ),
      http.post("http://localhost:8080/tokens/renew_access", () =>
        HttpResponse.json(
          { code: "unauthorized", message: "unauthorized" },
          { status: 401 },
        ),
      ),
    );

    await loginUser({ password: "password123", username: "favour" });
    await expect(logoutCurrentSession()).rejects.toMatchObject({
      code: "network_error",
    });
    await expect(renewAccessToken()).rejects.toMatchObject({
      code: "unauthorized",
    });
  });

  it("announces when a protected request remains unauthorized after refresh", async () => {
    const sessionExpired = vi.fn();
    window.addEventListener(sessionExpiredEvent, sessionExpired);
    mockServer.use(
      http.post("http://localhost:8080/users/login", () =>
        HttpResponse.json({
          access_token: "old-token",
          access_token_expires_at: "2099-01-01T00:00:00Z",
          user,
        }),
      ),
      http.post("http://localhost:8080/tokens/renew_access", () =>
        HttpResponse.json({
          access_token: "new-token",
          access_token_expires_at: "2099-01-01T00:00:00Z",
        }),
      ),
      http.get("http://localhost:8080/protected", () =>
        HttpResponse.json(
          { code: "unauthorized", message: "unauthorized" },
          { status: 401 },
        ),
      ),
    );

    await loginUser({ password: "password123", username: "favour" });
    await expect(authenticatedRequest("/protected")).rejects.toMatchObject({
      status: 401,
    });
    expect(sessionExpired).toHaveBeenCalledOnce();
    window.removeEventListener(sessionExpiredEvent, sessionExpired);
  });
});
