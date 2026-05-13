import { describe, expect, it } from 'vitest'

import { Route as MetricsRoute } from './metrics'

describe('legacy dashboard redirects', () => {
  it('keeps /dashboard/metrics pointed at AI usage', () => {
    expectBeforeLoadRedirect(
      MetricsRoute.options.beforeLoad,
      '/dashboard/ai-usage',
    )
  })
})

function expectBeforeLoadRedirect(
  beforeLoad: ((context: never) => unknown) | undefined,
  to: string,
) {
  expect(beforeLoad).toBeDefined()
  expect(() => beforeLoad?.(undefined as never)).toThrow(
    expect.objectContaining({
      options: expect.objectContaining({ to }),
    }),
  )
}
