import { expect, test, type Page } from "@playwright/test";

const authEnabled = process.env.E2E_AUTH_ENABLED === "true";
const adminEmail = process.env.E2E_ADMIN_EMAIL ?? "";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "";
const hasAuthSetup = authEnabled && Boolean(adminEmail) && Boolean(adminPassword);

async function loginAsAdmin(page: Page) {
  const response = await page.request.post("/api/auth/login", {
    data: {
      email: adminEmail,
      password: adminPassword,
    },
  });

  if (!response.ok()) {
    const body = await response.text();
    throw new Error(
      `Failed to login test user (${adminEmail}). Status ${response.status()}. Body: ${body}`,
    );
  }

  const sessionResponse = await page.request.get("/api/auth/session");
  expect(sessionResponse.ok()).toBeTruthy();
}

test.describe("admin authenticated routes @backend", () => {
  test.skip(
    !hasAuthSetup,
    "Set E2E_AUTH_ENABLED=true, E2E_ADMIN_EMAIL, and E2E_ADMIN_PASSWORD to run authenticated E2E tests.",
  );

  test("redirects authenticated users away from /login", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/login");
    await expect(page).toHaveURL(/\/setup\/onboard$/);
  });

  test("renders /dashboard", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/dashboard");
    await expect(page).toHaveURL(/\/dashboard$/);
  });

  test("renders /dashboard/ai-usage", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/dashboard/ai-usage");
    await expect(page).toHaveURL(/\/dashboard\/ai-usage$/);
  });

  test("redirects /dashboard/metrics to /dashboard/ai-usage", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/dashboard/metrics");
    await expect(page).toHaveURL(/\/dashboard\/ai-usage$/);
  });

  test("renders /dashboard/classes", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/dashboard/classes");
    await expect(page).toHaveURL(/\/dashboard\/classes$/);
  });

  test("renders /students/:id route shell", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/students/test-student-id");
    await expect(page).toHaveURL(/\/students\/test-student-id$/);
  });

  test("renders /parents/:id route shell", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/parents/test-parent-id");
    await expect(page).toHaveURL(/\/parents\/test-parent-id$/);
  });

  test("renders /settings/users for admin-level roles", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings/users");
    await expect(page).toHaveURL(/\/settings\/users$/);
  });

  test("renders /export for admin-level roles", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/export");
    await expect(page).toHaveURL(/\/export$/);
  });
});
