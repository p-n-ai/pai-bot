import type { ClassProgress } from './dashboard-types'

export interface DashboardSummary {
  attentionCount: number
  studentCount: number
  topicCount: number
  trackedScores: number
  averageMastery: number
  hasHeatmap: boolean
  coveragePercent: number
  strongestTopic: DashboardTopicSummary | null
  weakestTopic: DashboardTopicSummary | null
}

export interface DashboardTopicSummary {
  score: number
  topicID: string
}

export function getDashboardSummary(progress: ClassProgress): DashboardSummary {
  const totalSlots = progress.students.length * progress.topic_ids.length
  const scores = getScores(progress)
  const topicSummaries = getTopicSummaries(progress)

  return {
    attentionCount: getAttentionCount(progress),
    studentCount: progress.students.length,
    topicCount: progress.topic_ids.length,
    trackedScores: scores.length,
    averageMastery: getAveragePercent(scores),
    hasHeatmap: totalSlots > 0,
    coveragePercent: getCoveragePercent(scores.length, totalSlots),
    strongestTopic: topicSummaries.at(-1) ?? null,
    weakestTopic: topicSummaries[0] ?? null,
  }
}

function getScores(progress: ClassProgress): Array<number> {
  return progress.students.flatMap((student) =>
    getScoresForTopics(student.topics, progress.topic_ids),
  )
}

function getAveragePercent(scores: Array<number>): number {
  if (scores.length === 0) {
    return 0
  }

  const total = scores.reduce((sum, score) => sum + score, 0)

  return Math.round((total / scores.length) * 100)
}

function getTopicSummaries(
  progress: ClassProgress,
): Array<DashboardTopicSummary> {
  return progress.topic_ids
    .map((topicID) => getTopicSummary(progress, topicID))
    .filter((topic): topic is DashboardTopicSummary => topic !== null)
    .reduce<Array<DashboardTopicSummary>>(insertTopicByScore, [])
}

function getTopicSummary(
  progress: ClassProgress,
  topicID: string,
): DashboardTopicSummary | null {
  const scores = progress.students.flatMap((student) =>
    getScoreValue(student.topics[topicID]),
  )

  return scores.length > 0
    ? { score: getAveragePercent(scores), topicID }
    : null
}

function getStudentAverageScores(progress: ClassProgress): Array<number> {
  return progress.students.flatMap((student) => {
    const scores = getScoresForTopics(student.topics, progress.topic_ids)

    return scores.length > 0 ? [getAveragePercent(scores)] : []
  })
}

function getScoresForTopics(
  topics: Record<string, number>,
  topicIDs: Array<string>,
): Array<number> {
  return topicIDs.flatMap((topicID) => getScoreValue(topics[topicID]))
}

function getScoreValue(score: number | undefined): Array<number> {
  return typeof score === 'number' ? [score] : []
}

function getCoveragePercent(trackedScores: number, totalSlots: number): number {
  return totalSlots > 0 ? Math.round((trackedScores / totalSlots) * 100) : 0
}

function getAttentionCount(progress: ClassProgress): number {
  return getStudentAverageScores(progress).filter((score) => score < 50).length
}

function insertTopicByScore(
  sortedTopics: Array<DashboardTopicSummary>,
  topic: DashboardTopicSummary,
): Array<DashboardTopicSummary> {
  const insertIndex = sortedTopics.findIndex((item) => topic.score < item.score)

  if (insertIndex === -1) {
    return [...sortedTopics, topic]
  }

  return [
    ...sortedTopics.slice(0, insertIndex),
    topic,
    ...sortedTopics.slice(insertIndex),
  ]
}
