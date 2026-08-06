import { expect, test } from "@playwright/test";

test("explains the supported product clearly", async ({ page }) => {
  await page.goto("/");

  await expect(
    page.getByRole("heading", {
      name: "Your money, without mystery.",
    }),
  ).toBeVisible();
  await expect(page.getByRole("main")).toBeVisible();
  await expect(
    page.getByText("Internal Monierave transfers currently carry no fee."),
  ).toBeVisible();
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

  await expect(page.locator(".marketing-reveal").first()).toHaveCSS(
    "animation-name",
    "none",
  );
});
