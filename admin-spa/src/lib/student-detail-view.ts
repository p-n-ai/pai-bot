import { formatTopicLabel } from './topic-label'
import type { StudentConversation, StudentDetail } from './student-detail-types'

export { formatTopicLabel } from './topic-label'

export interface StudentActivityGridItem {
  count: number
  date: string
  level: number
  shortLabel: string
}

export interface StudentViewModel {
  activityGrid: Array<StudentActivityGridItem>
  hasConversations: boolean
  hasProgress: boolean
  radarData: Array<{
    mastery: number
    topic: string
  }>
  struggleAreas: StudentDetail['progress']
}

export function buildStudentViewModel(
  detail: StudentDetail,
  conversations: Array<StudentConversation>,
): StudentViewModel {
  return {
    activityGrid: buildStudentActivityGrid(conversations),
    hasConversations: conversations.length > 0,
    hasProgress: detail.progress.length > 0,
    radarData: detail.progress.map((item) => ({
      mastery: Math.round(item.mastery_score * 100),
      topic: formatTopicLabel(item.topic_id),
    })),
    struggleAreas: getStruggleAreas(detail),
  }
}

function getStruggleAreas(detail: StudentDetail): StudentDetail['progress'] {
  return detail.progress.filter((item) => item.mastery_score < 0.6)
}

export function getActivityTone(level: number): string {
  return activityTones[Math.min(Math.max(level, 0), activityTones.length - 1)]
}

function buildStudentActivityGrid(
  conversations: Array<StudentConversation>,
): Array<StudentActivityGridItem> {
  const timestamps = conversations.map((item) => toISODate(item.timestamp))
  const anchorDate = getLatestTimestamp(timestamps)
  const counts = countConversationDates(timestamps)

  return Array.from({ length: activityWindowDays }, (_, index) => {
    const date = addDays(anchorDate, index - (activityWindowDays - 1))
    const count = counts.get(date) ?? 0

    return {
      count,
      date,
      level: getActivityLevel(count),
      shortLabel: formatGridLabel(date),
    }
  })
}

function getLatestTimestamp(timestamps: Array<string | null>): string {
  for (let index = timestamps.length - 1; index >= 0; index -= 1) {
    const timestamp = timestamps[index]

    if (timestamp) {
      return timestamp
    }
  }

  return new Date().toISOString().slice(0, 10)
}

function getActivityLevel(count: number): number {
  return activityLevelThresholds.find((item) => count >= item.count)?.level ?? 0
}

function countConversationDates(
  timestamps: Array<string | null>,
): Map<string, number> {
  return timestamps.reduce((result, isoDate) => {
    if (isoDate) {
      result.set(isoDate, (result.get(isoDate) ?? 0) + 1)
    }

    return result
  }, new Map<string, number>())
}

function toISODate(value: string): string | null {
  const date = new Date(value)

  return Number.isNaN(date.getTime()) ? null : date.toISOString().slice(0, 10)
}

function addDays(isoDate: string, offset: number): string {
  const date = new Date(`${isoDate}T00:00:00.000Z`)
  date.setUTCDate(date.getUTCDate() + offset)
  return date.toISOString().slice(0, 10)
}

function formatGridLabel(isoDate: string): string {
  const date = new Date(`${isoDate}T00:00:00.000Z`)
  return activityDateFormatter.format(date)
}

const activityDateFormatter = new Intl.DateTimeFormat('en-US', {
  day: 'numeric',
  month: 'short',
  timeZone: 'UTC',
})

const activityTones = [
  'bg-slate-200 dark:bg-slate-800',
  'bg-sky-200 dark:bg-sky-700',
  'bg-sky-300 dark:bg-sky-500',
  'bg-sky-500 dark:bg-sky-400',
  'bg-sky-600 dark:bg-sky-300',
] as const

const activityLevelThresholds = [
  { count: 6, level: 4 },
  { count: 4, level: 3 },
  { count: 2, level: 2 },
  { count: 1, level: 1 },
] as const

const activityWindowDays = 14
