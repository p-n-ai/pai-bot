import { Schema } from 'effect'
import {
  LearningStreakSchema,
  ProgressItemSchema,
  StudentProfileSchema,
} from './learner-types'
import type { Schema as EffectSchema } from 'effect/Schema'

export const ParentProfileSchema = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  email: Schema.String,
  child_ids: Schema.mutable(Schema.Array(Schema.String)),
  created_at: Schema.String,
})

export interface ParentProfile extends EffectSchema.Type<
  typeof ParentProfileSchema
> {}

export const WeeklyStatsSchema = Schema.Struct({
  days_active: Schema.Number,
  messages_exchanged: Schema.Number,
  quizzes_completed: Schema.Number,
  needs_review_count: Schema.Number,
})

export interface WeeklyStats extends EffectSchema.Type<
  typeof WeeklyStatsSchema
> {}

const Encouragement = Schema.Struct({
  headline: Schema.String,
  text: Schema.String,
})

export const ParentSummarySchema = Schema.Struct({
  parent: ParentProfileSchema,
  child: StudentProfileSchema,
  streak: LearningStreakSchema,
  weekly_stats: WeeklyStatsSchema,
  mastery: Schema.mutable(Schema.Array(ProgressItemSchema)),
  encouragement: Encouragement,
})

export interface ParentSummary extends EffectSchema.Type<
  typeof ParentSummarySchema
> {}

const matchesParentSummary = Schema.is(ParentSummarySchema)

/** Returns whether an unknown response satisfies the complete parent summary contract. */
export function isParentSummary(value: unknown): value is ParentSummary {
  return matchesParentSummary(value)
}
