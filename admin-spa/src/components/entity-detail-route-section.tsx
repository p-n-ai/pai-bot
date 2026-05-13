import { AdminPageSection } from './admin-page-section'
import { ParentSummaryPanel } from './parent-summary-panel'
import { StudentDetailPanel } from './student-detail-panel'
import type { ReactNode } from 'react'

interface EntityDetailRouteSectionProps {
  children: ReactNode
  description: string
  eyebrow: string
  title: string
}

function EntityDetailRouteSection({
  children,
  description,
  eyebrow,
  title,
}: EntityDetailRouteSectionProps) {
  return (
    <AdminPageSection description={description} eyebrow={eyebrow} title={title}>
      {children}
    </AdminPageSection>
  )
}

export function ParentSummaryRouteSection({ parentID }: { parentID: string }) {
  return (
    <EntityDetailRouteSection
      description='Weekly learner activity, mastery, and suggested encouragement for home support.'
      eyebrow='Parent support summary'
      title='Child summary'
    >
      <ParentSummaryPanel parentID={parentID} />
    </EntityDetailRouteSection>
  )
}

export function StudentDetailRouteSection({
  studentID,
}: {
  studentID: string
}) {
  return (
    <EntityDetailRouteSection
      description='Learner profile, topic progress, streaks, and recent tutoring messages.'
      eyebrow='Student detail'
      title='Student summary'
    >
      <StudentDetailPanel studentID={studentID} />
    </EntityDetailRouteSection>
  )
}
