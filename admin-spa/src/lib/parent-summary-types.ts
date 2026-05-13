import {
  isLearningStreak,
  isProgressItem,
  isStudentProfile,
} from './learner-types'
import {
  hasNumberProps,
  hasStringProps,
  isRecord,
  isString,
} from './type-guards'
import type {
  LearningStreak,
  ProgressItem,
  StudentProfile,
} from './learner-types'

export interface ParentProfile {
  id: string
  name: string
  email: string
  child_ids: Array<string>
  created_at: string
}

export interface WeeklyStats {
  days_active: number
  messages_exchanged: number
  quizzes_completed: number
  needs_review_count: number
}

export interface ParentSummary {
  parent: ParentProfile
  child: StudentProfile
  streak: LearningStreak
  weekly_stats: WeeklyStats
  mastery: Array<ProgressItem>
  encouragement: {
    headline: string
    text: string
  }
}

export function isParentSummary(value: unknown): value is ParentSummary {
  if (!isRecord(value)) {
    return false
  }

  return [
    isParentProfile(value.parent),
    isStudentProfile(value.child),
    isLearningStreak(value.streak),
    isWeeklyStats(value.weekly_stats),
    Array.isArray(value.mastery) && value.mastery.every(isProgressItem),
    isEncouragement(value.encouragement),
  ].every(Boolean)
}

function isParentProfile(value: unknown): value is ParentProfile {
  return isRecord(value) && hasParentProfileShape(value)
}

function isWeeklyStats(value: unknown): value is WeeklyStats {
  return (
    isRecord(value) &&
    hasNumberProps(value, [
      'days_active',
      'messages_exchanged',
      'quizzes_completed',
      'needs_review_count',
    ])
  )
}

function isEncouragement(value: unknown): boolean {
  return isRecord(value) && hasStringProps(value, ['headline', 'text'])
}

function hasParentProfileShape(value: Record<string, unknown>): boolean {
  return (
    hasStringProps(value, ['id', 'name', 'email', 'created_at']) &&
    Array.isArray(value.child_ids) &&
    value.child_ids.every(isString)
  )
}
