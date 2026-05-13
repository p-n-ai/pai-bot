import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { OnboardingPanel } from '@/components/onboarding/onboarding-panel'

export const Route = createFileRoute('/_authenticated/setup/onboard')({
  component: OnboardingRoute,
})

function OnboardingRoute() {
  return (
    <AdminPageSection
      description='Create the first class and persist the starting classroom setup.'
      eyebrow='Setup'
      title='Set up your first class'
    >
      <OnboardingPanel />
    </AdminPageSection>
  )
}
