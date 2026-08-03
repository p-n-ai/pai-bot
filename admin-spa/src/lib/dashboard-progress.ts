import { getClassProgress } from './admin-api'
import type { ClassProgress } from './dashboard-types'

export type DashboardProgressSource = 'live'

export interface DashboardProgressResult {
  progress: ClassProgress
  source: DashboardProgressSource
}

export async function fetchDashboardProgress(
  classID: string,
  fetcher: typeof fetch = fetch,
): Promise<DashboardProgressResult> {
  return {
    progress: await getClassProgress(classID, fetcher),
    source: 'live',
  }
}
