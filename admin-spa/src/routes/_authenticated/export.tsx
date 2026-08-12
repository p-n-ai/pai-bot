import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { ExportPanel } from '@/components/export/export-panel'

export const Route = createFileRoute('/_authenticated/export')({
  component: ExportRoute,
})

function ExportRoute() {
  return (
    <AdminPageSection
      description='Download school records for reporting, migration, or audit work.'
      title='Download records'
    >
      <ExportPanel />
    </AdminPageSection>
  )
}
