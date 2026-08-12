import { useCallback, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'

import type { AIUsageSummary } from '@/lib/ai-usage-types'
import { AuthErrorAlert } from '@/components/shared/auth-error-alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { StatePanel } from '@/components/shared/state-panel'
import { upsertTokenBudgetWindow } from '@/lib/admin-api'
import { useSubmitStatus } from '@/hooks/use-submit-status'

export function AIUsageBudgetEditor({
  canManageBudget,
  onSaved,
  usage,
}: {
  canManageBudget: boolean
  onSaved: (usage: AIUsageSummary) => void
  usage: AIUsageSummary
}) {
  if (!canManageBudget) {
    return (
      <StatePanel title='Budget is read-only'>
        Ask a school administrator to change this budget.
      </StatePanel>
    )
  }

  return <EditableTokenBudget onSaved={onSaved} usage={usage} />
}

function EditableTokenBudget({
  onSaved,
  usage,
}: {
  onSaved: (usage: AIUsageSummary) => void
  usage: AIUsageSummary
}) {
  const form = useTokenBudgetForm({ onSaved, usage })
  const isConfigured = Boolean(usage.budget_limit_tokens)

  return (
    <form className='grid gap-6' onSubmit={form.handleSubmit}>
      <div>
        <p className='mb-2.5 text-xs font-semibold tracking-[0.12em] text-[var(--admin-muted)] uppercase'>
          Budget settings
        </p>
        <h3 className='text-xl leading-snug font-semibold tracking-[-0.02em] text-[var(--admin-ink)]'>
          {isConfigured ? 'Update budget' : 'Set a budget'}
        </h3>
        <p className='mt-2 max-w-md text-sm leading-6 text-[var(--admin-muted)]'>
          Set one token allowance for this school.
        </p>
      </div>
      <BudgetFields
        budgetTokens={form.budgetTokens}
        periodEnd={form.periodEnd}
        periodStart={form.periodStart}
        setBudgetTokens={form.setBudgetTokens}
        setPeriodEnd={form.setPeriodEnd}
        setPeriodStart={form.setPeriodStart}
      />

      <AuthErrorAlert message={form.error} title='Unable to save budget' />

      <div className='flex flex-col gap-4 pt-1 sm:flex-row sm:items-center sm:justify-between'>
        <p className='max-w-xs text-xs leading-5 text-[var(--admin-muted)]'>
          These dates define when the allowance starts and ends.
        </p>
        <Button disabled={form.isPending} type='submit'>
          {form.isPending ? 'Saving budget…' : 'Save token budget'}
        </Button>
      </div>
    </form>
  )
}

function useTokenBudgetForm({
  onSaved,
  usage,
}: {
  onSaved: (usage: AIUsageSummary) => void
  usage: AIUsageSummary
}) {
  const { beginSubmit, error, finishSubmit, isPending, setError } =
    useSubmitStatus('')
  const [budgetTokens, setBudgetTokens] = useState(
    readInitialBudgetTokens(usage),
  )
  const [periodStart, setPeriodStart] = useState(
    usage.budget_period_start ?? defaultStartDate(),
  )
  const [periodEnd, setPeriodEnd] = useState(
    usage.budget_period_end ?? defaultEndDate(),
  )

  const handleSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const parsedBudget = Number.parseInt(budgetTokens, 10)
      if (!Number.isFinite(parsedBudget) || parsedBudget <= 0) {
        setError('Enter a token budget greater than zero.')
        return
      }

      beginSubmit()
      upsertTokenBudgetWindow({
        budget_tokens: parsedBudget,
        period_start: periodStart,
        period_end: periodEnd,
      })
        .then(onSaved)
        .catch((caught: unknown) => {
          setError(
            caught instanceof Error
              ? caught.message
              : 'Unable to save the AI budget. Check your connection and try again.',
          )
        })
        .finally(finishSubmit)
    },
    [
      beginSubmit,
      budgetTokens,
      finishSubmit,
      onSaved,
      periodEnd,
      periodStart,
      setError,
    ],
  )

  return {
    budgetTokens,
    error,
    isPending,
    periodEnd,
    periodStart,
    setBudgetTokens,
    setPeriodEnd,
    setPeriodStart,
    handleSubmit,
  }
}

function BudgetFields({
  budgetTokens,
  periodEnd,
  periodStart,
  setBudgetTokens,
  setPeriodEnd,
  setPeriodStart,
}: {
  budgetTokens: string
  periodEnd: string
  periodStart: string
  setBudgetTokens: (value: string) => void
  setPeriodEnd: (value: string) => void
  setPeriodStart: (value: string) => void
}) {
  return (
    <div className='grid gap-4 sm:grid-cols-2'>
      <div className='sm:col-span-2'>
        <BudgetField
          id='token-budget-limit'
          label='Token limit'
          min={1}
          onChange={setBudgetTokens}
          type='number'
          value={budgetTokens}
        />
      </div>
      <BudgetField
        id='token-budget-start'
        label='Start date'
        onChange={setPeriodStart}
        type='date'
        value={periodStart}
      />
      <BudgetField
        id='token-budget-end'
        label='End date'
        onChange={setPeriodEnd}
        type='date'
        value={periodEnd}
      />
    </div>
  )
}

function BudgetField({
  id,
  label,
  min,
  onChange,
  type,
  value,
}: {
  id: string
  label: string
  min?: number
  onChange: (value: string) => void
  type: 'date' | 'number'
  value: string
}) {
  const updateValue = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      onChange(event.target.value)
    },
    [onChange],
  )

  return (
    <div className='field-stack'>
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        min={min}
        onChange={updateValue}
        required
        step={type === 'number' ? 1 : undefined}
        type={type}
        value={value}
      />
    </div>
  )
}

function readInitialBudgetTokens(usage: AIUsageSummary): string {
  const budget = usage.budget_limit_tokens ?? Number.NaN
  return Number.isFinite(budget) ? String(budget) : ''
}

function defaultStartDate(): string {
  return new Date().toISOString().slice(0, 10)
}

function defaultEndDate(): string {
  const now = new Date()
  const end = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 0))
  return end.toISOString().slice(0, 10)
}
