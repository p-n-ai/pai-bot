import { LoadingStatus, StatePanel } from './state-panel'

export function LoadState({
  error,
  errorTitle,
  loadingTitle,
  status,
}: {
  error: string | null
  errorTitle: string
  loadingTitle: string
  status: 'loading' | 'ready' | 'error'
}) {
  const views = {
    error: (
      <StatePanel role='alert' title={errorTitle}>
        {error ?? 'Request failed'}
      </StatePanel>
    ),
    loading: <LoadingStatus>{loadingTitle}</LoadingStatus>,
    ready: null,
  }

  return views[status]
}
