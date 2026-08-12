import { ParentSummaryPanel } from './parent-summary-panel'
import { StudentDetailPanel } from './student-detail-panel'
import type { ReactNode } from 'react'
import { AdminPageSection } from '@/components/shared/admin-page-section'

interface EntityDetailRouteSectionProps {
  children: ReactNode
  description: string
  title: string
}

function EntityDetailRouteSection({
  children,
  description,
  title,
}: EntityDetailRouteSectionProps) {
  return (
    <AdminPageSection description={description} title={title}>
      {children}
    </AdminPageSection>
  )
}

export function ParentSummaryRouteSection({ parentID }: { parentID: string }) {
  return (
    <EntityDetailRouteSection
      description='Weekly learner activity, mastery, and suggested encouragement for home support.'
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
      title='Student summary'
    >
      <StudentDetailPanel studentID={studentID} />
    </EntityDetailRouteSection>
  )
}
