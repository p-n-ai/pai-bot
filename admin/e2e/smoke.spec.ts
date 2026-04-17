import { expect, test } from "@playwright/test";

test.describe("admin login smoke", () => {
  test("renders the login hero and form fields", async ({ page }) => {
    await page.goto("/login");

    await expect(page.getByRole("heading", { name: "See who needs help before the exam." })).toBeVisible();
    await expect(page.getByLabel("Email")).toBeVisible();
    await expect(page.getByLabel("Password")).toBeVisible();
    await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();
  });
});
