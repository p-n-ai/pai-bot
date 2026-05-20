import { isRetrievalSearchHits } from './retrieval-lab-types'
import { isRecord, isString } from './type-guards'
import type {
  RetrievalSearchForm,
  RetrievalSearchHit,
  RetrievalSearchPayload,
} from './retrieval-lab-types'

export const defaultRetrievalSearchForm: RetrievalSearchForm = {
  collection_ids: '',
  include_inactive: false,
  kinds: '',
  limit: '5',
  metadata: '',
  query: 'linear equation',
  repeats: '3',
  source_types: '',
}

export const exampleRetrievalSearchForm: RetrievalSearchForm = {
  collection_ids: 'curriculum:matematik-form-1',
  include_inactive: false,
  kinds: 'topic_card,teaching_note',
  limit: '5',
  metadata: '{"form":"1"}',
  query: 'first step only for linear equations',
  repeats: '3',
  source_types: 'curriculum',
}

class RetrievalSearchError extends Error {
  constructor(
    message: string,
    readonly status: number | null,
    readonly raw: string,
  ) {
    super(message)
    this.name = 'RetrievalSearchError'
  }
}

// fallow-ignore-next-line complexity
export function buildRetrievalSearchPayload(
  form: RetrievalSearchForm,
): RetrievalSearchPayload {
  const metadata = parseMetadata(form.metadata)
  const collectionIDs = parseCSV(form.collection_ids)
  const kinds = parseCSV(form.kinds)
  const sourceTypes = parseCSV(form.source_types)

  return {
    collection_ids: collectionIDs.length > 0 ? collectionIDs : undefined,
    include_inactive: form.include_inactive,
    kinds: kinds.length > 0 ? kinds : undefined,
    limit: Number(form.limit) || 5,
    metadata,
    query: form.query.trim(),
    source_types: sourceTypes.length > 0 ? sourceTypes : undefined,
  }
}

export function getRetrievalRunCount(repeats: string): number {
  return Math.max(1, Number(repeats) || 1)
}

export function isExpiredRetrievalSession(message: string): boolean {
  return (
    message.includes('401 Unauthorized: expired token') ||
    message.includes('401 Unauthorized: missing bearer token')
  )
}

// fallow-ignore-next-line complexity
export function normalizeRetrievalError(message: string): string {
  if (message.includes('404 Not Found: 404 page not found')) {
    return 'Retrieval search endpoint unavailable. Restart the local backend and try again.'
  }

  if (message.includes('401 Unauthorized: expired token')) {
    return 'Your admin session expired. Sign in again to keep testing retrieval.'
  }

  if (message.includes('401 Unauthorized: missing bearer token')) {
    return 'You are signed out. Sign in again to use retrieval search.'
  }

  return message || 'Retrieval search failed.'
}

// fallow-ignore-next-line complexity
export async function searchRetrieval(
  payload: RetrievalSearchPayload,
  fetcher: typeof fetch = fetch,
): Promise<Array<RetrievalSearchHit>> {
  const response = await fetcher('/api/admin/retrieval/search', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    cache: 'no-store',
    body: JSON.stringify(payload),
  })
  const text = await response.text()

  if (!response.ok) {
    throw new RetrievalSearchError(
      `${response.status} ${response.statusText}: ${text}`,
      response.status,
      text,
    )
  }

  const parsed: unknown = text.trim() ? JSON.parse(text) : []

  if (!isRetrievalSearchHits(parsed)) {
    throw new RetrievalSearchError(
      'Invalid retrieval search response',
      response.status,
      text,
    )
  }

  return parsed
}

function parseCSV(value: string): Array<string> {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function parseMetadata(value: string): Record<string, string> | undefined {
  if (!value.trim()) {
    return undefined
  }

  const parsed: unknown = JSON.parse(value)

  if (!isStringRecord(parsed)) {
    throw new Error('Metadata must be a JSON object with string values')
  }

  return parsed
}

function isStringRecord(value: unknown): value is Record<string, string> {
  return isRecord(value) && Object.values(value).every(isString)
}
