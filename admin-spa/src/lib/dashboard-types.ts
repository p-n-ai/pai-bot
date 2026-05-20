import { isNumber, isRecord, isString } from './type-guards'

export interface ClassProgressStudent {
  id: string
  name: string
  topics: Record<string, number>
}

export interface ClassProgress {
  students: Array<ClassProgressStudent>
  topic_ids: Array<string>
}

export function readClassProgress(value: unknown): ClassProgress | null {
  return isRecord(value) ? buildClassProgress(value) : null
}

function buildClassProgress(value: Record<string, unknown>) {
  const students = readArrayOrNull(value.students, isClassProgressStudent)
  const topicIDs = readArrayOrNull(value.topic_ids, isString)

  return students && topicIDs ? { students, topic_ids: topicIDs } : null
}

function readArrayOrNull<T>(
  value: unknown,
  guard: (item: unknown) => item is T,
): Array<T> | null {
  const array = normalizeNullableArray(value)
  return Array.isArray(array) ? readGuardedArray(array, guard) : null
}

function normalizeNullableArray(value: unknown) {
  return value === null ? [] : value
}

function readGuardedArray<T>(
  value: Array<unknown>,
  guard: (item: unknown) => item is T,
): Array<T> | null {
  return value.every(guard) ? value : null
}

function isClassProgressStudent(value: unknown): value is ClassProgressStudent {
  return isRecord(value) && hasStudentShape(value)
}

function hasStudentShape(value: Record<string, unknown>): boolean {
  return [hasStudentID, hasStudentName, hasTopicScores].every((check) =>
    check(value),
  )
}

function hasStudentID(value: Record<string, unknown>): boolean {
  return typeof value.id === 'string'
}

function hasStudentName(value: Record<string, unknown>): boolean {
  return typeof value.name === 'string'
}

function hasTopicScores(value: Record<string, unknown>): boolean {
  return isRecord(value.topics) && Object.values(value.topics).every(isNumber)
}
