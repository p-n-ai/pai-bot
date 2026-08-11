import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { EmbedConfigPanel } from '@/components/settings/embed-config-panel'

export const Route = createFileRoute('/_authenticated/settings/embed')({
  component: EmbedSettingsRoute,
})

function EmbedSettingsRoute() {
  return (
    <AdminPageSection
      description='Choose the appearance, approve websites, and install school chat.'
      eyebrow='Technical tools'
      title='Website chat'
    >
      <EmbedConfigPanel />
    </AdminPageSection>
  )
}
