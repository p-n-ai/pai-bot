import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/admin-page-section'
import { AIUsageBudgetSection } from '@/components/ai-usage-budget-section'
import { AIUsageLoadBoundary } from '@/components/ai-usage-load-boundary'
import { useAuth } from '@/auth-provider'
import { useAIUsageState } from '@/hooks/use-ai-usage-state'
import { getAIUsageView } from '@/lib/ai-usage-view'

export const Route = createFileRoute('/_authenticated/settings/budget')({
  component: BudgetSettingsRoute,
})

function BudgetSettingsRoute() {
  const { auth } = useAuth()
  const [state, setState] = useAIUsageState('Token budget failed')

  return (
    <AdminPageSection
      description='Set the token allowance window for the current school workspace and review the remaining budget before rollout decisions.'
      eyebrow='School admin'
      title='Token budget'
    >
      <AIUsageLoadBoundary
        errorTitle='Unable to load token budget'
        loadingTitle='Loading token budget...'
        setState={setState}
        state={state}
      >
        {({ onUsageSaved, usage }) => (
          <AIUsageBudgetSection
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
