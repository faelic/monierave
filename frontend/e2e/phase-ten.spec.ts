import { expect, test } from "@playwright/test";

test("desktop marketing renders the static pipeline composition", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "chromium");

  await page.setViewportSize({ height: 823, width: 1440 });
  await page.goto("/");

  const visual = page.locator("[data-pipeline-mode]");
  await expect(visual).toHaveAttribute("data-pipeline-mode", "webgl");
  await expect(visual.locator("canvas")).toBeVisible();
});

test("authentication remains usable while its visual loads independently", async ({
  page,
}) => {
  await page.setViewportSize({ height: 900, width: 1440 });
  await page.goto("/login");

  await expect(page.getByLabel("Username")).toBeEditable();
  await expect(page.locator("#password")).toBeEditable();
  await expect(page.getByRole("button", { name: "Sign in" })).toBeEnabled();
  await expect(page.locator("aside [data-wallet-mode]")).toHaveAttribute(
    "data-wallet-mode",
    "interactive",
  );
});

test("compact and reduced-motion layouts stay canvas-free", async ({
  page,
}) => {
  await page.setViewportSize({ height: 800, width: 390 });
  await page.goto("/");

  await expect(page.locator("[data-pipeline-mode]")).toHaveAttribute(
    "data-pipeline-mode",
    "fallback",
  );
  await expect(page.locator("canvas")).toHaveCount(0);

  await page.setViewportSize({ height: 900, width: 1440 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.reload();

  await expect(page.locator("[data-pipeline-mode]")).toHaveAttribute(
    "data-pipeline-mode",
    "fallback",
  );
  await expect(page.locator("canvas")).toHaveCount(0);
});

test("mobile signup places the branded fallback before the form", async ({
  page,
}) => {
  await page.setViewportSize({ height: 844, width: 390 });
  await page.goto("/signup");

  const fallback = page.getByRole("img", {
    name: "An open Monierave digital wallet holding three branded payment cards",
  });
  const username = page.getByLabel("Username");

  await expect(fallback).toBeVisible();
  await expect(username).toBeEditable();
  await expect(page.locator("canvas")).toHaveCount(0);

  const imagePrecedesInput = await fallback.evaluate(
    (image, input) =>
      Boolean(
        image.compareDocumentPosition(input as Node) &
        Node.DOCUMENT_POSITION_FOLLOWING,
      ),
    await username.elementHandle(),
  );
  expect(imagePrecedesInput).toBe(true);
});

test("serves each replaceable card texture", async ({ request }) => {
  for (const texture of [
    "/assets/3d/cards/monierave-primary.svg",
    "/assets/3d/cards/monierave-secondary.svg",
    "/assets/3d/cards/monierave-supporting.svg",
  ]) {
    const response = await request.get(texture);
    expect(response.ok()).toBe(true);
    expect(response.headers()["content-type"]).toContain("image/svg+xml");
  }
});
