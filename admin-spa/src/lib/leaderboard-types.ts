import { Option, Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const LeaderboardEntrySchema = Schema.Struct({
  user_id: Schema.String,
  user_name: Schema.String,
  mastery_gain: Schema.Number,
  rank: Schema.Number,
})

export interface LeaderboardEntry extends EffectSchema.Type<
  typeof LeaderboardEntrySchema
> {}

const decodeLeaderboardEntries = Schema.decodeUnknownOption(
  Schema.mutable(Schema.Array(LeaderboardEntrySchema)),
)

/** Decodes the server-ranked weekly class leaderboard. */
export function readLeaderboardEntries(
  value: unknown,
): ReadonlyArray<LeaderboardEntry> | null {
  return Option.getOrNull(decodeLeaderboardEntries(value))
}
