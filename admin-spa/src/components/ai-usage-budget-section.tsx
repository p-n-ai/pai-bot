import type { AIUsageSummary } from '@/lib/ai-usage-types'
import type { AIUsageView } from '@/lib/ai-usage-view'
import { AIUsageBudgetEditor } from '@/components/ai-usage-budget-editor'
import { Badge } from '@/components/ui/badge'
import { StatItem } from '@/components/stat-item'
import { SurfaceSection } from '@/components/surface-section'
import { formatBudgetAmount } from '@/lib/ai-usage-view'

export function AIUsageBudgetSection({
  canManageBudget,
  onSaved,
  usage,
  view,
}: {
  canManageBudget: boolean
  onSaved: (usage: AIUsageSummary) => void
  usage: AIUsageSummary
  view: AIUsageView
}) {
  const usedTokens =
    typeof usage.budget_remaining_tokens === 'number' &&
    typeof usage.budget_limit_tokens === 'number'
      ? Math.max(0, usage.budget_limit_tokens - usage.budget_remaining_tokens)
      : null
  const progress =
    usedTokens !== null && usage.budget_limit_tokens
      ? Math.min(
          100,
          Math.round((usedTokens / usage.budget_limit_tokens) * 100),
        )
      : null

  return (
    <SurfaceSection
      description='Budget thresholds and remaining allowance for the active tenant-wide token window.'
      title='Token budget window'
    >
      <div className='dashboard-stack'>
        <div className='flex flex-wrap items-center gap-2'>
          <Badge variant={usage.budget_limit_tokens ? 'secondary' : 'outline'}>
            {usage.budget_limit_tokens ? 'Active window' : 'No active window'}
          </Badge>
          {progress !== null ? (
            <span className='text-sm text-muted-foreground'>
              {progress}% used
            </span>
          ) : null}
        </div>
        <section className='stat-grid' aria-label='Token budget summary'>
          <StatItem label='Window' value={view.budgetWindowLabel} />
          <StatItem
            label='Limit'
            value={formatBudgetAmount(view.budgetLimit, view.budgetUnit)}
          />
          <StatItem
            label='Remaining'
            value={formatBudgetAmount(view.budgetRemaining, view.budgetUnit)}
          />
        </section>
        {progress !== null ? (
          <progress
            aria-label='Token budget used'
            className='daily-token-progress'
            max={100}
            value={progress}
          />
        ) : null}
        <AIUsageBudgetEditor
          canManageBudget={canManageBudget}
          onSaved={onSaved}
          usage={usage}
        />
      </div>
    </SurfaceSection>
  )
}
