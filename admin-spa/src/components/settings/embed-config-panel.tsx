import { useCallback, useEffect, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'

import type { EmbedConfig } from '@/lib/embed-config-types'
import { AuthErrorAlert } from '@/components/shared/auth-error-alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { LoadState } from '@/components/shared/load-state'
import { StatePanel } from '@/components/shared/state-panel'
import {
  addEmbedOrigin,
  getEmbedConfig,
  removeEmbedOrigin,
  updateEmbedConfig,
} from '@/lib/admin-api'
import { useSubmitStatus } from '@/hooks/use-submit-status'

type EmbedLoadState = 'loading' | 'ready' | 'error'

export function EmbedConfigPanel() {
  const [config, setConfig] = useState<EmbedConfig | null>(null)
  const [loadState, setLoadState] = useState<EmbedLoadState>('loading')
  const [loadError, setLoadError] = useState('')
  const [origin, setOrigin] = useState('')
  const enabledSubmit = useSubmitStatus('')
  const originSubmit = useSubmitStatus('')

  const refreshConfig = useCallback(() => {
    async function loadConfig() {
      try {
        const payload = await getEmbedConfig()

        setConfig(payload)
        setLoadState('ready')
        setLoadError('')
      } catch (caught) {
        setLoadState('error')
        setLoadError(
          caught instanceof Error
            ? caught.message
            : 'Embed config could not be loaded.',
        )
      }
    }

    loadConfig().catch(() => {})
  }, [])

  useEffect(() => {
    refreshConfig()
  }, [refreshConfig])

  const handleEnabledChange = useCallback(() => {
    if (!config) {
      return
    }

    enabledSubmit.beginSubmit()
    updateEmbedConfig({
      enabled: !config.enabled,
      theme_config: config.theme_config,
    })
      .then(setConfig)
      .catch((caught: unknown) => {
        enabledSubmit.setError(
          caught instanceof Error
            ? caught.message
            : 'Embed availability could not be changed.',
        )
      })
      .finally(enabledSubmit.finishSubmit)
  }, [config, enabledSubmit])

  const handleAddOrigin = useCallback(() => {
    const nextOrigin = origin.trim()

    if (!nextOrigin) {
      originSubmit.setError('Origin is required.')
      return
    }

    originSubmit.beginSubmit()
    addEmbedOrigin(nextOrigin)
      .then(refreshConfig)
      .then(() => setOrigin(''))
      .catch((caught: unknown) => {
        originSubmit.setError(
          caught instanceof Error
            ? caught.message
            : 'Origin could not be added.',
        )
      })
      .finally(originSubmit.finishSubmit)
  }, [origin, originSubmit, refreshConfig])

  const handleRemoveOrigin = useCallback(
    (targetOrigin: string) => {
      originSubmit.beginSubmit()
      removeEmbedOrigin(targetOrigin)
        .then(refreshConfig)
        .catch((caught: unknown) => {
          originSubmit.setError(
            caught instanceof Error
              ? caught.message
              : 'Origin could not be removed.',
          )
        })
        .finally(originSubmit.finishSubmit)
    },
    [originSubmit, refreshConfig],
  )

  if (loadState !== 'ready') {
    return (
      <LoadState
        error={loadError}
        errorTitle='Could not load embed settings'
        loadingTitle='Loading embed settings'
        status={loadState}
      />
    )
  }

  if (!config) {
    return (
      <StatePanel title='Embed settings unavailable'>
        The backend did not return a tenant embed configuration.
      </StatePanel>
    )
  }

  return (
    <div className='grid gap-5'>
      <EmbedAvailability
        config={config}
        error={enabledSubmit.error}
        isPending={enabledSubmit.isPending}
        onToggle={handleEnabledChange}
      />
      <EmbedOrigins
        config={config}
        error={originSubmit.error}
        isPending={originSubmit.isPending}
        onAddOrigin={handleAddOrigin}
        onOriginChange={setOrigin}
        onRemoveOrigin={handleRemoveOrigin}
        origin={origin}
      />
    </div>
  )
}

function EmbedAvailability({
  config,
  error,
  isPending,
  onToggle,
}: {
  config: EmbedConfig
  error: string
  isPending: boolean
  onToggle: () => void
}) {
  const status = getEmbedAvailabilityStatus(config.enabled)

  return (
    <section
      aria-label='Embed availability'
      className='grid gap-4 rounded-lg border border-border bg-card p-6'
    >
      <div className='flex flex-wrap items-center justify-between gap-4'>
        <div>
          <h2 className='m-0 text-2xl leading-tight text-foreground'>
            Embed availability
          </h2>
          <p className='mt-2 mb-0 text-muted-foreground'>
            Allow the tenant widget to issue guest sessions from approved sites.
          </p>
        </div>
        <Button
          disabled={isPending}
          onClick={onToggle}
          type='button'
          variant={config.enabled ? 'outline' : 'default'}
        >
          {status.action}
        </Button>
      </div>
      <StatePanel title={status.title}>{status.description}</StatePanel>
      <AuthErrorAlert message={error} title='Update failed.' />
    </section>
  )
}

function getEmbedAvailabilityStatus(enabled: boolean) {
  return enabled
    ? {
        action: 'Disable embed',
        description: 'Approved origins can load the chat widget.',
        title: 'Widget enabled',
      }
    : {
        action: 'Enable embed',
        description: 'No host page can start an embed guest session yet.',
        title: 'Widget disabled',
      }
}

function EmbedOrigins({
  config,
  error,
  isPending,
  onAddOrigin,
  onOriginChange,
  onRemoveOrigin,
  origin,
}: {
  config: EmbedConfig
  error: string
  isPending: boolean
  onAddOrigin: () => void
  onOriginChange: (origin: string) => void
  onRemoveOrigin: (origin: string) => void
  origin: string
}) {
  const handleSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      onAddOrigin()
    },
    [onAddOrigin],
  )
  const handleOriginChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      onOriginChange(event.target.value)
    },
    [onOriginChange],
  )

  return (
    <section
      aria-label='Allowed embed origins'
      className='grid gap-4 rounded-lg border border-border bg-card p-6'
    >
      <header>
        <h2 className='m-0 text-2xl leading-tight text-foreground'>
          Allowed origins
        </h2>
        <p className='mt-2 mb-0 text-muted-foreground'>
          Add each production or staging site that may host the widget.
        </p>
      </header>
      <form className='flex flex-col gap-3 sm:flex-row' onSubmit={handleSubmit}>
        <Input
          aria-label='Allowed origin'
          onChange={handleOriginChange}
          placeholder='https://school.example'
          value={origin}
        />
        <Button disabled={isPending} type='submit'>
          Add origin
        </Button>
      </form>
      <EmbedOriginList
        isPending={isPending}
        onRemoveOrigin={onRemoveOrigin}
        origins={config.allowed_origins}
      />
      <AuthErrorAlert message={error} title='Origin update failed.' />
    </section>
  )
}

function EmbedOriginList({
  isPending,
  onRemoveOrigin,
  origins,
}: {
  isPending: boolean
  onRemoveOrigin: (origin: string) => void
  origins: Array<string>
}) {
  if (origins.length === 0) {
    return (
      <StatePanel title='No origins yet'>
        Add one origin before using the widget on a host page.
      </StatePanel>
    )
  }

  return (
    <ul className='m-0 grid list-none gap-2 p-0'>
      {origins.map((origin) => (
        <EmbedOriginItem
          isPending={isPending}
          key={origin}
          onRemoveOrigin={onRemoveOrigin}
          origin={origin}
        />
      ))}
    </ul>
  )
}

function EmbedOriginItem({
  isPending,
  onRemoveOrigin,
  origin,
}: {
  isPending: boolean
  onRemoveOrigin: (origin: string) => void
  origin: string
}) {
  const handleRemove = useCallback(() => {
    onRemoveOrigin(origin)
  }, [onRemoveOrigin, origin])

  return (
    <li className='flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-background p-3 text-sm'>
      <span className='break-all text-foreground'>{origin}</span>
      <Button
        disabled={isPending}
        onClick={handleRemove}
        type='button'
        variant='outline'
      >
        Remove
      </Button>
    </li>
  )
}
