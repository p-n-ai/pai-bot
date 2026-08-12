import type { AIUsageSummary } from '@/lib/ai-usage-types'
import type { AIUsageView } from '@/lib/ai-usage-view'
import { AIUsageBudgetEditor } from '@/components/ai-usage/ai-usage-budget-editor'
import { Badge } from '@/components/ui/badge'
import { StatItem } from '@/components/shared/stat-item'
import { SurfaceSection } from '@/components/shared/surface-section'
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
  const hasTokenBudget =
    usage.budget_limit_tokens !== null &&
    usage.budget_limit_tokens !== undefined &&
    usage.budget_limit_tokens > 0
  const usedTokens =
    usage.budget_remaining_tokens !== null &&
    usage.budget_remaining_tokens !== undefined &&
    usage.budget_limit_tokens !== null &&
    usage.budget_limit_tokens !== undefined
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
      description='Review the limit, dates, and remaining tokens for this school.'
      title='Token budget period'
    >
      <div className='grid gap-6'>
        {hasTokenBudget ? (
          <ActiveBudgetSummary progress={progress} view={view} />
        ) : (
          <EmptyBudgetSummary />
        )}
        <div className='rounded-2xl bg-[var(--admin-surface-muted)] p-4 sm:p-5'>
          <AIUsageBudgetEditor
            canManageBudget={canManageBudget}
            onSaved={onSaved}
            usage={usage}
          />
        </div>
      </div>
    </SurfaceSection>
  )
}

function ActiveBudgetSummary({
  progress,
  view,
}: {
  progress: number | null
  view: AIUsageView
}) {
  return (
    <div className='grid gap-4'>
      <div className='flex flex-wrap items-center gap-2'>
        <Badge variant='secondary'>Active period</Badge>
        {progress !== null ? (
          <span className='text-sm text-[var(--admin-muted)]'>
            {progress}% used
          </span>
        ) : null}
      </div>
      <section className='stat-grid' aria-label='Token budget summary'>
        <StatItem label='Period' value={view.budgetWindowLabel} />
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
    </div>
  )
}

function EmptyBudgetSummary() {
  return (
    <section
      aria-label='Token budget status'
      className='grid gap-3 rounded-2xl bg-[var(--admin-ink)] p-5 text-[var(--admin-surface)] sm:p-6'
    >
      <Badge className='w-fit border-0 bg-[var(--admin-accent)] text-[var(--admin-ink)]'>
        Setup needed
      </Badge>
      <div>
        <h3 className='text-xl font-semibold tracking-[-0.025em]'>
          No token budget set
        </h3>
        <p className='mt-2 max-w-xl text-sm leading-6 text-[var(--admin-nav-text)]'>
          Set a token limit and date range to manage this school’s AI use.
        </p>
      </div>
    </section>
  )
}
