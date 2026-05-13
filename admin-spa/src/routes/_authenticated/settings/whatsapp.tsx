import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/admin-page-section'
import { WhatsAppSetupPanel } from '@/components/whatsapp-setup-panel'

export const Route = createFileRoute('/_authenticated/settings/whatsapp')({
  component: WhatsAppSettingsRoute,
})

function WhatsAppSettingsRoute() {
  return (
    <AdminPageSection
      description='Link a WhatsApp account, check QR setup state, and disconnect an active whatsmeow session.'
      eyebrow='Integration'
      title='WhatsApp setup'
    >
      <WhatsAppSetupPanel />
    </AdminPageSection>
  )
}
