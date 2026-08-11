import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { AISettingsPanel } from '@/components/settings/ai-settings-panel'

export const Route = createFileRoute('/_authenticated/settings/ai')({
  component: AISettingsRoute,
})

function AISettingsRoute() {
  return (
    <AdminPageSection
      description='Choose providers, models, credentials, and AI features for every school.'
      title='AI settings'
    >
      <AISettingsPanel />
    </AdminPageSection>
  )
}
