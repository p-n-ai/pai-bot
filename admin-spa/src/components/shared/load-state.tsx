import { LoadingStatus, StatePanel } from './state-panel'
import { ContentSkeleton } from './content-skeleton'
import type { ContentSkeletonVariant } from './content-skeleton'

export function LoadState({
  error,
  errorTitle,
  loadingTitle,
  loadingVariant,
  status,
}: {
  error: string | null
  errorTitle: string
  loadingTitle: string
  loadingVariant?: ContentSkeletonVariant
  status: 'loading' | 'ready' | 'error'
}) {
  const views = {
    error: (
      <StatePanel role='alert' title={errorTitle}>
        {error ?? 'Request failed'}
      </StatePanel>
    ),
    loading: loadingVariant ? (
      <ContentSkeleton label={loadingTitle} variant={loadingVariant} />
    ) : (
      <LoadingStatus>{loadingTitle}</LoadingStatus>
    ),
    ready: null,
  }

  return views[status]
}
