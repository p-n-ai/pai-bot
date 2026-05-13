import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { ExportPanel } from '@/components/export/export-panel'

export const Route = createFileRoute('/_authenticated/export')({
  component: ExportRoute,
})

function ExportRoute() {
  return (
    <AdminPageSection
      description='Download tenant-scoped student, conversation, and progress datasets for reporting, migration, and audit workflows.'
      eyebrow='Administration'
      title='Data export'
    >
      <ExportPanel />
    </AdminPageSection>
  )
}
