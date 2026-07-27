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
  activeDays: number
  conversations: Array<StudentConversation>
  hasConversations: boolean
  hasProgress: boolean
  latestActivityAt: string | null
  recentMessageCount: number
  recommendation: {
    description: string
    title: string
  }
  radarData: Array<{
    mastery: number
    topic: string
  }>
  status: {
    description: string
    label: string
    tone: 'attention' | 'neutral' | 'positive'
  }
  strengthAreas: StudentDetail['progress']
  struggleAreas: StudentDetail['progress']
}

export function buildStudentViewModel(
  detail: StudentDetail,
  conversations: Array<StudentConversation>,
  currentDate = new Date(),
): StudentViewModel {
  const activityGrid = buildStudentActivityGrid(conversations, currentDate)
  const recentMessageCount = activityGrid.reduce(
    (total, item) => total + item.count,
    0,
  )
  const struggleAreas = getStruggleAreas(detail)
  const strengthAreas = getStrengthAreas(detail)

  return {
    activityGrid,
    activeDays: activityGrid.filter((item) => item.count > 0).length,
    conversations: getConversationsNewestFirst(conversations),
    hasConversations: conversations.length > 0,
    hasProgress: detail.progress.length > 0,
    latestActivityAt: getLatestConversationTimestamp(conversations),
    recentMessageCount,
    recommendation: getRecommendation(detail, struggleAreas),
    radarData: detail.progress.map((item) => ({
      mastery: Math.round(item.mastery_score * 100),
      topic: formatTopicLabel(item.topic_id),
    })),
    status: getLearnerStatus(detail, recentMessageCount, struggleAreas),
    strengthAreas,
    struggleAreas,
  }
}

function getStruggleAreas(detail: StudentDetail): StudentDetail['progress'] {
  return detail.progress.filter((item) => item.mastery_score < masteryThreshold)
}

function getStrengthAreas(detail: StudentDetail): StudentDetail['progress'] {
  return detail.progress.filter(
    (item) => item.mastery_score >= masteryThreshold,
  )
}

function getLearnerStatus(
  detail: StudentDetail,
  recentMessageCount: number,
  struggleAreas: StudentDetail['progress'],
): StudentViewModel['status'] {
  if (struggleAreas.length > 0) {
    return {
      description: `${struggleAreas.length} ${struggleAreas.length === 1 ? 'topic needs' : 'topics need'} targeted support.`,
      label: 'Needs attention',
      tone: 'attention',
    }
  }

  if (detail.progress.length > 0) {
    return {
      description:
        recentMessageCount > 0
          ? 'Recent learning activity has no topics below the support threshold.'
          : 'Mastery is developing, but no recent tutoring activity is recorded.',
      label: recentMessageCount > 0 ? 'On track' : 'Check in',
      tone: recentMessageCount > 0 ? 'positive' : 'neutral',
    }
  }

  return {
    description:
      'There is not enough progress data to assess this learner yet.',
    label: 'Getting started',
    tone: 'neutral',
  }
}

function getRecommendation(
  detail: StudentDetail,
  struggleAreas: StudentDetail['progress'],
): StudentViewModel['recommendation'] {
  const priorityTopic = getLowestMasteryArea(struggleAreas)

  if (priorityTopic) {
    const mastery = Math.round(priorityTopic.mastery_score * 100)
    const topic = formatTopicLabel(priorityTopic.topic_id)

    return {
      description: `${topic} is the lowest-mastery topic at ${mastery}%. Use a short worked example, then check understanding.`,
      title: `Review ${topic} next`,
    }
  }

  const strongestTopic = getHighestMasteryArea(getStrengthAreas(detail))
  if (strongestTopic) {
    const topic = formatTopicLabel(strongestTopic.topic_id)

    return {
      description: `${topic} is currently secure. Reinforce it with a mixed-practice question before introducing the next topic.`,
      title: `Build from ${topic}`,
    }
  }

  return {
    description:
      'Ask the learner to complete a short diagnostic conversation so strengths and support needs can be identified.',
    title: 'Start with a guided check-in',
  }
}

function getLowestMasteryArea(
  progress: StudentDetail['progress'],
): StudentDetail['progress'][number] | undefined {
  return progress.reduce<StudentDetail['progress'][number] | undefined>(
    (lowest, item) =>
      !lowest || item.mastery_score < lowest.mastery_score ? item : lowest,
    undefined,
  )
}

function getHighestMasteryArea(
  progress: StudentDetail['progress'],
): StudentDetail['progress'][number] | undefined {
  return progress.reduce<StudentDetail['progress'][number] | undefined>(
    (highest, item) =>
      !highest || item.mastery_score > highest.mastery_score ? item : highest,
    undefined,
  )
}

function getLatestConversationTimestamp(
  conversations: Array<StudentConversation>,
): string | null {
  let latestTimestamp: string | null = null
  let latestTime = Number.NEGATIVE_INFINITY

  conversations.forEach((conversation) => {
    const timestamp = new Date(conversation.timestamp).getTime()

    if (!Number.isNaN(timestamp) && timestamp > latestTime) {
      latestTimestamp = conversation.timestamp
      latestTime = timestamp
    }
  })

  return latestTimestamp
}

function getConversationsNewestFirst(
  conversations: Array<StudentConversation>,
): Array<StudentConversation> {
  const orderedConversations = [...conversations]

  // oxlint-disable-next-line unicorn/no-array-sort -- ES2022 does not provide Array.prototype.toSorted.
  return orderedConversations.sort(
    (left, right) =>
      getConversationTime(right.timestamp) -
      getConversationTime(left.timestamp),
  )
}

function getConversationTime(timestamp: string): number {
  const time = new Date(timestamp).getTime()

  return Number.isNaN(time) ? Number.MIN_SAFE_INTEGER : time
}

export function getActivityTone(level: number): string {
  return activityTones[Math.min(Math.max(level, 0), activityTones.length - 1)]
}

function buildStudentActivityGrid(
  conversations: Array<StudentConversation>,
  currentDate: Date,
): Array<StudentActivityGridItem> {
  const timestamps = conversations.map((item) => toISODate(item.timestamp))
  const anchorDate = currentDate.toISOString().slice(0, 10)
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
  'bg-slate-200 text-slate-700 dark:bg-slate-800 dark:text-slate-100',
  'bg-sky-200 text-sky-950 dark:bg-sky-700 dark:text-white',
  'bg-sky-300 text-sky-950 dark:bg-sky-500 dark:text-slate-950',
  'bg-sky-500 text-white dark:bg-sky-400 dark:text-slate-950',
  'bg-sky-600 text-white dark:bg-sky-300 dark:text-slate-950',
] as const

const activityLevelThresholds = [
  { count: 6, level: 4 },
  { count: 4, level: 3 },
  { count: 2, level: 2 },
  { count: 1, level: 1 },
] as const

const activityWindowDays = 14
const masteryThreshold = 0.6
