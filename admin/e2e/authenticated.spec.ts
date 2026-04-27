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

  const session = await sessionResponse.json();
  expect(session.user).toMatchObject({
    email: adminEmail,
  });
  expect(["admin", "platform_admin"]).toContain(session.user.role);
}

async function completeOnboarding(page: Page) {
  const current = await page.request.get("/api/admin/onboarding");
  expect(current.ok()).toBeTruthy();

  const view = await current.json();
  if (view.onboarding) {
    return;
  }

  const response = await page.request.post("/api/admin/onboarding", {
    data: {
      school_name: view.tenant_name ?? "CI Test School",
      curriculum: {
        syllabus_id: "kssm-algebra",
        label: "KSSM Algebra",
      },
      first_class: {
        name: "CI Test Class",
        slug: "ci-test-class",
      },
      bot_setup: {
        preset: "guided-practice",
      },
    },
  });

  if (!response.ok()) {
    const body = await response.text();
    throw new Error(`Failed to complete onboarding. Status ${response.status()}. Body: ${body}`);
  }
}

async function loginAsConfiguredAdmin(page: Page) {
  await loginAsAdmin(page);
  await completeOnboarding(page);
}

test.describe("admin authenticated routes @backend", () => {
  test.describe.configure({ mode: "serial" });

  test.skip(
    !hasAuthSetup,
    "Set E2E_AUTH_ENABLED=true, E2E_ADMIN_EMAIL, and E2E_ADMIN_PASSWORD to run authenticated E2E tests.",
  );

  test("redirects authenticated users away from /login", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/login");
    await expect(page).toHaveURL(/\/setup\/onboard$/);
  });

  test("renders configured admin route surfaces", async ({ page }) => {
    await loginAsConfiguredAdmin(page);

    const routes = [
      { path: "/dashboard", url: /\/dashboard$/, heading: "Dashboard" },
      { path: "/dashboard/ai-usage", url: /\/dashboard\/ai-usage$/, heading: "AI usage" },
      { path: "/dashboard/classes", url: /\/dashboard\/classes$/, heading: "Class management" },
      { path: "/settings/users", url: /\/settings\/users$/, heading: "User and invite management" },
      { path: "/export", url: /\/export$/, heading: "Data export" },
    ];

    for (const route of routes) {
      await test.step(route.path, async () => {
        await page.goto(route.path);
        await expect(page).toHaveURL(route.url);
        await expect(page.getByRole("heading", { name: route.heading })).toBeVisible();
      });
    }
  });

  test("redirects legacy dashboard metrics route to AI usage", async ({ page }) => {
    await loginAsConfiguredAdmin(page);
    await page.goto("/dashboard/metrics");
    await expect(page).toHaveURL(/\/dashboard\/ai-usage$/);
    await expect(page.getByRole("heading", { name: "AI usage" })).toBeVisible();
  });

  test("redirects configured admins away from /login to dashboard", async ({ page }) => {
    await loginAsConfiguredAdmin(page);
    await page.goto("/login");
    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
  });
});
