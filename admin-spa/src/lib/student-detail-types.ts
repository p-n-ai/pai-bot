import { Schema } from 'effect'
import {
  LearningStreakSchema,
  ProgressItemSchema,
  StudentProfileSchema,
} from './learner-types'
import type { Schema as EffectSchema } from 'effect/Schema'

export const StudentDetailSchema = Schema.Struct({
  student: StudentProfileSchema,
  progress: Schema.mutable(Schema.Array(ProgressItemSchema)),
  streak: LearningStreakSchema,
})

export interface StudentDetail extends EffectSchema.Type<
  typeof StudentDetailSchema
> {}

export const StudentConversationSchema = Schema.Struct({
  id: Schema.String,
  timestamp: Schema.String,
  role: Schema.Literals(['student', 'assistant']),
  text: Schema.String,
})

export interface StudentConversation extends EffectSchema.Type<
  typeof StudentConversationSchema
> {}

const StudentConversations = Schema.mutable(
  Schema.Array(StudentConversationSchema),
)
/** Returns whether an unknown response satisfies the student detail contract. */
export const isStudentDetail = Schema.is(StudentDetailSchema)

/** Returns whether an unknown response is a valid conversation list. */
export const isStudentConversations = Schema.is(StudentConversations)
