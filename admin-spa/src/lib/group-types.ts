import { Schema } from 'effect'

export const GroupRecordSchema = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  type: Schema.Literals(['class', 'study_group']),
  description: Schema.optionalKey(Schema.String),
  syllabus: Schema.optionalKey(Schema.String),
  subject: Schema.optionalKey(Schema.String),
  cadence: Schema.optionalKey(Schema.String),
  join_code: Schema.String,
  member_count: Schema.Number,
  created_at: Schema.optionalKey(Schema.String),
  updated_at: Schema.optionalKey(Schema.String),
})

export type GroupRecord = typeof GroupRecordSchema.Type

export const GroupMemberRecordSchema = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  role: Schema.Literals(['member', 'leader', 'teacher']),
  channel: Schema.String,
  mastery: Schema.Number,
})

export type GroupMemberRecord = typeof GroupMemberRecordSchema.Type

export const GroupDetailSchema = Schema.Struct({
  ...GroupRecordSchema.fields,
  members: Schema.mutable(Schema.Array(GroupMemberRecordSchema)),
})

export type GroupDetail = typeof GroupDetailSchema.Type

export interface CreateGroupInput {
  name: string
  type: 'class'
  syllabus: string
  cadence?: string
  subject: 'Mathematics'
}

const matchesGroupRecord = Schema.is(GroupRecordSchema)
const matchesGroupDetail = Schema.is(GroupDetailSchema)

/** Returns whether an unknown response satisfies the group summary contract. */
export function isGroupRecord(value: unknown): value is GroupRecord {
  return matchesGroupRecord(value)
}

/** Returns whether an unknown response satisfies the group detail contract. */
export function isGroupDetail(value: unknown): value is GroupDetail {
  return matchesGroupDetail(value)
}
