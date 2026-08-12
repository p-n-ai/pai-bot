const buildAIPages = [
  'overview',
  'character',
  'curriculum',
  'teaching',
  'test',
  'publish',
  'activity',
] as const

export type BuildAIPageKey = (typeof buildAIPages)[number]

export interface BuildAISearch {
  readonly page: BuildAIPageKey
}

/** Parses Build AI query state with Overview as the safe destination. */
export function parseBuildAISearch(
  search: Record<string, unknown>,
): BuildAISearch {
  const page = buildAIPages.find((candidate) => candidate === search.page)
  return { page: page ?? 'overview' }
}
