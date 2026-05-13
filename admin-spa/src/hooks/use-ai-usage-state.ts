import { useEffect, useState } from 'react'

import type { AIUsageSummary } from '@/lib/ai-usage-types'
import { getAIUsage } from '@/lib/admin-api'

export type AIUsageLoadState =
  | { status: 'loading'; usage: null; error: null }
  | { status: 'ready'; usage: AIUsageSummary; error: null }
  | { status: 'error'; usage: null; error: string }

export function useAIUsageState(fallbackError: string) {
  const [state, setState] = useState<AIUsageLoadState>({
    status: 'loading',
    usage: null,
    error: null,
  })

  useEffect(() => {
    let active = true

    getAIUsage()
      .then((usage) => {
        if (!active) {
          return
        }

        setState({ status: 'ready', usage, error: null })
      })
      .catch((caught: unknown) => {
        if (!active) {
          return
        }

        setState({
          status: 'error',
          usage: null,
          error: caught instanceof Error ? caught.message : fallbackError,
        })
      })

    return () => {
      active = false
    }
  }, [fallbackError])

  return [state, setState] as const
}
