import { getClassProgress } from './admin-api'
import type { ClassProgress } from './dashboard-types'

export type DashboardProgressSource = 'live'

export interface DashboardProgressResult {
  progress: ClassProgress
  source: DashboardProgressSource
}

export async function fetchDashboardProgress(
  fetcher: typeof fetch = fetch,
): Promise<DashboardProgressResult> {
  return {
    progress: await getClassProgress('all-students', fetcher),
    source: 'live',
  }
}
