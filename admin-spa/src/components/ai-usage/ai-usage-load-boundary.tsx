import { useCallback } from 'react'
import type { ReactNode } from 'react'

import type { AIUsageLoadState } from '@/hooks/use-ai-usage-state'
import type { AIUsageSummary } from '@/lib/ai-usage-types'
import type { ContentSkeletonVariant } from '@/components/shared/content-skeleton'
import { LoadState } from '@/components/shared/load-state'

export function AIUsageLoadBoundary({
  children,
  errorTitle,
  loadingTitle,
  loadingVariant = 'analytics',
  setState,
  state,
}: {
  children: (args: {
    onUsageSaved: (usage: AIUsageSummary) => void
    usage: AIUsageSummary
  }) => ReactNode
  errorTitle: string
  loadingTitle: string
  loadingVariant?: ContentSkeletonVariant
  setState: (state: AIUsageLoadState) => void
  state: AIUsageLoadState
}) {
  const handleUsageSaved = useCallback(
    (usage: AIUsageSummary) => {
      setState({ status: 'ready', usage, error: null })
    },
    [setState],
  )

  if (state.status !== 'ready') {
    return (
      <LoadState
        error={state.error}
        errorTitle={errorTitle}
        loadingTitle={loadingTitle}
        loadingVariant={loadingVariant}
        status={state.status}
      />
    )
  }

  return children({ onUsageSaved: handleUsageSaved, usage: state.usage })
}
