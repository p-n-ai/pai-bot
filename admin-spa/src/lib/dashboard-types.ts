import { Option, Schema, flow } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const ClassProgressStudentSchema = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  topics: Schema.Record(Schema.String, Schema.Number),
})

export interface ClassProgressStudent extends EffectSchema.Type<
  typeof ClassProgressStudentSchema
> {}

const ClassProgressStudents = Schema.mutable(
  Schema.Array(ClassProgressStudentSchema),
)
const TopicIDs = Schema.mutable(Schema.Array(Schema.String))

const ClassProgressWire = Schema.Struct({
  students: Schema.NullOr(ClassProgressStudents),
  topic_ids: Schema.NullOr(TopicIDs),
})

export const ClassProgressSchema = Schema.Struct({
  students: ClassProgressStudents,
  topic_ids: TopicIDs,
})

export interface ClassProgress extends EffectSchema.Type<
  typeof ClassProgressSchema
> {}

const decodeClassProgress = Schema.decodeUnknownOption(ClassProgressWire)

function normalizeClassProgress(
  progress: typeof ClassProgressWire.Type,
): ClassProgress {
  return {
    students: progress.students ?? [],
    topic_ids: progress.topic_ids ?? [],
  }
}

/** Decodes class progress while normalizing nullable backend slices to arrays. */
export const readClassProgress = flow(
  decodeClassProgress,
  Option.map(normalizeClassProgress),
  Option.getOrNull,
)
