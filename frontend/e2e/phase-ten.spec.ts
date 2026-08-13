import { expect, test, type Page } from "@playwright/test";

async function mockAnonymousSession(page: Page) {
  await page.route("http://localhost:8080/**", async (route) => {
    if (route.request().method() === "OPTIONS") {
      await route.fulfill({ status: 204 });
      return;
    }

    await route.fulfill({
      contentType: "application/json",
      json: {
        code: "unauthorized",
        message: "unauthorized",
      },
      status: 401,
    });
  });
}

test("desktop marketing renders the rotating globe composition", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "chromium");

  await page.setViewportSize({ height: 900, width: 1440 });
  await page.goto("/");

  const visual = page.locator("[data-rotating-globe-stage]");
  await expect(visual).toHaveAttribute("data-rotating-globe-motion", "active");
  await expect(page.getByTestId("celestial-money-globe")).toBeVisible();
  await expect(visual.locator("canvas")).toHaveCount(0);
});

test("authentication remains usable over its static visual atmosphere", async ({
  page,
}) => {
  await mockAnonymousSession(page);
  await page.setViewportSize({ height: 900, width: 1440 });
  await page.goto("/login");

  await expect(page.getByLabel("Username")).toBeEditable();
  await expect(page.locator("#password")).toBeEditable();
  await expect(page.getByRole("button", { name: "Sign in" })).toBeEnabled();
  await expect(page.locator("[data-auth-atmosphere]")).toBeVisible();
  await expect(page.locator("canvas")).toHaveCount(0);
});

test("compact and reduced-motion layouts preserve the globe concept", async ({
  page,
}) => {
  await page.setViewportSize({ height: 800, width: 390 });
  await page.goto("/");

  await expect(page.locator("[data-rotating-globe-stage]")).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);

  await page.setViewportSize({ height: 900, width: 1440 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.reload();

  await expect(page.locator("[data-rotating-globe-stage]")).toHaveAttribute(
    "data-rotating-globe-motion",
    "static",
  );
});

test("mobile signup keeps its atmosphere behind the usable form", async ({
  page,
}) => {
  await page.setViewportSize({ height: 844, width: 390 });
  await page.goto("/signup");

  const username = page.getByLabel("Username");

  await expect(page.locator("[data-auth-atmosphere]")).toBeVisible();
  await expect(username).toBeEditable();
  await expect(page.locator("canvas")).toHaveCount(0);
});
