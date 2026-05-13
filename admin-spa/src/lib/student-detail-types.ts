import {
  isLearningStreak,
  isProgressItem,
  isStudentProfile,
} from './learner-types'
import { hasStringProps, isRecord } from './type-guards'
import type {
  LearningStreak,
  ProgressItem,
  StudentProfile,
} from './learner-types'

export interface StudentDetail {
  student: StudentProfile
  progress: Array<ProgressItem>
  streak: LearningStreak
}

export interface StudentConversation {
  id: string
  timestamp: string
  role: 'student' | 'assistant'
  text: string
}

export function isStudentDetail(value: unknown): value is StudentDetail {
  if (!isRecord(value)) {
    return false
  }

  return [
    isStudentProfile(value.student),
    isLearningStreak(value.streak),
    Array.isArray(value.progress) && value.progress.every(isProgressItem),
  ].every(Boolean)
}

export function isStudentConversations(
  value: unknown,
): value is Array<StudentConversation> {
  return Array.isArray(value) && value.every(isStudentConversation)
}

function isStudentConversation(value: unknown): value is StudentConversation {
  return isRecord(value) && hasStudentConversationShape(value)
}

function hasStudentConversationShape(value: Record<string, unknown>): boolean {
  return (
    hasStringProps(value, ['id', 'timestamp', 'text']) &&
    isMessageRole(value.role)
  )
}

function isMessageRole(value: unknown): value is StudentConversation['role'] {
  return value === 'student' || value === 'assistant'
}
