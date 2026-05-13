import type { AIUsageView } from '@/lib/ai-usage-view'
import { StatePanel } from '@/components/state-panel'
import { SurfaceSection } from '@/components/surface-section'
import {
  formatAIUsageDateLabel,
  formatCompactNumber,
} from '@/lib/ai-usage-view'

export function AIUsageDailyTrend({ view }: { view: AIUsageView }) {
  return (
    <SurfaceSection
      description='Recent day-by-day token volume from the admin AI usage API.'
      title='Daily token trend'
    >
      {view.hasDailyTrend ? (
        <div className='daily-token-trend'>
          {view.dailyUsage.map((point) => (
            <DailyTrendRow
              key={point.date}
              peak={view.dailyTrendPeak}
              point={point}
            />
          ))}
        </div>
      ) : (
        <StatePanel title='No daily token trend yet'>
          Daily usage bars will appear once the tenant starts generating AI
          traffic.
        </StatePanel>
      )}
    </SurfaceSection>
  )
}

function DailyTrendRow({
  peak,
  point,
}: {
  peak: number
  point: AIUsageView['dailyUsage'][number]
}) {
  return (
    <div className='daily-token-row'>
      <div>
        <span>{formatAIUsageDateLabel(point.date)}</span>
        <span>{formatCompactNumber(point.tokens)} tokens</span>
      </div>
      <progress
        aria-label={`${formatAIUsageDateLabel(point.date)} token usage`}
        className='daily-token-progress'
        max={peak || 1}
        value={point.tokens}
      />
    </div>
  )
}
