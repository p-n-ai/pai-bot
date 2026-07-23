import { expect, test } from '@playwright/test'

const validURL = process.env.FOCUSED_PAGE_VALID_URL ?? ''
const wrongTokenURL = process.env.FOCUSED_PAGE_WRONG_TOKEN_URL ?? ''
const expiredURL = process.env.FOCUSED_PAGE_EXPIRED_URL ?? ''
const revokedURL = process.env.FOCUSED_PAGE_REVOKED_URL ?? ''

test.skip(
  !validURL || !wrongTokenURL || !expiredURL || !revokedURL,
  'focused-page Go test server is required',
)

test('redeems a private focused page after removing the capability fragment', async ({
  page,
}) => {
  let urlDuringRedemption = ''
  page.on('request', (request) => {
    if (request.method() === 'POST' && request.url().includes('/a/')) {
      urlDuringRedemption = page.url()
    }
  })

  const response = await page.goto(validURL)

  await expect(
    page.getByRole('heading', { name: 'A message for Aina' }),
  ).toBeVisible()
  await expect(page.getByText('Private goal report')).toBeVisible()
  await expect(
    page.getByRole('link', { name: 'Continue with P&AI' }),
  ).toBeVisible()
  await expect(page).not.toHaveURL(/#/)
  expect(urlDuringRedemption).not.toContain('#')
  expect(response?.headers()['cache-control']).toContain('no-store')
  expect(response?.headers()['content-security-policy']).toContain(
    "default-src 'none'",
  )
  expect(response?.headers()['referrer-policy']).toBe('no-referrer')
})

test('shows safe browser errors without revealing private content', async ({
  page,
}) => {
  const cases = [
    [wrongTokenURL, 'This page is unavailable.'],
    [expiredURL, 'This page has expired.'],
    [revokedURL, 'This page is no longer available.'],
    [validURL.replace(/#.*$/, ''), 'This page is unavailable.'],
  ] as const

  for (const [url, message] of cases) {
    await page.goto(url)
    await expect(page.getByRole('heading', { name: message })).toBeVisible()
    await expect(page.getByText('Private goal report')).toHaveCount(0)
    await expect(page).not.toHaveURL(/#/)
  }
})

test('fits the private message and action on a mobile viewport', async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(validURL)

  await expect(
    page.getByRole('heading', { name: 'A message for Aina' }),
  ).toBeVisible()
  await expect(
    page.getByRole('link', { name: 'Continue with P&AI' }),
  ).toBeVisible()
  const horizontalOverflow = await page.evaluate(
    () => document.documentElement.scrollWidth - innerWidth,
  )
  expect(horizontalOverflow).toBe(0)
})
