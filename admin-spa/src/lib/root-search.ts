import { Option, Schema, flow } from 'effect'

import { isSafeRedirectPath } from './rbac'

export interface RootSearch {
  next?: string
}

const RootSearchSchema = Schema.Struct({
  next: Schema.optionalKey(Schema.String),
})

function normalizeRootSearch(search: typeof RootSearchSchema.Type): RootSearch {
  return isSafeRedirectPath(search.next) ? { next: search.next } : {}
}

/** Parses safe root-route search parameters. */
export const parseRootSearch = flow(
  Schema.decodeUnknownOption(RootSearchSchema),
  Option.map(normalizeRootSearch),
  Option.getOrElse((): RootSearch => ({})),
)
