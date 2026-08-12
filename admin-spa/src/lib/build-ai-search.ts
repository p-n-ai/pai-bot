import { Option, Schema, flow } from 'effect'

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

const BuildAISearchSchema = Schema.Struct({
  page: Schema.optionalKey(Schema.Literals(buildAIPages)),
})

function normalizeBuildAISearch(
  search: typeof BuildAISearchSchema.Type,
): BuildAISearch {
  return { page: search.page ?? 'overview' }
}

/** Parses Build AI query state with Overview as the safe destination. */
export const parseBuildAISearch = flow(
  Schema.decodeUnknownOption(BuildAISearchSchema),
  Option.map(normalizeBuildAISearch),
  Option.getOrElse((): BuildAISearch => ({ page: 'overview' })),
)
