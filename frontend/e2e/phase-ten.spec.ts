import { expect, test } from "@playwright/test";

test("desktop marketing can pause the optional 3D visual", async ({ page }) => {
  await page.setViewportSize({ height: 900, width: 1440 });
  await page.goto("/");

  const visual = page.locator("[data-vault-mode]").first();
  await expect(visual).toHaveAttribute("data-vault-mode", "interactive");
  await expect(visual.locator("canvas")).toBeVisible();

  await page.getByRole("button", { name: "Pause 3D preview" }).click();
  await expect(visual).toHaveAttribute("data-vault-mode", "static");
  await expect(visual.locator("canvas")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Use 3D preview" }),
  ).toBeVisible();
});

test("authentication remains usable while its visual loads independently", async ({
  page,
}) => {
  await page.setViewportSize({ height: 900, width: 1440 });
  await page.goto("/login");

  await expect(page.getByLabel("Username")).toBeEditable();
  await expect(page.locator("#password")).toBeEditable();
  await expect(page.getByRole("button", { name: "Sign in" })).toBeEnabled();
  await expect(page.locator("[data-vault-mode]")).toHaveAttribute(
    "data-vault-mode",
    "interactive",
  );
});

test("compact and reduced-motion layouts stay canvas-free", async ({
  page,
}) => {
  await page.setViewportSize({ height: 800, width: 390 });
  await page.goto("/");

  await expect(page.locator("[data-vault-mode]").first()).toHaveAttribute(
    "data-vault-mode",
    "static",
  );
  await expect(page.locator("canvas")).toHaveCount(0);

  await page.setViewportSize({ height: 900, width: 1440 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.reload();

  await expect(page.locator("[data-vault-mode]").first()).toHaveAttribute(
    "data-vault-mode",
    "static",
  );
  await expect(page.locator("canvas")).toHaveCount(0);
});
