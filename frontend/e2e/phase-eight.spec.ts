import { expect, test, type Page, type Route } from "@playwright/test";

const user = {
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
};

test("updates safe profile fields through the authenticated contract", async ({
  page,
}) => {
  let updateBody: unknown;
  await mockAuthenticatedAPI(page, async (route) => {
    if (
      new URL(route.request().url()).pathname === "/users/me" &&
      route.request().method() === "PATCH"
    ) {
      updateBody = await route.request().postDataJSON();
      return json(route, { ...user, full_name: "Favour A. Ututu" });
    }
    return false;
  });

  await page.goto("/app/profile");
  await expect(page.getByRole("heading", { name: "Profile" })).toBeVisible();
  await page.getByRole("link", { name: "Edit profile" }).click();
  await page.getByLabel("Full name").fill("Favour A. Ututu");
  await page.getByRole("button", { name: "Save profile" }).click();
  await expect(page.getByText("Profile updated.")).toBeVisible();
  expect(updateBody).toEqual({
    email: "favour@example.com",
    full_name: "Favour A. Ututu",
  });
});

test("signs out every session after consequence confirmation", async ({
  page,
}) => {
  let logoutAllCount = 0;
  let loggedOut = false;
  await mockAuthenticatedAPI(page, async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/sessions/logout-all") {
      logoutAllCount += 1;
      loggedOut = true;
      await route.fulfill({ headers: corsHeaders(), status: 204 });
      return;
    }
    if (path === "/tokens/renew_access" && loggedOut) {
      return json(
        route,
        { code: "unauthorized", message: "unauthorized" },
        401,
      );
    }
    return false;
  });

  await page.goto("/app/security");
  await page.getByRole("button", { name: "Review logout" }).click();
  await expect(page.getByText(/need to sign in again/i)).toBeVisible();
  await page.getByRole("button", { name: "Sign out everywhere" }).click();
  await expect(page).toHaveURL(/\/login(?:\?.*)?$/);
  expect(logoutAllCount).toBe(1);
});

async function mockAuthenticatedAPI(
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
        access_token: "profile-token",
        access_token_expires_at: "2099-01-01T00:00:00Z",
      });
    }
    if (path === "/users/me") return json(route, user);
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
    "Access-Control-Allow-Headers": "Authorization, Content-Type",
    "Access-Control-Allow-Methods": "GET, POST, PATCH, OPTIONS",
    "Access-Control-Allow-Origin": "http://127.0.0.1:3000",
  };
}
