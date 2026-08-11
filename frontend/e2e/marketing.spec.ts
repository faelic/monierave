import { expect, test, type Page } from "@playwright/test";

const activeUser = {
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

test("offers the dashboard instead of sign in after restoring a session", async ({
  page,
}) => {
  await page.route("http://localhost:8080/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (route.request().method() === "OPTIONS") {
      await route.fulfill({ status: 204 });
      return;
    }
    if (path === "/tokens/renew_access") {
      await route.fulfill({
        contentType: "application/json",
        json: {
          access_token: "restored-access-token",
          access_token_expires_at: "2099-01-01T00:00:00Z",
        },
      });
      return;
    }
    if (path === "/users/me") {
      await route.fulfill({
        contentType: "application/json",
        json: activeUser,
      });
      return;
    }
    await route.fulfill({
      contentType: "application/json",
      json: { code: "not_found", message: "not found" },
      status: 404,
    });
  });

  await page.goto("/");

  const navigation = page.getByRole("banner");
  await expect(
    navigation.getByRole("link", { name: "Open dashboard" }),
  ).toBeVisible();
  await expect(navigation.getByRole("link", { name: "Sign in" })).toHaveCount(
    0,
  );
});

test("explains the supported product clearly", async ({ page }) => {
  await mockAnonymousSession(page);
  await page.goto("/");

  await expect(
    page.getByRole("heading", {
      name: "Your money, without mystery.",
    }),
  ).toBeVisible();
  await expect(page.getByRole("main")).toBeVisible();
  await expect(page.getByText("Zero-fee internal")).toBeVisible();
  await expect(
    page.getByRole("heading", {
      name: "See what changes when money moves.",
    }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});

test("keeps a dark canvas behind section entrance transitions", async ({
  page,
}) => {
  await mockAnonymousSession(page);
  await page.goto("/");
  const shells = page.locator("[data-section-reveal-shell]");

  await expect(shells).toHaveCount(5);
  const colors = await shells.evaluateAll((elements) =>
    elements.map((element) => getComputedStyle(element).backgroundColor),
  );
  expect(colors).not.toContain("rgba(0, 0, 0, 0)");
  expect(colors).not.toContain("rgb(247, 247, 245)");
  await expect(page.getByRole("main")).toHaveCSS(
    "background-color",
    "rgb(5, 6, 8)",
  );
});

test("offers a focused final conversion path", async ({ page }) => {
  await mockAnonymousSession(page);
  await page.goto("/");
  await expect(
    page.getByRole("heading", {
      name: "Ready to get started?",
    }),
  ).toBeVisible();
  await page
    .locator('[aria-labelledby="closing-cta-title"]')
    .getByRole("link", { name: "Get started" })
    .click();
  await expect(page).toHaveURL(/\/signup$/);
});

test("orders navigation by the sections on the page", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "chromium");
  await mockAnonymousSession(page);
  await page.goto("/");
  const navigation = page.getByRole("navigation", {
    name: "Primary navigation",
  });

  await expect(navigation.getByRole("link")).toHaveText([
    "How it works",
    "Workspace",
    "Money movement",
  ]);
  await expect(
    navigation.getByRole("link", { name: "How it works" }),
  ).toHaveAttribute("href", "/#how-it-works");
  await expect(
    navigation.getByRole("link", { name: "Workspace" }),
  ).toHaveAttribute("href", "/#product-architecture");
  await expect(
    navigation.getByRole("link", { name: "Money movement" }),
  ).toHaveAttribute("href", "/#money-movement");

  await navigation.getByRole("link", { name: "Money movement" }).click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page.locator("#money-movement")).toBeInViewport();
  await expect(
    navigation.getByRole("link", { name: "Money movement" }),
  ).toHaveAttribute("aria-current", "location");
});

test("mobile navigation opens, closes, and follows routes", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name === "chromium");

  await mockAnonymousSession(page);
  await page.goto("/");
  await page.getByRole("button", { name: "Open navigation" }).click();

  const dialog = page.getByRole("dialog", { name: "Navigation" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("link", { name: "How it works" }).click();

  await expect(page).toHaveURL(/\/$/);
  await expect(page.locator("#how-it-works")).toBeInViewport();
  await expect(dialog).not.toBeVisible();
});

test("removes nonessential entrance motion when requested", async ({
  page,
}) => {
  await mockAnonymousSession(page);
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/");

  await expect(page.locator("[data-rotating-globe-stage]")).toHaveAttribute(
    "data-rotating-globe-motion",
    "static",
  );
});

test("matches the canonical desktop hero structure", async ({ page }) => {
  await page.setViewportSize({ height: 900, width: 1440 });
  await mockAnonymousSession(page);
  await page.goto("/");

  const heading = page.getByRole("heading", {
    name: "Your money, without mystery.",
  });
  const box = await heading.boundingBox();

  expect(box).not.toBeNull();
  expect(box?.x).toBeGreaterThanOrEqual(40);
  expect(box?.x).toBeLessThanOrEqual(80);
  expect(box?.y).toBeGreaterThanOrEqual(220);
  expect(box?.y).toBeLessThanOrEqual(300);
  await expect(
    page.locator(".landing-hero").getByRole("link", { name: "Get started" }),
  ).toBeVisible();
  await expect(page.getByTestId("celestial-money-globe")).toBeVisible();
  const globeBox = await page
    .getByTestId("celestial-money-globe")
    .boundingBox();
  expect(globeBox?.width ?? 0).toBeGreaterThanOrEqual(300);
  expect(globeBox?.width ?? 0).toBeLessThanOrEqual(420);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});

async function mockAnonymousSession(page: Page) {
  await page.route("http://localhost:8080/**", async (route) => {
    if (route.request().method() === "OPTIONS") {
      await route.fulfill({ status: 204 });
      return;
    }
    await route.fulfill({
      contentType: "application/json",
      json: { code: "unauthorized", message: "unauthorized" },
      status: 401,
    });
  });
}
