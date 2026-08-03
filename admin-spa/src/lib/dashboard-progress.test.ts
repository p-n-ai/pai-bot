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

    await expect(fetchDashboardProgress('class-1', fetcher)).resolves.toEqual({
      progress: {
        students: [],
        topic_ids: [],
      },
      source: 'live',
    })
    expect(fetcher).toHaveBeenCalledWith('/api/admin/classes/class-1/progress', {
      credentials: 'include',
      cache: 'no-store',
      headers: {},
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

    await expect(
      fetchDashboardProgress('all-students', fetcher),
    ).resolves.toEqual({
      progress: {
        students: [],
        topic_ids: [],
      },
      source: 'live',
    })
  })
})
