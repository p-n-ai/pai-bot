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

  test("renders authenticated routes", async ({ page }) => {
    await loginAsAdmin(page);

    await test.step("redirects authenticated users away from /login", async () => {
      await page.goto("/login");
      await expect(page).toHaveURL(/\/setup\/onboard$/);
    });

    const routes = [
      { path: "/dashboard", expected: /\/dashboard$/ },
      { path: "/dashboard/ai-usage", expected: /\/dashboard\/ai-usage$/ },
      { path: "/dashboard/metrics", expected: /\/dashboard\/ai-usage$/ },
      { path: "/dashboard/classes", expected: /\/dashboard\/classes$/ },
      {
        path: "/students/test-student-id",
        expected: /\/students\/test-student-id$/,
      },
      {
        path: "/parents/test-parent-id",
        expected: /\/parents\/test-parent-id$/,
      },
      { path: "/settings/users", expected: /\/settings\/users$/ },
      { path: "/export", expected: /\/export$/ },
    ] as const;

    for (const route of routes) {
      await test.step(`renders ${route.path}`, async () => {
        await page.goto(route.path);
        await expect(page).toHaveURL(route.expected);
      });
    }
  });
});
