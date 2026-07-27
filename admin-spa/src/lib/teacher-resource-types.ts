export interface TeacherResource {
  id: string
  filename: string
  title: string
  source_type: 'pdf' | 'docx' | 'pptx'
  media_type: string
  byte_size: number
  chunk_count: number
  active: boolean
  class_ids: Array<string>
  created_at: string
  updated_at: string
  uploader_id?: string
  uploader_name?: string
  extraction_error?: string
}

export interface UploadTeacherResourceInput {
  file: File
  title: string
  classIDs: Array<string>
}

export function isTeacherResource(value: unknown): value is TeacherResource {
  if (!isRecord(value)) {
    return false
  }

  return (
    typeof value.id === 'string' &&
    typeof value.filename === 'string' &&
    typeof value.title === 'string' &&
    isSourceType(value.source_type) &&
    typeof value.media_type === 'string' &&
    typeof value.byte_size === 'number' &&
    typeof value.chunk_count === 'number' &&
    typeof value.active === 'boolean' &&
    Array.isArray(value.class_ids) &&
    value.class_ids.every((classID) => typeof classID === 'string') &&
    typeof value.created_at === 'string' &&
    typeof value.updated_at === 'string' &&
    isOptionalString(value.uploader_id) &&
    isOptionalString(value.uploader_name) &&
    isOptionalString(value.extraction_error)
  )
}

export function isAllowedTeacherResourceFile(file: File): boolean {
  const extension = file.name.split('.').pop()?.toLowerCase()

  return extension === 'pdf' || extension === 'docx' || extension === 'pptx'
}

function isSourceType(value: unknown): value is TeacherResource['source_type'] {
  return value === 'pdf' || value === 'docx' || value === 'pptx'
}

function isOptionalString(value: unknown): boolean {
  return value === undefined || typeof value === 'string'
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
