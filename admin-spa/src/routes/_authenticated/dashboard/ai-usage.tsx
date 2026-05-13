import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'

import type { AIUsageSummary } from '@/lib/ai-usage-types'
import { AdminPageSection } from '@/components/admin-page-section'
import { AIUsageBudgetSection } from '@/components/ai-usage-budget-section'
import { AIUsageDailyTrend } from '@/components/ai-usage-daily-trend'
import { AIUsageLoadBoundary } from '@/components/ai-usage-load-boundary'
import { AIUsageOverviewStats } from '@/components/ai-usage-overview-stats'
import { AIUsageProviderBreakdown } from '@/components/ai-usage-provider-breakdown'
import { useAuth } from '@/auth-provider'
import { useAIUsageState } from '@/hooks/use-ai-usage-state'
import { getAIUsageView } from '@/lib/ai-usage-view'

export const Route = createFileRoute('/_authenticated/dashboard/ai-usage')({
  component: AIUsageRoute,
})

function AIUsageRoute() {
  const { auth } = useAuth()
  const [state, setState] = useAIUsageState('AI usage failed')

  return (
    <AdminPageSection
      description='Token volume, budget status, and provider mix for the current school workspace.'
      eyebrow='AI operations'
      title='AI usage'
    >
      <AIUsageLoadBoundary
        errorTitle='Unable to load AI usage'
        loadingTitle='Loading AI usage...'
        setState={setState}
        state={state}
      >
        {({ onUsageSaved, usage }) => (
          <AIUsageReady
            canManageBudget={auth.session?.user.role === 'admin'}
            onUsageSaved={onUsageSaved}
            usage={usage}
          />
        )}
      </AIUsageLoadBoundary>
    </AdminPageSection>
  )
}

function AIUsageReady({
  canManageBudget,
  onUsageSaved,
  usage,
}: {
  canManageBudget: boolean
  onUsageSaved: (usage: AIUsageSummary) => void
  usage: AIUsageSummary
}) {
  const view = getAIUsageView(usage)
  const handleSavedUsage = useCallback(
    (nextUsage: AIUsageSummary) => {
      onUsageSaved(nextUsage)
    },
    [onUsageSaved],
  )

  return (
    <div className='dashboard-stack'>
      <AIUsageOverviewStats view={view} />
      <AIUsageBudgetSection
        canManageBudget={canManageBudget}
        onSaved={handleSavedUsage}
        usage={usage}
        view={view}
      />
      <AIUsageDailyTrend view={view} />
      <AIUsageProviderBreakdown view={view} />
    </div>
  )
}
