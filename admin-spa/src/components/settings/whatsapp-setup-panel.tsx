import { useCallback, useEffect, useState } from 'react'

import type { WhatsAppStatus } from '@/lib/whatsapp-types'
import { AuthErrorAlert } from '@/components/shared/auth-error-alert'
import { Button } from '@/components/ui/button'
import { LoadState } from '@/components/shared/load-state'
import { StatePanel } from '@/components/shared/state-panel'
import { disconnectWhatsApp, getWhatsAppStatus } from '@/lib/admin-api'
import { useSubmitStatus } from '@/hooks/use-submit-status'

type WhatsAppLoadState = 'loading' | 'ready' | 'error'

export function WhatsAppSetupPanel() {
  const statusState = useWhatsAppStatus()
  const disconnect = useWhatsAppDisconnect(statusState.refreshStatus)

  return (
    <WhatsAppStatusView disconnect={disconnect} statusState={statusState} />
  )
}

function useWhatsAppStatus() {
  const [status, setStatus] = useState<WhatsAppStatus | null>(null)
  const [loadState, setLoadState] = useState<WhatsAppLoadState>('loading')
  const [loadError, setLoadError] = useState('')
  const refreshStatus = useCallback(() => {
    getWhatsAppStatus()
      .then((payload) => {
        setStatus(payload)
        setLoadState('ready')
        setLoadError('')
      })
      .catch((caught: unknown) => {
        setLoadState('error')
        setLoadError(
          caught instanceof Error
            ? caught.message
            : 'Check that WhatsApp is enabled on the server.',
        )
      })
  }, [])

  useEffect(() => {
    refreshStatus()
    const interval = window.setInterval(refreshStatus, 5000)

    return () => window.clearInterval(interval)
  }, [refreshStatus])

  return { loadError, loadState, refreshStatus, status }
}

function useWhatsAppDisconnect(refreshStatus: () => void) {
  const disconnect = useSubmitStatus('')
  const handleDisconnect = useCallback(() => {
    disconnect.beginSubmit()
    disconnectWhatsApp()
      .then(refreshStatus)
      .catch((caught: unknown) => {
        disconnect.setError(
          caught instanceof Error ? caught.message : 'Failed to disconnect',
        )
      })
      .finally(disconnect.finishSubmit)
  }, [disconnect, refreshStatus])

  return { ...disconnect, handleDisconnect }
}

function WhatsAppStatusView({
  disconnect,
  statusState,
}: {
  disconnect: ReturnType<typeof useWhatsAppDisconnect>
  statusState: ReturnType<typeof useWhatsAppStatus>
}) {
  if (statusState.loadState !== 'ready') {
    return (
      <LoadState
        error={statusState.loadError}
        errorTitle='Could not load WhatsApp status'
        loadingTitle='Loading WhatsApp status'
        status={statusState.loadState}
      />
    )
  }

  return (
    <WhatsAppReady
      disconnect={disconnect}
      handleRetry={statusState.refreshStatus}
      status={statusState.status}
    />
  )
}

function WhatsAppReady({
  disconnect,
  handleRetry,
  status,
}: {
  disconnect: ReturnType<typeof useWhatsAppDisconnect>
  handleRetry: () => void
  status: WhatsAppStatus | null
}) {
  if (!status) {
    return (
      <StatePanel title='WhatsApp status unavailable'>
        Check that WhatsApp is enabled on the server.
      </StatePanel>
    )
  }

  if (status.connected) {
    return (
      <WhatsAppConnected
        error={disconnect.error}
        isPending={disconnect.isPending}
        onDisconnect={disconnect.handleDisconnect}
      />
    )
  }

  return <WhatsAppQRCode handleRetry={handleRetry} status={status} />
}

function WhatsAppConnected({
  error,
  isPending,
  onDisconnect,
}: {
  error: string
  isPending: boolean
  onDisconnect: () => void
}) {
  return (
    <section
      aria-label='WhatsApp connected'
      className='grid gap-6 rounded-lg border border-border bg-card p-6'
    >
      <header>
        <h2 className='m-0 text-2xl leading-tight text-foreground'>
          WhatsApp connected
        </h2>
        <p className='mt-2 mb-0 text-muted-foreground'>
          Your WhatsApp account is linked and the bot is active.
        </p>
      </header>
      <div
        aria-label='WhatsApp active session'
        className='flex flex-wrap items-center justify-between gap-4 rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-emerald-950'
      >
        <span aria-hidden='true' className='text-2xl leading-none'>
          &#9989;
        </span>
        <div className='min-w-0 flex-1'>
          <strong>Session active</strong>
          <p className='mt-1 mb-0 text-sm'>
            Messages are being sent and received via WhatsApp.
          </p>
        </div>
        <Button disabled={isPending} onClick={onDisconnect} type='button'>
          {isPending ? 'Disconnecting...' : 'Disconnect'}
        </Button>
      </div>
      <AuthErrorAlert message={error} title='Disconnect failed.' />
    </section>
  )
}

function WhatsAppQRCode({
  handleRetry,
  status,
}: {
  handleRetry: () => void
  status: WhatsAppStatus
}) {
  if (!status.qr_image) {
    return (
      <section
        aria-label='Waiting for QR code'
        className='grid justify-items-center gap-4 p-8 text-center text-muted-foreground'
      >
        <span
          aria-hidden='true'
          className='size-8 animate-spin rounded-full border-4 border-muted border-t-foreground'
        />
        <p className='m-0 text-sm'>Waiting for QR code from server...</p>
        <Button onClick={handleRetry} type='button' variant='outline'>
          Retry
        </Button>
      </section>
    )
  }

  return (
    <section className='grid justify-items-center gap-5'>
      <img
        alt='WhatsApp QR Code'
        className='aspect-square w-full max-w-64 rounded-lg border border-border'
        src={status.qr_image}
      />
      <StatePanel title='Link WhatsApp'>
        <p>Scan the QR code below with your phone to connect.</p>
        <p>
          Open WhatsApp on your phone, then go to Settings, Linked Devices, and
          Link a Device.
        </p>
        <p>Page refreshes automatically every 5 seconds.</p>
      </StatePanel>
    </section>
  )
}
