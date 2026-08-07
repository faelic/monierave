import { expect, test, type Page, type Route } from "@playwright/test";

const verifiedUser = {
  account_status: "active",
  created_at: "2026-08-01T09:00:00Z",
  email: "favour@example.com",
  email_bounced_at: null,
  email_deliverability_status: "deliverable",
  email_verified_at: "2026-08-01T09:05:00Z",
  full_name: "Favour Ututu",
  password_changed_at: "2026-08-01T09:00:00Z",
  registration_expires_at: null,
  username: "favour22",
} as const;

const accounts = [
  {
    account_number: "4839201756",
    balance: 125050,
    closed_at: null,
    created_at: "2026-08-01T10:00:00Z",
    currency: "USD",
    id: "11111111-1111-4111-8111-111111111111",
    status: "active",
    updated_at: "2026-08-05T12:00:00Z",
  },
  {
    account_number: "9374028641",
    balance: 4200,
    closed_at: null,
    created_at: "2026-08-02T10:00:00Z",
    currency: "EUR",
    id: "22222222-2222-4222-8222-222222222222",
    status: "frozen",
    updated_at: "2026-08-05T12:00:00Z",
  },
] as const;

test("renders the verified dashboard without combining currencies", async ({
  page,
}) => {
  await mockDashboardAPI(page);
  await page.goto("/app");

  await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible();
  await expect(page.getByText("Current posted balance").first()).toBeVisible();
  await expect(page.getByText(/USD\s*1,250\.50/)).toBeVisible();
  await expect(page.getByText(/EUR\s*42\.00/)).toBeVisible();
  await expect(page.getByText("Acme Market")).toBeVisible();
  await expect(page.getByText(/Latest five transactions/)).toBeVisible();
  await expect(page.getByText(/available balance/i)).toHaveCount(0);
  await expect(page.getByText(/USD\s*1,292\.50/)).toHaveCount(0);

  await page.getByRole("button", { name: "Hide balances" }).click();
  await expect(page.getByText(/USD\s*1,250\.50/)).toHaveCount(0);
  await expect(page.getByText(/EUR\s*42\.00/)).toHaveCount(0);
  await expect(page.getByLabel("Current posted balance hidden")).toHaveCount(2);

  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});

test("redirects an anonymous dashboard request to safe sign in", async ({
  page,
}) => {
  await mockAPI(page, unauthorized);
  await page.goto("/app");

  await expect(page).toHaveURL(/\/login\?returnTo=%2Fapp$/, {
    timeout: 15_000,
  });
  await expect(
    page.getByRole("heading", { name: "Sign in securely." }),
  ).toBeVisible();
});

test("redirects a pending customer away from financial information", async ({
  page,
}) => {
  await mockAPI(page, async (route) => {
    const path = requestPath(route);
    if (path === "/tokens/renew_access") {
      return json(route, {
        access_token: "pending-token",
        access_token_expires_at: "2099-01-01T00:00:00Z",
      });
    }
    if (path === "/users/me") {
      return json(route, {
        ...verifiedUser,
        account_status: "pending",
        email_verified_at: null,
        registration_expires_at: "2026-08-08T09:00:00Z",
      });
    }
    if (path === "/users/me/email-status") {
      return json(route, {
        account_status: "pending",
        bounced_at: null,
        deliverability_status: "unknown",
        email: verifiedUser.email,
        registration_expires_at: "2026-08-08T09:00:00Z",
        verified_at: null,
      });
    }
    return unauthorized(route);
  });

  await page.goto("/app");
  await expect(page).toHaveURL(/\/verification-needed$/, { timeout: 15_000 });
  await expect(page.getByText(/USD\s*1,250\.50/)).toHaveCount(0);
  await expect(
    page.getByRole("heading", { name: "Verify your email." }),
  ).toBeVisible();
});

test("shows an explicit empty account state", async ({ page }) => {
  await mockDashboardAPI(page, []);
  await page.goto("/app");

  await expect(
    page.getByRole("heading", { name: "No financial accounts yet." }),
  ).toBeVisible();
  await expect(page.getByText(/USD\s*1,250\.50/)).toHaveCount(0);
  await expect(page.getByText(/EUR\s*42\.00/)).toHaveCount(0);
  await expect(page.getByText("Primary overview account")).toHaveCount(0);
});

async function mockDashboardAPI(
  page: Page,
  accountResponse: readonly (typeof accounts)[number][] = accounts,
) {
  await mockAPI(page, async (route) => {
    const path = requestPath(route);
    if (path === "/tokens/renew_access") {
      return json(route, {
        access_token: "dashboard-token",
        access_token_expires_at: "2099-01-01T00:00:00Z",
      });
    }
    if (path === "/users/me") {
      return json(route, verifiedUser);
    }
    if (path === "/accounts") {
      expect(new URL(route.request().url()).searchParams.get("page_size")).toBe(
        "100",
      );
      return json(route, accountResponse);
    }
    if (path === `/accounts/${accounts[0].id}/transactions`) {
      return json(route, {
        transactions: [
          {
            account_id: accounts[0].id,
            amount: 2500,
            balance_after: 125050,
            counterparty: "Acme Market",
            counterparty_type: "account",
            created_at: "2026-08-05T12:00:00Z",
            currency: "USD",
            direction: "outgoing",
            id: "33333333-3333-4333-8333-333333333333",
            narration: "Lunch",
            posted_at: "2026-08-05T12:00:01Z",
            reference: "TXN-ABC123",
            status: "posted",
            type: "internal_transfer",
          },
        ],
      });
    }
    return unauthorized(route);
  });
}

async function mockAPI(page: Page, handler: (route: Route) => Promise<void>) {
  await page.route("http://localhost:8080/**", async (route) => {
    if (route.request().method() === "OPTIONS") {
      await route.fulfill({ headers: corsHeaders(), status: 204 });
      return;
    }
    await handler(route);
  });
}

function requestPath(route: Route) {
  return new URL(route.request().url()).pathname;
}

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({
    headers: { ...corsHeaders(), "Content-Type": "application/json" },
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
