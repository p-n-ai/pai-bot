import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/admin-page-section'
import { EmbedConfigPanel } from '@/components/embed-config-panel'

export const Route = createFileRoute('/_authenticated/settings/embed')({
  component: EmbedSettingsRoute,
})

function EmbedSettingsRoute() {
  return (
    <AdminPageSection
      description='Enable the embeddable chat widget and keep host origins explicit for the current tenant.'
      eyebrow='School admin'
      title='Embed widget'
    >
      <EmbedConfigPanel />
    </AdminPageSection>
  )
}
