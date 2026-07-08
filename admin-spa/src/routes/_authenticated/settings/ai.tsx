import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { AISettingsPanel } from '@/components/settings/ai-settings-panel'

export const Route = createFileRoute('/_authenticated/settings/ai')({
  component: AISettingsRoute,
})

function AISettingsRoute() {
  return (
    <AdminPageSection
      description='Default provider, OpenRouter model and key, and AI feature flags. Changes apply live to every tenant — no restart.'
      eyebrow='Platform'
      title='AI settings'
    >
      <AISettingsPanel />
    </AdminPageSection>
  )
}
