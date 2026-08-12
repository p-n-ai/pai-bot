import { Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const JoinClassViewSchema = Schema.Struct({
  class_id: Schema.String,
  class_name: Schema.String,
  class_slug: Schema.String,
  curriculum_label: Schema.String,
  school_name: Schema.String,
})

export interface JoinClassView extends EffectSchema.Type<
  typeof JoinClassViewSchema
> {}

/** Returns whether an unknown response satisfies the join-class view contract. */
export const isJoinClassView = Schema.is(JoinClassViewSchema)
