import { Schema } from 'effect'

export const JoinClassViewSchema = Schema.Struct({
  class_id: Schema.String,
  class_name: Schema.String,
  class_slug: Schema.String,
  curriculum_label: Schema.String,
  school_name: Schema.String,
})

export type JoinClassView = typeof JoinClassViewSchema.Type

const matchesJoinClassView = Schema.is(JoinClassViewSchema)

/** Returns whether an unknown response satisfies the join-class view contract. */
export function isJoinClassView(value: unknown): value is JoinClassView {
  return matchesJoinClassView(value)
}
