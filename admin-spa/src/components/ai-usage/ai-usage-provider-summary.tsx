import type { AIUsageView } from '@/lib/ai-usage-view'
import { StatItem } from '@/components/shared/stat-item'
import { formatBudgetAmount } from '@/lib/ai-usage-view'

export function AIUsageProviderSummary({ view }: { view: AIUsageView }) {
  return (
    <section className='stat-grid' aria-label='Provider summary'>
      <StatItem
        label='Monthly cost'
        value={formatBudgetAmount(view.monthlyCost, 'usd')}
      />
      <StatItem
        label='Budget cap (USD)'
        value={formatUSDBudgetLimit(view.usdBudgetLimit)}
      />
      <StatItem
        label='Top provider'
        value={view.topProvider?.provider ?? 'None yet'}
      />
    </section>
  )
}

function formatUSDBudgetLimit(value: number | null): string {
  return value === null ? 'Not set' : formatBudgetAmount(value, 'usd')
}
