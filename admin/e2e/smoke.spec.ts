import { expect, test } from "@playwright/test";

test.describe("admin public entry flows", () => {
  test("renders the login hero and form fields", async ({ page }) => {
    await page.goto("/login");

    await expect(page.getByRole("heading", { name: "See who needs help before the exam." })).toBeVisible();
    await expect(page.getByLabel("Email")).toBeVisible();
    await expect(page.getByLabel("Password")).toBeVisible();
    await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();
  });

  test("redirects root path to login and preserves next query parameter", async ({ page }) => {
    await page.goto("/?next=/dashboard");
    await expect(page).toHaveURL(/\/login\?next=%2Fdashboard$/);
  });

  test("shows mapped auth error message on login when auth_error is present", async ({ page }) => {
    await page.goto("/login?auth_error=google_auth_failed");
    await expect(page.getByText("Google sign-in failed. Please try again.")).toBeVisible();
  });

  test("theme toggle is available on public routes and updates the html class", async ({ page }) => {
    await page.goto("/login");
    const themeToggle = page.getByRole("button", { name: /theme/i });
    await expect(themeToggle).toBeVisible();

    const html = page.locator("html");
    const before = (await html.getAttribute("class")) ?? "";

    await themeToggle.click();

    await expect
      .poll(async () => (await html.getAttribute("class")) ?? "")
      .not.toBe(before);
  });

  test("invite acceptance submit button is disabled when token is missing", async ({ page }) => {
    await page.goto("/activate");
    await expect(page.getByRole("heading", { name: "Accept your invite and set the password for this workspace." })).toBeVisible();
    await expect(page.getByRole("button", { name: "Accept invite" })).toBeDisabled();
  });

  test("invite acceptance submit button is enabled when token is present", async ({ page }) => {
    await page.goto("/activate?token=test-token");
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
