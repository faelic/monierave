import { expect, test } from "@playwright/test";

test("explains the supported product clearly", async ({ page }) => {
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
      name: "Financial clarity has a vocabulary.",
    }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});

test("navigates to the public security explanation", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("link", { name: "Our security approach" }).click();

  await expect(page).toHaveURL(/\/security$/);
  await expect(
    page.getByRole("heading", {
      name: "Protection, explained plainly.",
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", {
      name: "Clear boundaries are part of security.",
    }),
  ).toBeVisible();
});

test("mobile navigation opens, closes, and follows routes", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name === "chromium");

  await page.goto("/");
  await page.getByRole("button", { name: "Open navigation" }).click();

  const dialog = page.getByRole("dialog", { name: "Navigation" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("link", { name: "Security" }).click();

  await expect(page).toHaveURL(/\/security$/);
  await expect(dialog).not.toBeVisible();
});

test("removes nonessential entrance motion when requested", async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/");

  await expect(page.locator("[data-pipeline-mode]")).toHaveAttribute(
    "data-pipeline-mode",
    "fallback",
  );
  await expect(page.locator(".landing-hero canvas")).toHaveCount(0);
});

test("matches the canonical desktop hero structure", async ({ page }) => {
  await page.setViewportSize({ height: 823, width: 1440 });
  await page.goto("/");

  const heading = page.getByRole("heading", {
    name: "Your money, without mystery.",
  });
  const box = await heading.boundingBox();

  expect(box).not.toBeNull();
  expect(box?.x).toBeGreaterThanOrEqual(65);
  expect(box?.x).toBeLessThanOrEqual(90);
  expect(box?.y).toBeGreaterThanOrEqual(145);
  expect(box?.y).toBeLessThanOrEqual(220);
  await expect(heading.locator("span")).toHaveCount(3);
  await expect(page.getByRole("link", { name: "Get started" })).toBeVisible();
  await expect(
    page.getByRole("img", {
      name: "A metallic network of Monierave payment rails",
    }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});
