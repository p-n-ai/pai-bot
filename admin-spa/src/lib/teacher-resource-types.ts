import { Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const TeacherResourceSchema = Schema.Struct({
  id: Schema.String,
  filename: Schema.String,
  title: Schema.String,
  source_type: Schema.Literals(['pdf', 'docx', 'pptx']),
  media_type: Schema.String,
  byte_size: Schema.Number,
  chunk_count: Schema.Number,
  active: Schema.Boolean,
  class_ids: Schema.mutable(Schema.Array(Schema.String)),
  created_at: Schema.String,
  updated_at: Schema.String,
  uploader_id: Schema.optionalKey(Schema.String),
  uploader_name: Schema.optionalKey(Schema.String),
  extraction_error: Schema.optionalKey(Schema.String),
})

export interface TeacherResource extends EffectSchema.Type<
  typeof TeacherResourceSchema
> {}

export interface UploadTeacherResourceInput {
  file: File
  title: string
  classIDs: Array<string>
}

/** Returns whether an unknown response satisfies the teacher-resource contract. */
export const isTeacherResource = Schema.is(TeacherResourceSchema)

export function isAllowedTeacherResourceFile(file: File): boolean {
  const extension = file.name.split('.').pop()?.toLowerCase()

  return extension === 'pdf' || extension === 'docx' || extension === 'pptx'
}
