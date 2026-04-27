import { expect, test } from "@playwright/test";

test.describe("admin public entry flows", () => {
  test("renders the login form fields", async ({ page }) => {
    await page.goto("/login");

    await expect(page.getByRole("main")).toBeVisible();
    await expect(page.locator('input[name="email"]')).toBeVisible();
    await expect(page.locator('input[name="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test("keeps the landing page visible and preserves next on the login link", async ({ page }) => {
    await page.goto("/?next=/dashboard");
    await expect(page).toHaveURL(/\/\?next=\/dashboard$/);
    await expect(page.getByRole("main")).toBeVisible();
    await expect(page.locator('a[href="/login?next=%2Fdashboard"]').first()).toHaveAttribute(
      "href",
      "/login?next=%2Fdashboard",
    );
  });

  test("public login route renders without protected navigation chrome", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByRole("button", { name: /theme/i })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Dashboard" })).toHaveCount(0);
  });

  test("invite acceptance submit button is disabled when token is missing", async ({ page }) => {
    await page.goto("/activate");
    await expect(page.getByRole("main")).toBeVisible();
    await expect(page.getByRole("button", { name: "Accept invite" })).toBeDisabled();
  });

  test("invite acceptance form hydrates and becomes submittable when token is present", async ({ page }) => {
    await page.goto("/activate?token=test-token");
    await page.getByLabel("Full name").fill("Teacher One");
    await page.getByLabel("Password").fill("strong-pass-1");
    await expect(page.getByRole("button", { name: "Accept invite" })).toBeEnabled();
  });
});

test.describe("admin protected route redirects", () => {
  const protectedPaths = [
    "/dashboard",
    "/dashboard/ai-usage",
    "/dashboard/classes",
    "/dashboard/metrics",
    "/students/test-student-id",
    "/parents/test-parent-id",
    "/settings/users",
    "/export",
  ];

  for (const path of protectedPaths) {
    test(`redirects unauthenticated access from ${path} to login`, async ({ page }) => {
      await page.goto(path);
      await expect(page).toHaveURL(new RegExp(`/login\\?next=${encodeURIComponent(path)}$`));
    });
  }
});
