import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { AIUsageBudgetPage } from '@/components/ai-usage/ai-usage-budget-page'
import { AIUsageLoadBoundary } from '@/components/ai-usage/ai-usage-load-boundary'
import { useAuth } from '@/auth-provider'
import { useAIUsageState } from '@/hooks/use-ai-usage-state'
import { getAIUsageView } from '@/lib/ai-usage-view'

export const Route = createFileRoute('/_authenticated/settings/budget')({
  component: BudgetSettingsRoute,
})

function BudgetSettingsRoute() {
  const { auth } = useAuth()
  const [state, setState] = useAIUsageState(
    'Unable to load the AI budget. Check your connection and try again.',
  )

  return (
    <AdminPageSection
      description='Control the school-wide token allowance and the dates it applies.'
      eyebrow='School administration'
      title='AI budget'
    >
      <AIUsageLoadBoundary
        errorTitle='Unable to load token budget'
        loadingTitle='Loading AI budget…'
        loadingVariant='budget'
        setState={setState}
        state={state}
      >
        {({ onUsageSaved, usage }) => (
          <AIUsageBudgetPage
            canManageBudget={
              auth.session?.user.role === 'admin' ||
              auth.session?.user.role === 'platform_admin'
            }
            onSaved={onUsageSaved}
            usage={usage}
            view={getAIUsageView(usage)}
          />
        )}
      </AIUsageLoadBoundary>
    </AdminPageSection>
  )
}
