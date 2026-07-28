import { Schema } from 'effect'

import {
  LearningStreakSchema,
  ProgressItemSchema,
  StudentProfileSchema,
} from './learner-types'

export const StudentDetailSchema = Schema.Struct({
  student: StudentProfileSchema,
  progress: Schema.mutable(Schema.Array(ProgressItemSchema)),
  streak: LearningStreakSchema,
})

export type StudentDetail = typeof StudentDetailSchema.Type

export const StudentConversationSchema = Schema.Struct({
  id: Schema.String,
  timestamp: Schema.String,
  role: Schema.Literals(['student', 'assistant']),
  text: Schema.String,
})

export type StudentConversation = typeof StudentConversationSchema.Type

const StudentConversations = Schema.mutable(
  Schema.Array(StudentConversationSchema),
)
const matchesStudentDetail = Schema.is(StudentDetailSchema)
const matchesStudentConversations = Schema.is(StudentConversations)

/** Returns whether an unknown response satisfies the student detail contract. */
export function isStudentDetail(value: unknown): value is StudentDetail {
  return matchesStudentDetail(value)
}

/** Returns whether an unknown response is a valid conversation list. */
export function isStudentConversations(
  value: unknown,
): value is Array<StudentConversation> {
  return matchesStudentConversations(value)
}
