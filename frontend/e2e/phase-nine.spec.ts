import { expect, test } from "@playwright/test";

test("keeps the 3D experiment isolated and free of financial data", async ({
  page,
}) => {
  await page.goto("/lab/vault");

  await expect(
    page.getByRole("heading", {
      level: 1,
      name: "Security, expressed without spectacle.",
    }),
  ).toBeVisible();
  await expect(page.getByText("No financial truth")).toBeVisible();
  await expect(page.getByText(/current posted balance/i)).toHaveCount(0);
  await expect(page.locator("main")).not.toContainText(/\d{10}/);
});

test("uses the intentional still fallback with reduced motion", async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/lab/vault");

  await expect(
    page.getByText(
      "The still version is active because reduced motion is enabled.",
    ),
  ).toBeVisible();
  await expect(page.getByTestId("vault-static-fallback")).toBeVisible();
  await expect(page.locator("canvas")).toHaveCount(0);
});

test("the laboratory fits without horizontal overflow", async ({ page }) => {
  await page.goto("/lab/vault");

  const dimensions = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth,
  }));

  expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
  await expect(
    page.getByRole("link", { name: "Exit laboratory" }),
  ).toHaveAttribute("href", "/");
});
