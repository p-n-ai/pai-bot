import { Schema } from 'effect'

export const StudentProfileSchema = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  external_id: Schema.String,
  channel: Schema.String,
  form: Schema.String,
  created_at: Schema.String,
})

export type StudentProfile = typeof StudentProfileSchema.Type

export const ProgressItemSchema = Schema.Struct({
  topic_id: Schema.String,
  mastery_score: Schema.Number,
  ease_factor: Schema.Number,
  interval_days: Schema.Number,
  next_review_at: Schema.NullOr(Schema.String),
  last_studied_at: Schema.NullOr(Schema.String),
})

export type ProgressItem = typeof ProgressItemSchema.Type

export const LearningStreakSchema = Schema.Struct({
  current: Schema.Number,
  longest: Schema.Number,
  total_xp: Schema.Number,
})

export type LearningStreak = typeof LearningStreakSchema.Type

const matchesStudentProfile = Schema.is(StudentProfileSchema)
const matchesProgressItem = Schema.is(ProgressItemSchema)
const matchesLearningStreak = Schema.is(LearningStreakSchema)

/** Returns whether an unknown response satisfies the student profile contract. */
export function isStudentProfile(value: unknown): value is StudentProfile {
  return matchesStudentProfile(value)
}

/** Returns whether an unknown response satisfies the progress item contract. */
export function isProgressItem(value: unknown): value is ProgressItem {
  return matchesProgressItem(value)
}

/** Returns whether an unknown response satisfies the learning streak contract. */
export function isLearningStreak(value: unknown): value is LearningStreak {
  return matchesLearningStreak(value)
}
