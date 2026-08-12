import { expect, test, type Page, type Route } from "@playwright/test";

const pendingUser = {
  account_status: "pending",
  created_at: "2026-08-01T09:00:00Z",
  email: "favour@example.com",
  email_bounced_at: null,
  email_deliverability_status: "unknown",
  email_verified_at: null,
  full_name: "Favour Ututu",
  password_changed_at: "2026-08-01T09:00:00Z",
  registration_expires_at: "2026-08-08T09:00:00Z",
  username: "favour22",
} as const;

test("creates a registration and explains the email confirmation step", async ({
  page,
}) => {
  await mockAPI(page, async (route) => {
    const request = route.request();
    if (request.method() === "POST" && requestURL(route) === "/users") {
      expect(await request.postDataJSON()).toEqual({
        email: "favour@example.com",
        full_name: "Favour Ututu",
        password: "clear-sky-2026",
        username: "favour22",
      });
      return json(route, pendingUser, 201);
    }
    return unauthorized(route);
  });

  await page.goto("/signup");
  await page.getByLabel("Full name").fill("Favour Ututu");
  await page.getByLabel("Username").fill("favour22");
  await page.getByLabel("Email address").fill("FAVOUR@example.com");
  await page.getByLabel("Password", { exact: true }).fill("clear-sky-2026");
  await page.getByRole("button", { name: "Create profile" }).click();

  await expect(page).toHaveURL(/\/signup\/check-email$/, { timeout: 15_000 });
  await expect(
    page.getByRole("heading", { name: "Check your email." }),
  ).toBeVisible();
  await expect(page.getByText("valid for 24 hours")).toBeVisible();
  await expect(page.getByText(/does not sign you in/i)).toBeVisible();
});

test("shows a safe compromised-password response without leaking details", async ({
  page,
}) => {
  await mockAPI(page, async (route) => {
    if (route.request().method() === "POST" && requestURL(route) === "/users") {
      return json(
        route,
        {
          code: "password_compromised",
          message: "password was found in a breach",
        },
        400,
      );
    }
    return unauthorized(route);
  });

  await page.goto("/signup");
  await page.getByLabel("Full name").fill("Favour Ututu");
  await page.getByLabel("Username").fill("favour22");
  await page.getByLabel("Email address").fill("favour@example.com");
  await page.getByLabel("Password", { exact: true }).fill("password123");
  await page.getByRole("button", { name: "Create profile" }).click();

  await expect(page.getByText(/known data breaches/i)).toBeVisible();
  await expect(page).toHaveURL(/\/signup$/);
});

test("routes a pending login into the limited application", async ({
  page,
}) => {
  await mockPendingAuthentication(page);

  await page.goto("/login?returnTo=https%3A%2F%2Fattacker.example%2Fcollect");
  await page.getByLabel("Username").fill("favour22");
  await page.getByLabel("Password", { exact: true }).fill("clear-sky-2026");
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page).toHaveURL(/\/app$/, { timeout: 15_000 });
  await expect(
    page.getByRole("heading", {
      name: "Verify your email to activate banking.",
    }),
  ).toBeVisible();
  await expect(page.getByText("Protected until verified")).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Profile" }).first(),
  ).toBeVisible();
  await expect(page).not.toHaveURL(/attacker/);
});

test("resends verification and replaces an undeliverable address", async ({
  page,
}) => {
  let email: string = pendingUser.email;
  let resendCount = 0;
  await mockPendingAuthentication(page, async (route) => {
    const path = requestURL(route);
    const method = route.request().method();

    if (path === "/users/me/resend-verification" && method === "POST") {
      resendCount += 1;
      return json(
        route,
        { job_id: "job-safe-id", message: "verification email queued" },
        202,
      );
    }
    if (path === "/users/me" && method === "PATCH") {
      const body = (await route.request().postDataJSON()) as {
        current_password: string;
        email: string;
      };
      expect(body).toEqual({
        current_password: "clear-sky-2026",
        email: "valid@example.com",
      });
      email = body.email;
      return json(route, {
        ...pendingUser,
        email,
        email_deliverability_status: "unknown",
        registration_expires_at: "2026-08-13T10:00:00Z",
      });
    }
    if (path === "/users/me/email-status" && method === "GET") {
      return json(route, {
        account_status: "pending",
        bounced_at: null,
        deliverability_status: "unknown",
        email,
        registration_expires_at: "2026-08-13T10:00:00Z",
        restricted_features: [
          "Create and manage financial accounts",
          "Send and receive transfers",
        ],
        verified_at: null,
      });
    }
    return false;
  });

  await signInPendingUser(page);
  await page.getByRole("button", { name: "Resend verification" }).click();
  await expect(page.getByText(/fresh verification email/i)).toBeVisible();
  expect(resendCount).toBe(1);

  await page.locator("summary").click();
  await page.getByLabel("Email address").fill("valid@example.com");
  await page.getByLabel("Current password").fill("clear-sky-2026");
  await page.getByRole("button", { name: "Update email" }).click();
  await expect(page).toHaveURL(/\/signup\/check-email$/, { timeout: 15_000 });
  await expect(
    page.getByRole("heading", { name: "Check your email." }),
  ).toBeVisible();
  await expect(
    page.getByText("valid@example.com", { exact: true }),
  ).toHaveCount(0);
});

test("auth screens fit the viewport without horizontal overflow", async ({
  page,
}) => {
  await mockAPI(page, unauthorized);
  await page.goto("/login");

  await expect(
    page.getByRole("heading", { name: "Welcome back" }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});

test("short landscape signup keeps the complete form in one viewport", async ({
  page,
}) => {
  await page.setViewportSize({ height: 576, width: 1024 });
  await mockAPI(page, unauthorized);
  await page.goto("/signup");

  await expect(page.getByText("At least 8 characters.")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Create profile" }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollHeight <= window.innerHeight,
    ),
  ).toBe(true);
});

async function mockPendingAuthentication(
  page: Page,
  extension?: (route: Route) => Promise<boolean | void>,
) {
  let signedIn = false;
  await mockAPI(page, async (route) => {
    if (extension) {
      const handled = await extension(route);
      if (handled !== false) {
        return;
      }
    }

    const path = requestURL(route);
    const method = route.request().method();
    if (path === "/users/login" && method === "POST") {
      signedIn = true;
      return json(route, {
        access_token: "pending-access-token",
        access_token_expires_at: "2099-01-01T00:00:00Z",
        session_id: "session-not-rendered",
        user: pendingUser,
      });
    }
    if (signedIn && path === "/tokens/renew_access" && method === "POST") {
      return json(route, {
        access_token: "renewed-pending-access-token",
        access_token_expires_at: "2099-01-01T00:00:00Z",
      });
    }
    if (signedIn && path === "/users/me" && method === "GET") {
      return json(route, pendingUser);
    }
    if (path === "/users/me/email-status" && method === "GET") {
      return json(route, {
        account_status: "pending",
        bounced_at: null,
        deliverability_status: "unknown",
        email: pendingUser.email,
        latest_job: {
          accepted_at: "2026-08-01T09:01:00Z",
          bounce_subtype: null,
          bounce_type: null,
          bounced_at: null,
          delivered_at: null,
          delivery_status: "accepted",
          id: "email-job-id",
          provider_message_id: null,
          worker_status: "sent",
        },
        registration_expires_at: pendingUser.registration_expires_at,
        restricted_features: [
          "Create and manage financial accounts",
          "Send and receive transfers",
        ],
        verified_at: null,
      });
    }
    return unauthorized(route);
  });
}

async function signInPendingUser(page: Page) {
  await page.goto("/login");
  await page.getByLabel("Username").fill("favour22");
  await page.getByLabel("Password", { exact: true }).fill("clear-sky-2026");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(/\/app$/, { timeout: 15_000 });
  await page.getByRole("link", { name: "Continue verification" }).click();
  await expect(page).toHaveURL(/\/verification-needed$/, { timeout: 15_000 });
  await expect(
    page.getByText(pendingUser.email, { exact: true }),
  ).toBeVisible();
}

async function mockAPI(page: Page, handler: (route: Route) => Promise<void>) {
  await page.route("http://localhost:8080/**", async (route) => {
    if (route.request().method() === "OPTIONS") {
      await route.fulfill({
        headers: corsHeaders(),
        status: 204,
      });
      return;
    }
    await handler(route);
  });
}

function requestURL(route: Route) {
  return new URL(route.request().url()).pathname;
}

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({
    headers: {
      ...corsHeaders(),
      "Content-Type": "application/json",
    },
    json: body,
    status,
  });
}

async function unauthorized(route: Route) {
  await json(route, { code: "unauthorized", message: "unauthorized" }, 401);
}

function corsHeaders() {
  return {
    "Access-Control-Allow-Credentials": "true",
    "Access-Control-Allow-Headers": "Authorization, Content-Type",
    "Access-Control-Allow-Methods": "GET, POST, PATCH, OPTIONS",
    "Access-Control-Allow-Origin": "http://127.0.0.1:3000",
  };
}
