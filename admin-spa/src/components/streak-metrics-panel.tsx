import { AdminHighlightPanel } from '@/components/admin-highlight-panel'
import { Metric } from '@/components/metric'
import { formatDayCount } from '@/lib/date-format'

export function StreakMetricsPanel({
  current,
  longest,
  totalXP,
}: {
  current: number
  longest: number
  totalXP: number
}) {
  return (
    <AdminHighlightPanel className='sm:grid-cols-3 lg:grid-cols-1'>
      <Metric label='Current streak' value={formatDayCount(current)} />
      <Metric label='Longest streak' value={formatDayCount(longest)} />
      <Metric label='Total XP' value={String(totalXP)} />
    </AdminHighlightPanel>
  )
}
