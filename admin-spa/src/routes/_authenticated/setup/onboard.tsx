import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { OnboardingPanel } from '@/components/onboarding/onboarding-panel'

export const Route = createFileRoute('/_authenticated/setup/onboard')({
  component: OnboardingRoute,
})

function OnboardingRoute() {
  return (
    <AdminPageSection
      description='Choose a syllabus, create your first class, and set up its tutor.'
      eyebrow='Setup'
      title='Set up your first class'
    >
      <OnboardingPanel />
    </AdminPageSection>
  )
}
