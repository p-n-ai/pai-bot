import type { AIUsageView } from '@/lib/ai-usage-view'
import { StatItem } from '@/components/stat-item'
import { formatCompactNumber } from '@/lib/ai-usage-view'

export function AIUsageOverviewStats({ view }: { view: AIUsageView }) {
  return (
    <section className='stat-grid' aria-label='AI usage overview'>
      <StatItem
        label='Tokens'
        note={getTopProviderNote(view)}
        value={formatCompactNumber(view.totalTokens)}
      />
      <StatItem
        label='Messages'
        note='Current aggregate window'
        value={formatCompactNumber(view.totalMessages)}
      />
      <StatItem
        label='Budget'
        note={view.budgetWindowLabel}
        value={view.budgetLabel}
      />
      <StatItem
        label='Per learner'
        note={getPerLearnerNote(view)}
        value={formatCompactNumber(view.averageTokensPerLearner)}
      />
    </section>
  )
}

function getTopProviderNote(view: AIUsageView): string {
  return view.topProvider
    ? `${view.topProvider.provider} top source`
    : 'No provider activity yet'
}

function getPerLearnerNote(view: AIUsageView): string {
  return view.averageTokensPerLearner === null
    ? 'Token average'
    : 'Average tokens'
}
