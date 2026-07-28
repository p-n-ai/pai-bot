import { Schema } from 'effect'

import {
  LearningStreakSchema,
  ProgressItemSchema,
  StudentProfileSchema,
} from './learner-types'

export const ParentProfileSchema = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  email: Schema.String,
  child_ids: Schema.mutable(Schema.Array(Schema.String)),
  created_at: Schema.String,
})

export type ParentProfile = typeof ParentProfileSchema.Type

export const WeeklyStatsSchema = Schema.Struct({
  days_active: Schema.Number,
  messages_exchanged: Schema.Number,
  quizzes_completed: Schema.Number,
  needs_review_count: Schema.Number,
})

export type WeeklyStats = typeof WeeklyStatsSchema.Type

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

export type ParentSummary = typeof ParentSummarySchema.Type

const matchesParentSummary = Schema.is(ParentSummarySchema)

/** Returns whether an unknown response satisfies the complete parent summary contract. */
export function isParentSummary(value: unknown): value is ParentSummary {
  return matchesParentSummary(value)
}
