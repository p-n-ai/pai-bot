import { hasStringProps, isRecord } from './type-guards'

export interface JoinClassView {
  class_id: string
  class_name: string
  class_slug: string
  curriculum_label: string
  school_name: string
}

export function isJoinClassView(value: unknown): value is JoinClassView {
  if (!isRecord(value)) {
    return false
  }

  return hasStringProps(value, [
    'class_id',
    'class_name',
    'class_slug',
    'curriculum_label',
    'school_name',
  ])
}
