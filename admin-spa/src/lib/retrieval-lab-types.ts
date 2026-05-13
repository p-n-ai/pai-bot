import {
  hasNumberProps,
  hasStringProps,
  isRecord,
  isString,
} from './type-guards'

export interface RetrievalSearchForm {
  collection_ids: string
  include_inactive: boolean
  kinds: string
  limit: string
  metadata: string
  query: string
  repeats: string
  source_types: string
}

export interface RetrievalSearchPayload {
  collection_ids?: Array<string>
  include_inactive: boolean
  kinds?: Array<string>
  limit: number
  metadata?: Record<string, string>
  query: string
  source_types?: Array<string>
}

export interface RetrievalDocument {
  body?: string
  collection_id?: string
  id: string
  kind: string
  metadata?: Record<string, string>
  source_type?: string
  title: string
}

export interface RetrievalSearchHit {
  document: RetrievalDocument
  excerpt: string
  high_signal_terms: number
  matched_terms: number
  score: number
}

export function isRetrievalSearchHits(
  value: unknown,
): value is Array<RetrievalSearchHit> {
  return Array.isArray(value) && value.every(isRetrievalSearchHit)
}

function isRetrievalSearchHit(value: unknown): value is RetrievalSearchHit {
  return isRecord(value) && hasRetrievalSearchHitShape(value)
}

function hasRetrievalSearchHitShape(value: Record<string, unknown>): boolean {
  return (
    isRetrievalDocument(value.document) &&
    hasStringProps(value, ['excerpt']) &&
    hasNumberProps(value, ['high_signal_terms', 'matched_terms', 'score'])
  )
}

function isRetrievalDocument(value: unknown): value is RetrievalDocument {
  return isRecord(value) && hasRetrievalDocumentShape(value)
}

function hasRetrievalDocumentShape(value: Record<string, unknown>): boolean {
  return (
    hasStringProps(value, ['id', 'kind', 'title']) &&
    hasOptionalStringProps(value, ['body', 'collection_id', 'source_type']) &&
    optionalStringRecord(value.metadata)
  )
}

function hasOptionalStringProps(
  record: Record<string, unknown>,
  keys: Array<string>,
): boolean {
  return keys.every((key) => optionalString(record[key]))
}

function optionalString(value: unknown): boolean {
  return value === undefined || isString(value)
}

function optionalStringRecord(value: unknown): boolean {
  if (value === undefined) {
    return true
  }

  if (!isRecord(value)) {
    return false
  }

  return Object.values(value).every(isString)
}
