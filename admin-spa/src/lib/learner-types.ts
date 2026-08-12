import { Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const StudentProfileSchema = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  external_id: Schema.String,
  channel: Schema.String,
  form: Schema.String,
  created_at: Schema.String,
})

export interface StudentProfile extends EffectSchema.Type<
  typeof StudentProfileSchema
> {}

export const ProgressItemSchema = Schema.Struct({
  topic_id: Schema.String,
  mastery_score: Schema.Number,
  ease_factor: Schema.Number,
  interval_days: Schema.Number,
  next_review_at: Schema.NullOr(Schema.String),
  last_studied_at: Schema.NullOr(Schema.String),
})

export interface ProgressItem extends EffectSchema.Type<
  typeof ProgressItemSchema
> {}

export const LearningStreakSchema = Schema.Struct({
  current: Schema.Number,
  longest: Schema.Number,
  total_xp: Schema.Number,
})

export interface LearningStreak extends EffectSchema.Type<
  typeof LearningStreakSchema
> {}

/** Returns whether an unknown response satisfies the student profile contract. */
export const isStudentProfile = Schema.is(StudentProfileSchema)

/** Returns whether an unknown response satisfies the progress item contract. */
export const isProgressItem = Schema.is(ProgressItemSchema)

/** Returns whether an unknown response satisfies the learning streak contract. */
export const isLearningStreak = Schema.is(LearningStreakSchema)
