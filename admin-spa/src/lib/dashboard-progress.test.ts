import { describe, expect, it, vi } from 'vitest'

import { fetchDashboardProgress } from './dashboard-progress'

describe('dashboard progress', () => {
  it('wraps live class progress with a live source marker', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          students: [],
          topic_ids: [],
        }),
    })

    await expect(fetchDashboardProgress(fetcher)).resolves.toEqual({
      progress: {
        students: [],
        topic_ids: [],
      },
      source: 'live',
    })
  })

  it('normalizes empty backend progress slices', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          students: null,
          topic_ids: null,
        }),
    })

    await expect(fetchDashboardProgress(fetcher)).resolves.toEqual({
      progress: {
        students: [],
        topic_ids: [],
      },
      source: 'live',
    })
  })
})
