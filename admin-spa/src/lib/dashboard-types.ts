import { Option, Schema } from 'effect'

export const ClassProgressStudentSchema = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  topics: Schema.Record(Schema.String, Schema.Number),
})

export type ClassProgressStudent = typeof ClassProgressStudentSchema.Type

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

export type ClassProgress = typeof ClassProgressSchema.Type

const decodeClassProgress = Schema.decodeUnknownOption(ClassProgressWire)

/** Decodes class progress while normalizing nullable backend slices to arrays. */
export function readClassProgress(value: unknown): ClassProgress | null {
  return Option.match(decodeClassProgress(value), {
    onNone: () => null,
    onSome: (progress) => ({
      students: progress.students ?? [],
      topic_ids: progress.topic_ids ?? [],
    }),
  })
}
