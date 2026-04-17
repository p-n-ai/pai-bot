import { expect, test, type Page } from "@playwright/test";

const adminEmail = process.env.E2E_ADMIN_EMAIL ?? "platform-admin@example.com";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "demo-password";

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

test.describe("admin authenticated routes", () => {
  test("redirects authenticated users away from /login", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/login");
    await expect(page).toHaveURL(/\/dashboard$/);
  });

  test("renders /dashboard", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/dashboard");
    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
    await expect(page.getByText("Mastery heatmap")).toBeVisible();
  });

  test("renders /dashboard/ai-usage", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/dashboard/ai-usage");
    await expect(page).toHaveURL(/\/dashboard\/ai-usage$/);
    await expect(page.getByRole("heading", { name: "Budget and provider usage" })).toBeVisible();
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
    await expect(page.getByRole("heading", { name: "Class management" })).toBeVisible();
  });

  test("renders /students/:id route shell", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/students/test-student-id");
    await expect(page).toHaveURL(/\/students\/test-student-id$/);
    await expect(page.getByText("Student detail")).toBeVisible();
  });

  test("renders /parents/:id route shell", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/parents/test-parent-id");
    await expect(page).toHaveURL(/\/parents\/test-parent-id$/);
    await expect(page.getByText("Parent support summary")).toBeVisible();
  });

  test("renders /settings/users for admin-level roles", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings/users");
    await expect(page).toHaveURL(/\/settings\/users$/);
    await expect(page.getByRole("heading", { name: "User and invite management" })).toBeVisible();
  });

  test("renders /export for admin-level roles", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/export");
    await expect(page).toHaveURL(/\/export$/);
    await expect(page.getByRole("heading", { name: "Data export" })).toBeVisible();
  });
});
