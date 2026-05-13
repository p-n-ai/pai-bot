import { readNextPath } from './redirect-search'

export interface RootSearch {
  next?: string
}

export function parseRootSearch(search: Record<string, unknown>): RootSearch {
  return {
    next: readNextPath(search.next),
  }
}
