import type { AIUsageView } from '@/lib/ai-usage-view'
import { StatePanel } from '@/components/shared/state-panel'
import { SurfaceSection } from '@/components/shared/surface-section'
import {
  formatAIUsageDateLabel,
  formatCompactNumber,
} from '@/lib/ai-usage-view'

export function AIUsageDailyTrend({ view }: { view: AIUsageView }) {
  return (
    <SurfaceSection
      description='Compare daily token use across the current reporting period.'
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
          Daily totals will appear after this school starts using AI features.
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
