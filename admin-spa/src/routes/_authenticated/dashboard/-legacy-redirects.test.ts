import { describe, expect, it } from 'vitest'

import { Route as MetricsRoute } from './metrics'
import { Route as RetrievalLabRoute } from './retrieval-lab'

describe('legacy dashboard redirects', () => {
  it('keeps /dashboard/metrics pointed at AI usage', () => {
    expectBeforeLoadRedirect(
      MetricsRoute.options.beforeLoad,
      '/dashboard/ai-usage',
    )
  })

  it('returns retired retrieval lab links to the dashboard', () => {
    expectBeforeLoadRedirect(RetrievalLabRoute.options.beforeLoad, '/dashboard')
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
