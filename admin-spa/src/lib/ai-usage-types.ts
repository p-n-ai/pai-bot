import { Schema } from 'effect'

const OptionalNullableNumber = Schema.optionalKey(Schema.NullOr(Schema.Number))

export const AIProviderUsageSchema = Schema.Struct({
  provider: Schema.String,
  model: Schema.String,
  messages: Schema.Number,
  input_tokens: Schema.Number,
  output_tokens: Schema.Number,
  total_tokens: Schema.Number,
})

export type AIProviderUsage = typeof AIProviderUsageSchema.Type

export const AIUsageDailyPointSchema = Schema.Struct({
  date: Schema.String,
  messages: Schema.Number,
  tokens: Schema.Number,
  cost_usd: OptionalNullableNumber,
})

export type AIUsageDailyPoint = typeof AIUsageDailyPointSchema.Type

export const AIUsageSummarySchema = Schema.Struct({
  total_messages: Schema.Number,
  total_input_tokens: Schema.Number,
  total_output_tokens: Schema.Number,
  providers: Schema.mutable(Schema.Array(AIProviderUsageSchema)),
  monthly_cost_usd: OptionalNullableNumber,
  budget_limit_usd: OptionalNullableNumber,
  per_student_average_tokens: OptionalNullableNumber,
  per_student_average_cost_usd: OptionalNullableNumber,
  budget_limit_tokens: OptionalNullableNumber,
  budget_used_tokens: OptionalNullableNumber,
  budget_remaining_tokens: OptionalNullableNumber,
  budget_period_start: Schema.optionalKey(Schema.String),
  budget_period_end: Schema.optionalKey(Schema.String),
  daily_usage: Schema.optionalKey(
    Schema.mutable(Schema.Array(AIUsageDailyPointSchema)),
  ),
})

export type AIUsageSummary = typeof AIUsageSummarySchema.Type

export interface UpsertTokenBudgetWindowInput {
  budget_tokens: number
  period_start: string
  period_end: string
}

const matchesAIUsageSummary = Schema.is(AIUsageSummarySchema)

/** Returns whether an unknown response satisfies the complete AI usage contract. */
export function isAIUsageSummary(value: unknown): value is AIUsageSummary {
  return matchesAIUsageSummary(value)
}
