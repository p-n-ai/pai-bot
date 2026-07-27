import type { ClassProgress, ClassProgressStudent } from './dashboard-types'

export type LearnerProgressFilter =
  | 'all'
  | 'attention'
  | 'on-track'
  | 'unmeasured'

export interface DashboardLearner {
  averageMastery: number | null
  needsAttention: boolean
  nextAction: string
  status: Exclude<LearnerProgressFilter, 'all'>
  student: ClassProgressStudent
  weakestTopicID: string | null
}

export function getDashboardLearners(
  progress: ClassProgress,
  query = '',
  filter: LearnerProgressFilter = 'all',
): Array<DashboardLearner> {
  const normalizedQuery = query.trim().toLocaleLowerCase()
  const learners: Array<DashboardLearner> = []

  for (const student of progress.students) {
    const learner = getDashboardLearner(progress.topic_ids, student)
    const matchesFilter = filter === 'all' || learner.status === filter
    const matchesQuery = learner.student.name
      .toLocaleLowerCase()
      .includes(normalizedQuery)

    if (matchesFilter && matchesQuery) {
      learners.push(learner)
    }
  }

  // oxlint-disable-next-line unicorn/no-array-sort -- ES2022 does not provide Array.prototype.toSorted.
  return learners.sort(compareLearners)
}

function getDashboardLearner(
  topicIDs: Array<string>,
  student: ClassProgressStudent,
): DashboardLearner {
  let scoreCount = 0
  let scoreTotal = 0
  let weakestScore = Number.POSITIVE_INFINITY
  let weakestTopicID: string | null = null

  for (const topicID of topicIDs) {
    const score = student.topics[topicID]

    if (typeof score !== 'number') {
      continue
    }

    scoreCount += 1
    scoreTotal += score

    if (score < weakestScore) {
      weakestScore = score
      weakestTopicID = topicID
    }
  }

  const averageMastery =
    scoreCount > 0 ? Math.round((scoreTotal / scoreCount) * 100) : null
  const status = getLearnerStatus(averageMastery)

  return {
    averageMastery,
    needsAttention: status === 'attention',
    nextAction: getNextAction(status, weakestTopicID),
    status,
    student,
    weakestTopicID,
  }
}

function getLearnerStatus(
  averageMastery: number | null,
): DashboardLearner['status'] {
  if (averageMastery === null) {
    return 'unmeasured'
  }

  return averageMastery < 50 ? 'attention' : 'on-track'
}

function getNextAction(
  status: DashboardLearner['status'],
  weakestTopicID: string | null,
): string {
  if (status === 'unmeasured') {
    return 'Check in after their first activity'
  }

  if (status === 'attention' && weakestTopicID) {
    return `Review ${formatTopicLabel(weakestTopicID)}`
  }

  return 'Open progress and plan the next stretch'
}

function compareLearners(
  left: DashboardLearner,
  right: DashboardLearner,
): number {
  const statusDifference =
    learnerStatusOrder[left.status] - learnerStatusOrder[right.status]

  if (statusDifference !== 0) {
    return statusDifference
  }

  if (
    left.averageMastery !== null &&
    right.averageMastery !== null &&
    left.averageMastery !== right.averageMastery
  ) {
    return left.averageMastery - right.averageMastery
  }

  return left.student.name.localeCompare(right.student.name)
}

const learnerStatusOrder: Record<DashboardLearner['status'], number> = {
  attention: 0,
  unmeasured: 1,
  'on-track': 2,
}

function formatTopicLabel(topicID: string): string {
  return topicID
    .split('-')
    .filter(Boolean)
    .map((word) => `${word[0].toUpperCase()}${word.slice(1)}`)
    .join(' ')
}
