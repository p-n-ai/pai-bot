import {
  hasNumberProps,
  hasStringProps,
  isRecord,
  optionalStringOrNull,
} from './type-guards'

export interface StudentProfile {
  id: string
  name: string
  external_id: string
  channel: string
  form: string
  created_at: string
}

export interface ProgressItem {
  topic_id: string
  mastery_score: number
  ease_factor: number
  interval_days: number
  next_review_at: string | null
  last_studied_at: string | null
}

export interface LearningStreak {
  current: number
  longest: number
  total_xp: number
}

export function isStudentProfile(value: unknown): value is StudentProfile {
  return isRecord(value) && hasStudentProfileShape(value)
}

export function isProgressItem(value: unknown): value is ProgressItem {
  return isRecord(value) && hasProgressItemShape(value)
}

export function isLearningStreak(value: unknown): value is LearningStreak {
  return (
    isRecord(value) && hasNumberProps(value, ['current', 'longest', 'total_xp'])
  )
}

function hasStudentProfileShape(value: Record<string, unknown>): boolean {
  return hasStringProps(value, [
    'id',
    'name',
    'external_id',
    'channel',
    'form',
    'created_at',
  ])
}

function hasProgressItemShape(value: Record<string, unknown>): boolean {
  return (
    hasStringProps(value, ['topic_id']) &&
    hasNumberProps(value, ['mastery_score', 'ease_factor', 'interval_days']) &&
    hasReviewDates(value)
  )
}

function hasReviewDates(value: Record<string, unknown>): boolean {
  return (
    optionalStringOrNull(value.next_review_at) &&
    optionalStringOrNull(value.last_studied_at)
  )
}
