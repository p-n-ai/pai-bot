import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { EmbedConfigPanel } from '@/components/settings/embed-config-panel'

export const Route = createFileRoute('/_authenticated/settings/embed')({
  component: EmbedSettingsRoute,
})

function EmbedSettingsRoute() {
  return (
    <AdminPageSection
      description='Configure, approve hosts, and install the school chat widget.'
      eyebrow='Technical tools'
      title='Website chat'
    >
      <EmbedConfigPanel />
    </AdminPageSection>
  )
}
