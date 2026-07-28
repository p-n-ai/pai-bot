import { defineConfig, devices } from '@playwright/test'

const isCI = Boolean(process.env.CI)
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://127.0.0.1:5173'
const backendEnabled = process.env.E2E_BACKEND_ENABLED === 'true'
const useSystemChrome = process.env.PLAYWRIGHT_USE_SYSTEM_CHROME === 'true'

export default defineConfig({
  testDir: './e2e',
  testMatch: 'admin-auth.spec.ts',
  fullyParallel: true,
  forbidOnly: isCI,
  retries: isCI ? 2 : 0,
  workers: isCI ? 1 : undefined,
  reporter: isCI ? [['github'], ['html', { open: 'never' }]] : [['list']],
  outputDir: 'test-results/admin-e2e',
  use: {
    baseURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  grepInvert: backendEnabled ? undefined : /@backend/,
  webServer: process.env.PLAYWRIGHT_BASE_URL
    ? undefined
    : {
        command: 'pnpm dev --host 127.0.0.1 --port 5173',
        url: baseURL,
        reuseExistingServer: !isCI,
        timeout: 120_000,
      },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        ...(useSystemChrome ? { channel: 'chrome' as const } : {}),
      },
    },
  ],
})
