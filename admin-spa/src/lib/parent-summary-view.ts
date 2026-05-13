import { formatTopicLabel } from './topic-label'
import type { ProgressItem } from './learner-types'
import type { ParentSummary } from './parent-summary-types'

export interface ParentSummaryView {
  contextLine: string
  encouragementHeadline: string
  encouragementText: string
  hasMastery: boolean
  masteryRows: Array<ProgressItem>
}

export function getParentSummaryView(
  summary: ParentSummary | null,
): ParentSummaryView {
  const masteryRows = summary?.mastery ?? []
  const encouragement = getEncouragement(summary)

  return {
    contextLine: buildParentContextLine(summary),
    encouragementHeadline: encouragement.headline,
    encouragementText: encouragement.text,
    hasMastery: masteryRows.length > 0,
    masteryRows,
  }
}

export function formatParentTopicLabel(topicID: string): string {
  return formatTopicLabel(topicID)
}

export function getParentMasteryTone(score: number): string {
  if (score >= 0.75) {
    return 'bg-emerald-500'
  }

  if (score >= 0.5) {
    return 'bg-sky-500'
  }

  return 'bg-amber-500'
}

function buildParentContextLine(summary: ParentSummary | null): string {
  return summary ? buildLoadedContextLine(summary) : unloadedContextLine
}

function buildLoadedContextLine(summary: ParentSummary): string {
  const contact = summary.parent.email || summary.parent.name || 'Parent'
  return `${summary.child.form} | ${summary.child.channel} | Parent contact ${contact}`
}

function getEncouragement(summary: ParentSummary | null) {
  return summary?.encouragement ?? defaultEncouragement
}

const unloadedContextLine =
  'Pulling weekly activity, mastery, and encouragement from the admin API.'

const defaultEncouragement = {
  headline: 'A suggested encouragement will appear here soon.',
  text: "Once the weekly summary is ready, you'll see a short message you can send or say at home.",
}
