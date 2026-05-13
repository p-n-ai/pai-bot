import type {
  AIProviderUsage,
  AIUsageDailyPoint,
  AIUsageSummary,
} from './ai-usage-types'

export interface AIUsageView {
  totalTokens: number
  totalMessages: number
  topProvider: AIProviderUsage | null
  budgetLabel: string
  budgetRemaining: number | null
  budgetWindowLabel: string
  budgetLimit: number | null
  budgetUnit: 'tokens' | 'usd' | 'none'
  averageTokensPerLearner: number | null
  dailyTrendPeak: number
  dailyUsage: Array<AIUsageDailyPoint>
  hasDailyTrend: boolean
  monthlyCost: number | null
  providers: Array<AIProviderUsage>
  usdBudgetLimit: number | null
}

interface BudgetView {
  label: string
  limit: number | null
  remaining: number | null
  unit: AIUsageView['budgetUnit']
}

interface TokenBudgetNumbers {
  limit: number
  remaining: number | null
  used: number
}

type BudgetStatusKey = 'exceeded' | 'near' | 'within'
type BudgetSource = 'token' | 'usd'

const budgetLabels: Record<BudgetSource, Record<BudgetStatusKey, string>> = {
  token: {
    exceeded: 'Token budget exceeded',
    near: 'Near token budget limit',
    within: 'Within token budget',
  },
  usd: {
    exceeded: 'Budget exceeded',
    near: 'Near budget limit',
    within: 'Within budget',
  },
}

const emptyBudgetView: BudgetView = {
  label: 'No token budget configured',
  limit: null,
  remaining: null,
  unit: 'none',
}

export function getAIUsageView(usage: AIUsageSummary): AIUsageView {
  const totalTokens = usage.total_input_tokens + usage.total_output_tokens
  const budget = getBudgetView(usage)
  const dailyTrend = getDailyTrendView(readDailyUsage(usage))
  const usdSummary = getUSDSummary(usage)

  return {
    totalTokens,
    totalMessages: usage.total_messages,
    topProvider: getTopProvider(usage.providers),
    budgetLabel: budget.label,
    budgetRemaining: budget.remaining,
    budgetWindowLabel: formatBudgetWindowLabel(
      readOptionalString(usage.budget_period_start),
      readOptionalString(usage.budget_period_end),
    ),
    budgetLimit: budget.limit,
    budgetUnit: budget.unit,
    averageTokensPerLearner: readNullableNumber(
      usage.per_student_average_tokens,
    ),
    dailyTrendPeak: dailyTrend.peak,
    dailyUsage: dailyTrend.usage,
    hasDailyTrend: dailyTrend.hasTrend,
    monthlyCost: usdSummary.monthlyCost,
    providers: usage.providers,
    usdBudgetLimit: usdSummary.budgetLimit,
  }
}

export function formatCompactNumber(value: number | null): string {
  if (value === null) {
    return '--'
  }

  return new Intl.NumberFormat('en', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value)
}

export function formatBudgetAmount(
  value: number | null,
  unit: AIUsageView['budgetUnit'],
): string {
  if (unit === 'usd') {
    return formatUSD(value)
  }

  return formatCompactNumber(value)
}

function formatBudgetWindowLabel(start: string, end: string): string {
  return budgetWindowFormatters[getBudgetWindowKey(start, end)](start, end)
}

type BudgetWindowKey = 'both' | 'empty' | 'end' | 'start'

const budgetWindowFormatters: Record<
  BudgetWindowKey,
  (start: string, end: string) => string
> = {
  both: (start, end) =>
    `${formatAIUsageDateLabel(start)} to ${formatAIUsageDateLabel(end)}`,
  empty: () => 'No active token window',
  end: (_start, end) => `Ends ${formatAIUsageDateLabel(end)}`,
  start: (start) => `Started ${formatAIUsageDateLabel(start)}`,
}

function getBudgetWindowKey(start: string, end: string): BudgetWindowKey {
  const key = `${Number(Boolean(start))}${Number(Boolean(end))}`
  const keys: Record<string, BudgetWindowKey> = {
    '00': 'empty',
    '01': 'end',
    '10': 'start',
    '11': 'both',
  }

  return keys[key] ?? 'empty'
}

export function formatAIUsageDateLabel(value: string): string {
  if (!value) {
    return 'Not set'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return aiUsageDayFormatter.format(date)
}

const aiUsageDayFormatter = new Intl.DateTimeFormat('en-US', {
  day: 'numeric',
  month: 'short',
  timeZone: 'UTC',
})

function formatUSD(value: number | null): string {
  if (value === null) {
    return '--'
  }

  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: value < 100 ? 2 : 0,
    maximumFractionDigits: 2,
  }).format(value)
}

function getTopProvider(
  providers: Array<AIProviderUsage>,
): AIProviderUsage | null {
  return providers.reduce<AIProviderUsage | null>((best, current) => {
    if (!best || current.total_tokens > best.total_tokens) {
      return current
    }

    return best
  }, null)
}

function getDailyTrendPeak(
  dailyUsage: NonNullable<AIUsageSummary['daily_usage']>,
): number {
  return dailyUsage.reduce((peak, point) => Math.max(peak, point.tokens), 0)
}

function getDailyTrendView(dailyUsage: Array<AIUsageDailyPoint>): {
  hasTrend: boolean
  peak: number
  usage: Array<AIUsageDailyPoint>
} {
  return {
    hasTrend: dailyUsage.length > 0,
    peak: getDailyTrendPeak(dailyUsage),
    usage: dailyUsage,
  }
}

function readDailyUsage(usage: AIUsageSummary): Array<AIUsageDailyPoint> {
  return usage.daily_usage ?? []
}

function getUSDSummary(usage: AIUsageSummary): {
  budgetLimit: number | null
  monthlyCost: number | null
} {
  return {
    budgetLimit: usage.budget_limit_usd ?? null,
    monthlyCost: usage.monthly_cost_usd ?? null,
  }
}

function getBudgetView(usage: AIUsageSummary): BudgetView {
  return getTokenBudgetView(usage) ?? getUSDBudgetView(usage) ?? emptyBudgetView
}

function getTokenBudgetView(usage: AIUsageSummary): BudgetView | null {
  const budget = getTokenBudgetNumbers(usage)
  if (!budget) {
    return null
  }

  return {
    label: getBudgetLabel(budget.used / budget.limit, 'token'),
    limit: budget.limit,
    remaining: budget.remaining,
    unit: 'tokens',
  }
}

function getTokenBudgetNumbers(
  usage: AIUsageSummary,
): TokenBudgetNumbers | null {
  const limit = readPositiveNumber(usage.budget_limit_tokens)
  if (limit === null) {
    return null
  }

  return getTokenBudgetWithLimit(limit, usage)
}

function getTokenBudgetWithLimit(
  limit: number,
  usage: AIUsageSummary,
): TokenBudgetNumbers | null {
  const used = readNumber(usage.budget_used_tokens)
  if (used === null) {
    return null
  }

  return { limit, used, remaining: usage.budget_remaining_tokens ?? null }
}

function getUSDBudgetView(usage: AIUsageSummary): BudgetView | null {
  const monthlyCost = usage.monthly_cost_usd
  const limit = usage.budget_limit_usd

  if (!hasNumber(monthlyCost) || !hasPositiveNumber(limit)) {
    return null
  }

  return {
    label: getBudgetLabel(monthlyCost / limit, 'usd'),
    limit,
    remaining: Math.max(limit - monthlyCost, 0),
    unit: 'usd',
  }
}

function getBudgetLabel(ratio: number, unit: 'token' | 'usd'): string {
  return budgetLabels[unit][getBudgetStatusKey(ratio)]
}

function getBudgetStatusKey(ratio: number): BudgetStatusKey {
  if (ratio >= 1) {
    return 'exceeded'
  }

  return ratio >= 0.8 ? 'near' : 'within'
}

function hasNumber(value: number | null | undefined): value is number {
  return typeof value === 'number'
}

function hasPositiveNumber(value: number | null | undefined): value is number {
  return hasNumber(value) && value > 0
}

function readNumber(value: number | null | undefined): number | null {
  return hasNumber(value) ? value : null
}

function readPositiveNumber(value: number | null | undefined): number | null {
  return hasPositiveNumber(value) ? value : null
}

function readNullableNumber(value: number | null | undefined): number | null {
  return value ?? null
}

function readOptionalString(value: string | undefined): string {
  return value ?? ''
}
