import { expect, test, type Page, type Route } from "@playwright/test";

const account = {
  account_number: "4839201756",
  balance: 125050,
  closed_at: null,
  created_at: "2026-08-01T10:00:00Z",
  currency: "USD",
  id: "11111111-1111-4111-8111-111111111111",
  status: "active",
  updated_at: "2026-08-05T12:00:00Z",
};

test("resolves, reviews, and posts an idempotent transfer", async ({
  page,
}) => {
  let transferKey: string | null = null;
  await mockBankingAPI(page, async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/accounts/resolve") {
      return json(route, {
        account_name: "A**** M*****",
        account_number: "******8641",
        can_receive: true,
        currency: "USD",
      });
    }
    if (path === "/transfers") {
      transferKey = route.request().headers()["idempotency-key"] ?? null;
      expect(await route.request().postDataJSON()).toMatchObject({
        amount: 1050,
        currency: "USD",
        from_account_id: account.id,
        to_account_number: "9374028641",
      });
      return json(
        route,
        {
          fee: 0,
          from_account: { ...account, balance: 124000 },
          to_account: { account_number: "******8641", currency: "USD" },
          transaction: {
            amount: 1050,
            created_at: "2026-08-06T10:00:00Z",
            currency: "USD",
            id: "33333333-3333-4333-8333-333333333333",
            narration: "Lunch",
            posted_at: "2026-08-06T10:00:01Z",
            reference: "TXN-POSTED-1",
            status: "posted",
            type: "internal_transfer",
          },
        },
        201,
      );
    }
    return false;
  });

  await page.goto("/app/transfers/new");
  await page.getByLabel("Recipient account number").fill("9374028641");
  await page.getByRole("button", { name: "Search" }).click();
  await expect(page.getByText("A**** M*****").first()).toBeVisible();
  await expect(page.getByText(/\*{6}8641/).first()).toBeVisible();
  await page.getByLabel("Amount (USD)").fill("10.50");
  await page.getByLabel("Narration").fill("Lunch");
  await page.getByRole("button", { name: "Continue to review" }).click();
  await expect(
    page.getByRole("heading", { name: "Check every detail." }),
  ).toBeVisible();
  await expect(page.getByText(/USD\s*10\.50/).first()).toBeVisible();
  await expect(page.getByText(/Fee/)).toBeVisible();
  await page.getByRole("button", { name: /Send USD\s*10\.50/ }).click();
  await expect(
    page.getByRole("heading", { name: "Transfer posted." }),
  ).toBeVisible();
  await expect(page.getByText("A**** M*****")).toBeVisible();
  await expect(page.getByText(/\*{6}8641/)).toBeVisible();
  expect(transferKey).toMatch(/^[0-9a-f-]{36}$/);
});

test("creates an account without accepting client-generated identity fields", async ({
  page,
}) => {
  let createBody: unknown;
  await mockBankingAPI(page, async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/accounts" && route.request().method() === "POST") {
      createBody = await route.request().postDataJSON();
      return json(route, { ...account, balance: 0 }, 201);
    }
    if (path === `/accounts/${account.id}`) {
      return json(route, { ...account, balance: 0 });
    }
    if (path.endsWith("/transactions")) {
      return json(route, { transactions: [] });
    }
    return false;
  });

  await page.goto("/app/accounts/new");
  await page.getByText("EUR", { exact: true }).click();
  await page.getByRole("button", { name: "Open EUR account" }).click();
  await expect(page).toHaveURL(new RegExp(`/app/accounts/${account.id}$`));
  expect(createBody).toEqual({ currency: "EUR" });
});

test("prevents opening a duplicate currency account and links to the existing account", async ({
  page,
}) => {
  let createRequests = 0;
  await mockBankingAPI(page, async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/accounts" && route.request().method() === "POST") {
      createRequests += 1;
    }
    return false;
  });

  await page.goto("/app/accounts/new");
  await expect(page.getByText("Already open")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Already have a USD account" }),
  ).toBeDisabled();
  await page.getByRole("link", { name: "View USD account" }).click();
  await expect(page).toHaveURL(new RegExp(`/app/accounts/${account.id}$`));
  expect(createRequests).toBe(0);
});

async function mockBankingAPI(
  page: Page,
  extension: (route: Route) => Promise<boolean | void>,
) {
  await page.route("http://localhost:8080/**", async (route) => {
    if (route.request().method() === "OPTIONS") {
      await route.fulfill({ headers: corsHeaders(), status: 204 });
      return;
    }
    const handled = await extension(route);
    if (handled !== false) return;
    const path = new URL(route.request().url()).pathname;
    if (path === "/tokens/renew_access") {
      return json(route, {
        access_token: "banking-token",
        access_token_expires_at: "2099-01-01T00:00:00Z",
      });
    }
    if (path === "/users/me") {
      return json(route, {
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
      });
    }
    if (path === "/accounts") return json(route, [account]);
    return json(route, { code: "not_found", message: "not found" }, 404);
  });
}

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({
    body: JSON.stringify(body),
    contentType: "application/json",
    headers: corsHeaders(),
    status,
  });
}

function corsHeaders() {
  return {
    "Access-Control-Allow-Credentials": "true",
    "Access-Control-Allow-Headers":
      "Authorization, Content-Type, Idempotency-Key",
    "Access-Control-Allow-Methods": "GET, POST, PATCH, DELETE, OPTIONS",
    "Access-Control-Allow-Origin": "http://127.0.0.1:3000",
  };
}
