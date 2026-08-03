import type { LeaderboardEntry } from './leaderboard-types'

export interface LeaderboardRowView {
  readonly studentID: string
  readonly studentName: string
  readonly rankLabel: string
  readonly gainLabel: string
  readonly gainTone: 'positive' | 'neutral' | 'negative'
}

/** Projects one server-ranked entry into display-only leaderboard values. */
export function getLeaderboardRowView(
  entry: LeaderboardEntry,
): LeaderboardRowView {
  const points = Math.round(entry.mastery_gain * 100)
  return {
    studentID: entry.user_id,
    studentName: entry.user_name,
    rankLabel: `#${entry.rank}`,
    gainLabel: points > 0 ? `+${points} pts` : `${points} pts`,
    gainTone: points > 0 ? 'positive' : points < 0 ? 'negative' : 'neutral',
  }
}
