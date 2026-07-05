import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { AISettingsPanel } from '@/components/settings/ai-settings-panel'

export const Route = createFileRoute('/_authenticated/settings/ai')({
  component: AISettingsRoute,
})

function AISettingsRoute() {
  return (
    <AdminPageSection
      description='Pick the default AI provider, tune the OpenRouter model and key, and switch platform-wide feature flags. These settings apply to every tenant.'
      eyebrow='School admin'
      title='AI settings'
    >
      <AISettingsPanel />
    </AdminPageSection>
  )
}
