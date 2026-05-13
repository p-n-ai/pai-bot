import { hasNumberProps, hasStringProps, isRecord } from './type-guards'

export interface AIProviderUsage {
  provider: string
  model: string
  messages: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
}

export interface AIUsageDailyPoint {
  date: string
  messages: number
  tokens: number
  cost_usd?: number | null
}

export interface AIUsageSummary {
  total_messages: number
  total_input_tokens: number
  total_output_tokens: number
  providers: Array<AIProviderUsage>
  monthly_cost_usd?: number | null
  budget_limit_usd?: number | null
  per_student_average_tokens?: number | null
  per_student_average_cost_usd?: number | null
  budget_limit_tokens?: number | null
  budget_used_tokens?: number | null
  budget_remaining_tokens?: number | null
  budget_period_start?: string
  budget_period_end?: string
  daily_usage?: Array<AIUsageDailyPoint>
}

export interface UpsertTokenBudgetWindowInput {
  budget_tokens: number
  period_start: string
  period_end: string
}

export function isAIUsageSummary(value: unknown): value is AIUsageSummary {
  return isRecord(value) && hasAIUsageSummaryShape(value)
}

function hasAIUsageSummaryShape(value: Record<string, unknown>): boolean {
  return (
    hasNumberProps(value, [
      'total_messages',
      'total_input_tokens',
      'total_output_tokens',
    ]) &&
    Array.isArray(value.providers) &&
    value.providers.every(isAIProviderUsage)
  )
}

function isAIProviderUsage(value: unknown): value is AIProviderUsage {
  if (!isRecord(value)) {
    return false
  }

  return (
    hasStringProps(value, ['provider', 'model']) &&
    hasNumberProps(value, [
      'messages',
      'input_tokens',
      'output_tokens',
      'total_tokens',
    ])
  )
}
