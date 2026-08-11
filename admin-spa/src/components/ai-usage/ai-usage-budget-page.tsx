import { GaugeIcon } from 'lucide-react'

import type { AIUsageSummary } from '@/lib/ai-usage-types'
import type { AIUsageView } from '@/lib/ai-usage-view'
import { AIUsageBudgetEditor } from '@/components/ai-usage/ai-usage-budget-editor'
import { formatCompactNumber } from '@/lib/ai-usage-view'

type BudgetSnapshot =
  | { status: 'empty' }
  | {
      limit: number
      progress: number | null
      remaining: number | null
      status: 'active'
      used: number | null
      window: string
    }

/** Renders the dedicated school-wide token budget workspace. */
export function AIUsageBudgetPage({
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
  const snapshot = getBudgetSnapshot(usage, view)

  return (
    <section
      aria-label='AI budget configuration'
      className='mt-8 overflow-hidden rounded-[1.75rem] bg-[var(--admin-surface)] ring-1 ring-[var(--admin-line)]'
    >
      <div className='grid lg:grid-cols-[minmax(19rem,0.86fr)_minmax(24rem,1.14fr)]'>
        <BudgetStatusPanel snapshot={snapshot} view={view} />
        <div className='flex min-w-0 flex-col justify-center p-6 sm:p-8 lg:p-10'>
          <AIUsageBudgetEditor
            canManageBudget={canManageBudget}
            onSaved={onSaved}
            usage={usage}
          />
        </div>
      </div>
    </section>
  )
}

function BudgetStatusPanel({
  snapshot,
  view,
}: {
  snapshot: BudgetSnapshot
  view: AIUsageView
}) {
  return (
    <div className='flex flex-col bg-[var(--admin-ink)] p-6 text-[var(--admin-surface)] sm:p-8 lg:min-h-[34rem] lg:p-10'>
      <div className='flex items-center justify-between gap-4'>
        <p className='flex items-center gap-2 text-xs font-semibold tracking-[0.14em] text-[var(--admin-nav-text)] uppercase'>
          <span
            aria-hidden='true'
            className='size-2 rounded-full bg-[var(--admin-accent)]'
          />
          {snapshot.status === 'active' ? 'Active budget' : 'No active budget'}
        </p>
        <GaugeIcon
          aria-hidden='true'
          className='size-5 text-[var(--admin-accent)]'
          strokeWidth={1.5}
        />
      </div>

      {snapshot.status === 'active' ? (
        <ActiveBudgetStatus snapshot={snapshot} view={view} />
      ) : (
        <EmptyBudgetStatus />
      )}
    </div>
  )
}

function EmptyBudgetStatus() {
  return (
    <div className='flex flex-1 flex-col'>
      <div className='my-auto py-12'>
        <h2 className='max-w-[12ch] text-[clamp(2rem,4vw,3.25rem)] leading-[0.98] font-semibold tracking-[-0.045em] text-balance'>
          Set a clear limit for this school.
        </h2>
        <p className='mt-5 max-w-md text-sm leading-6 text-[var(--admin-nav-text)]'>
          Choose a token allowance and the dates it applies. Usage is measured
          against one school-wide window.
        </p>
      </div>

      <dl className='grid gap-5 border-t border-white/12 pt-6 text-sm'>
        <BudgetDefinition
          description='The total tokens available during the window.'
          label='Allowance'
        />
        <BudgetDefinition
          description='The start and end dates used for measurement.'
          label='Schedule'
        />
        <BudgetDefinition
          description='All AI usage in this school workspace.'
          label='Scope'
        />
      </dl>
    </div>
  )
}

function ActiveBudgetStatus({
  snapshot,
  view,
}: {
  snapshot: Extract<BudgetSnapshot, { status: 'active' }>
  view: AIUsageView
}) {
  return (
    <div className='flex flex-1 flex-col'>
      <div className='my-auto py-12'>
        <p className='text-sm font-medium text-[var(--admin-nav-text)]'>
          Remaining allowance
        </p>
        <h2 className='mt-2 text-[clamp(3rem,7vw,5rem)] leading-none font-semibold tracking-[-0.06em] tabular-nums'>
          {formatCompactNumber(snapshot.remaining)}
          <span className='ms-2 text-lg tracking-[-0.02em] text-[var(--admin-nav-muted)]'>
            tokens
          </span>
        </h2>
        <p className='mt-4 text-sm text-[var(--admin-nav-text)]'>
          {view.budgetLabel}
        </p>
        {snapshot.progress !== null ? (
          <div className='mt-7 grid gap-2'>
            <progress
              aria-label='Token budget used'
              className='budget-workspace-progress'
              max={100}
              value={snapshot.progress}
            />
            <p className='text-xs font-medium text-[var(--admin-nav-muted)]'>
              {snapshot.progress}% used
            </p>
          </div>
        ) : null}
      </div>

      <dl className='grid grid-cols-2 gap-x-6 gap-y-5 border-t border-white/12 pt-6 text-sm'>
        <BudgetMetric label='Used' value={formatCompactNumber(snapshot.used)} />
        <BudgetMetric
          label='Limit'
          value={formatCompactNumber(snapshot.limit)}
        />
        <div className='col-span-2'>
          <BudgetMetric label='Window' value={snapshot.window} />
        </div>
      </dl>
    </div>
  )
}

function BudgetDefinition({
  description,
  label,
}: {
  description: string
  label: string
}) {
  return (
    <div className='grid grid-cols-[5.5rem_minmax(0,1fr)] gap-4'>
      <dt className='font-semibold text-[var(--admin-surface)]'>{label}</dt>
      <dd className='leading-5 text-[var(--admin-nav-muted)]'>{description}</dd>
    </div>
  )
}

function BudgetMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className='grid gap-1.5'>
      <dt className='text-xs font-medium tracking-[0.1em] text-[var(--admin-nav-muted)] uppercase'>
        {label}
      </dt>
      <dd className='font-semibold text-[var(--admin-surface)] tabular-nums'>
        {value}
      </dd>
    </div>
  )
}

function getBudgetSnapshot(
  usage: AIUsageSummary,
  view: AIUsageView,
): BudgetSnapshot {
  const limit = usage.budget_limit_tokens
  if (typeof limit !== 'number' || limit <= 0) {
    return { status: 'empty' }
  }

  const reportedRemaining = readNonNegativeNumber(usage.budget_remaining_tokens)
  const remaining =
    reportedRemaining === null ? null : Math.min(limit, reportedRemaining)
  const recordedUsed = readNonNegativeNumber(usage.budget_used_tokens)
  const used =
    recordedUsed ?? (remaining === null ? null : Math.max(0, limit - remaining))

  return {
    limit,
    progress:
      used === null
        ? null
        : Math.min(100, Math.max(0, Math.round((used / limit) * 100))),
    remaining,
    status: 'active',
    used,
    window: view.budgetWindowLabel,
  }
}

function readNonNegativeNumber(
  value: number | null | undefined,
): number | null {
  return typeof value === 'number' ? Math.max(0, value) : null
}
