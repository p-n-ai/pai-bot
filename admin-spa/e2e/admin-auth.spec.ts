import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

const adminEmail = process.env.E2E_ADMIN_EMAIL ?? ''
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? ''
const hasBackendAuth = Boolean(adminEmail && adminPassword)

const protectedRoutes = [
  { path: '/dashboard', heading: 'Today' },
  { path: '/dashboard/classes', heading: 'Classes' },
  { path: '/settings/users', heading: 'Staff access' },
  { path: '/export', heading: 'Data export' },
] as const

test.describe('admin public auth flows', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/auth/session', async (route) => {
      await route.fulfill({ status: 401, body: '' })
    })
    await page.route('**/api/auth/capabilities', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ google_login: false }),
      })
    })
  })

  test('renders the login form at the public root', async ({ page }) => {
    await page.goto('/')

    await expect(
      page.getByRole('heading', { name: 'Welcome back.' }),
    ).toBeVisible()
    await expect(page.getByLabel('Email')).toBeVisible()
    await expect(page.getByLabel('Password')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible()
  })

  test('redirects protected routes to the root with an encoded return path', async ({
    page,
  }) => {
    for (const { path } of protectedRoutes) {
      await page.goto(path)

      await expect
        .poll(() => {
          const url = new URL(page.url())
          return `${url.pathname}${url.search}`
        })
        .toBe(`/?next=${encodeURIComponent(path)}`)
      await expect(
        page.getByRole('heading', { name: 'Welcome back.' }),
      ).toBeVisible()
    }
  })
})

test.describe('admin authenticated routes @backend', () => {
  test.skip(
    !hasBackendAuth,
    'Set E2E_ADMIN_EMAIL and E2E_ADMIN_PASSWORD to run backend E2E tests.',
  )

  test('uses the backend session across representative admin routes', async ({
    page,
  }) => {
    await loginAsAdmin(page)

    await page.goto('/')
    await expect(page).toHaveURL(/\/dashboard$/)
    await expect(page.getByRole('heading', { name: 'Today' })).toBeVisible()

    for (const { path, heading } of protectedRoutes) {
      await test.step(`opens ${path}`, async () => {
        await page.goto(path)
        await expect(page).toHaveURL(new RegExp(`${escapeRegExp(path)}$`))
        await expect(page.getByRole('heading', { name: heading })).toBeVisible()
      })
    }
  })
})

async function loginAsAdmin(page: Page): Promise<void> {
  const loginResponse = await page.request.post('/api/auth/login', {
    data: {
      email: adminEmail,
      password: adminPassword,
    },
  })
  expect(
    loginResponse.ok(),
    `Backend login failed with status ${loginResponse.status()}`,
  ).toBeTruthy()

  const sessionResponse = await page.request.get('/api/auth/session')
  expect(
    sessionResponse.ok(),
    `Session lookup failed with status ${sessionResponse.status()}`,
  ).toBeTruthy()
  const session: unknown = await sessionResponse.json()
  expect(session).toEqual(
    expect.objectContaining({
      user: expect.objectContaining({
        role: expect.stringMatching(/^(admin|platform_admin)$/),
      }),
    }),
  )
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
