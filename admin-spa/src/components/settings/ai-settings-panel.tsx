import { useCallback, useEffect, useRef, useState } from 'react'
import type { ChangeEvent, FormEvent, ReactNode } from 'react'

import type { AISettings, UpdateAISettingsInput } from '@/lib/ai-settings-types'
import { AuthErrorAlert } from '@/components/shared/auth-error-alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LoadState } from '@/components/shared/load-state'
import { StatePanel } from '@/components/shared/state-panel'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { getAISettings, updateAISettings } from '@/lib/admin-api'
import { useSubmitStatus } from '@/hooks/use-submit-status'

type PanelState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'ready'; settings: AISettings }

type SubmitStatus = ReturnType<typeof useSubmitStatus>
type SubmitSection = 'provider' | 'model' | 'key' | 'flags'

export function AISettingsPanel() {
  const [state, setState] = useState<PanelState>({ status: 'loading' })
  const [model, setModel] = useState('')
  const [keyInput, setKeyInput] = useState('')
  const [isReplacingKey, setIsReplacingKey] = useState(false)
  const requestSeq = useRef(0)
  const sectionSeq = useRef<Record<string, number>>({})
  const providerSubmit = useSubmitStatus('')
  const modelSubmit = useSubmitStatus('')
  const keySubmit = useSubmitStatus('')
  const flagsSubmit = useSubmitStatus('')

  useEffect(() => {
    getAISettings()
      .then((payload) => {
        setState({ status: 'ready', settings: payload })
        setModel(payload.openrouterModel)
      })
      .catch((caught: unknown) => {
        setState({
          status: 'error',
          message:
            caught instanceof Error
              ? caught.message
              : 'AI settings could not be loaded.',
        })
      })
  }, [])

  const submitSettings = useCallback(
    (
      section: SubmitSection,
      input: UpdateAISettingsInput,
      submit: SubmitStatus,
      fallbackMessage: string,
      onSaved?: (next: AISettings) => void,
    ) => {
      // Overlapping saves can resolve out of order; only the newest wins.
      const seq = ++requestSeq.current
      sectionSeq.current[section] = seq
      submit.beginSubmit()
      updateAISettings(input)
        .then((next) => {
          if (seq !== requestSeq.current) {
            return
          }

          setState({ status: 'ready', settings: next })
          onSaved?.(next)
        })
        .catch((caught: unknown) => {
          if (seq !== sectionSeq.current[section]) {
            return
          }

          submit.setError(
            caught instanceof Error ? caught.message : fallbackMessage,
          )
        })
        .finally(() => {
          if (seq !== sectionSeq.current[section]) {
            return
          }

          submit.finishSubmit()
        })
    },
    [],
  )

  const handleProviderChange = useCallback(
    (provider: string) => {
      submitSettings(
        'provider',
        { defaultProvider: provider },
        providerSubmit,
        'Default provider could not be changed.',
      )
    },
    [providerSubmit, submitSettings],
  )

  const handleModelSave = useCallback(() => {
    submitSettings(
      'model',
      { openrouterModel: model.trim() },
      modelSubmit,
      'OpenRouter model could not be saved.',
      (next) => setModel(next.openrouterModel),
    )
  }, [model, modelSubmit, submitSettings])

  const handleKeySave = useCallback(() => {
    if (!keyInput.trim()) {
      keySubmit.setError('API key is required.')
      return
    }

    // Write-only secret: drop the submitted key from state as soon as the
    // backend confirms; only set/last4 ever comes back.
    submitSettings(
      'key',
      { openrouterApiKey: keyInput },
      keySubmit,
      'OpenRouter API key could not be saved.',
      () => {
        setKeyInput('')
        setIsReplacingKey(false)
      },
    )
  }, [keyInput, keySubmit, submitSettings])

  const handleKeyClear = useCallback(() => {
    submitSettings(
      'key',
      { openrouterApiKey: '' },
      keySubmit,
      'OpenRouter API key could not be cleared.',
      () => {
        setKeyInput('')
        setIsReplacingKey(false)
      },
    )
  }, [keySubmit, submitSettings])

  const handleReplaceKey = useCallback(() => {
    setIsReplacingKey(true)
  }, [])

  const handleReplaceCancel = useCallback(() => {
    setKeyInput('')
    setIsReplacingKey(false)
  }, [])

  const handleFlagToggle = useCallback(
    (name: string, enabled: boolean) => {
      submitSettings(
        'flags',
        { flags: { [name]: !enabled } },
        flagsSubmit,
        'Feature flag could not be changed.',
      )
    },
    [flagsSubmit, submitSettings],
  )

  const handleFlagReset = useCallback(
    (name: string) => {
      submitSettings(
        'flags',
        { flags: { [name]: null } },
        flagsSubmit,
        'Feature flag could not be reset.',
      )
    },
    [flagsSubmit, submitSettings],
  )

  if (state.status !== 'ready') {
    return (
      <LoadState
        error={state.status === 'error' ? state.message : null}
        errorTitle='Could not load AI settings'
        loadingTitle='Loading AI settings'
        status={state.status}
      />
    )
  }

  const { settings } = state

  return (
    <div className='grid gap-5'>
      <DefaultProviderSection
        error={providerSubmit.error}
        isPending={providerSubmit.isPending}
        onProviderChange={handleProviderChange}
        settings={settings}
      />
      <OpenRouterModelSection
        error={modelSubmit.error}
        isPending={modelSubmit.isPending}
        model={model}
        onModelChange={setModel}
        onSave={handleModelSave}
      />
      <OpenRouterKeySection
        error={keySubmit.error}
        isPending={keySubmit.isPending}
        isReplacing={isReplacingKey}
        keyInput={keyInput}
        keySource={settings.sources.openrouterKey}
        keyStatus={settings.openrouterKey}
        onCancelReplace={handleReplaceCancel}
        onClear={handleKeyClear}
        onKeyInputChange={setKeyInput}
        onReplace={handleReplaceKey}
        onSave={handleKeySave}
      />
      <FeatureFlagsSection
        error={flagsSubmit.error}
        flags={settings.flags}
        isPending={flagsSubmit.isPending}
        onReset={handleFlagReset}
        onToggle={handleFlagToggle}
        sources={settings.sources.flags}
      />
    </div>
  )
}

function SettingsSection({
  children,
  description,
  label,
  title,
}: {
  children: ReactNode
  description: string
  label: string
  title: string
}) {
  return (
    <section
      aria-label={label}
      className='grid gap-4 rounded-lg border border-border bg-card p-6'
    >
      <header>
        <h2 className='m-0 text-2xl leading-tight text-foreground'>{title}</h2>
        <p className='mt-2 mb-0 text-muted-foreground'>{description}</p>
      </header>
      {children}
    </section>
  )
}

function DefaultProviderSection({
  error,
  isPending,
  onProviderChange,
  settings,
}: {
  error: string
  isPending: boolean
  onProviderChange: (provider: string) => void
  settings: AISettings
}) {
  return (
    <SettingsSection
      description='Route tutor turns through this provider unless a task overrides it.'
      label='Default AI provider'
      title='Default provider'
    >
      <div className='flex flex-col gap-2'>
        <Label htmlFor='ai-default-provider'>Provider</Label>
        <Select
          disabled={isPending}
          onValueChange={onProviderChange}
          value={settings.defaultProvider}
        >
          <SelectTrigger className='sm:max-w-xs' id='ai-default-provider'>
            <SelectValue placeholder='Not set' />
          </SelectTrigger>
          <SelectContent>
            {settings.availableProviders.map((provider) => (
              <SelectItem key={provider} value={provider}>
                {provider}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <AuthErrorAlert message={error} title='Provider update failed.' />
    </SettingsSection>
  )
}

function OpenRouterModelSection({
  error,
  isPending,
  model,
  onModelChange,
  onSave,
}: {
  error: string
  isPending: boolean
  model: string
  onModelChange: (model: string) => void
  onSave: () => void
}) {
  const handleSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      onSave()
    },
    [onSave],
  )
  const handleModelChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      onModelChange(event.target.value)
    },
    [onModelChange],
  )

  return (
    <SettingsSection
      description='Model slug requested when turns route through OpenRouter.'
      label='OpenRouter model'
      title='OpenRouter model'
    >
      <form className='flex flex-col gap-3 sm:flex-row' onSubmit={handleSubmit}>
        <Input
          aria-label='OpenRouter model'
          onChange={handleModelChange}
          placeholder='anthropic/claude-sonnet-4.5'
          value={model}
        />
        <Button disabled={isPending} type='submit'>
          Save model
        </Button>
      </form>
      <AuthErrorAlert message={error} title='Model update failed.' />
    </SettingsSection>
  )
}

function OpenRouterKeySection({
  error,
  isPending,
  isReplacing,
  keyInput,
  keySource,
  keyStatus,
  onCancelReplace,
  onClear,
  onKeyInputChange,
  onReplace,
  onSave,
}: {
  error: string
  isPending: boolean
  isReplacing: boolean
  keyInput: string
  keySource: string
  keyStatus: AISettings['openrouterKey']
  onCancelReplace: () => void
  onClear: () => void
  onKeyInputChange: (value: string) => void
  onReplace: () => void
  onSave: () => void
}) {
  const showMaskedState = keyStatus.set && !isReplacing

  return (
    <SettingsSection
      description='The key is write-only: it is stored encrypted and never shown again.'
      label='OpenRouter API key'
      title='OpenRouter API key'
    >
      {showMaskedState ? (
        <ConfiguredKeyState
          fromEnv={keySource === 'env'}
          isPending={isPending}
          last4={keyStatus.last4}
          onClear={onClear}
          onReplace={onReplace}
        />
      ) : (
        <KeyEntryForm
          isPending={isPending}
          isReplacing={isReplacing}
          keyInput={keyInput}
          onCancelReplace={onCancelReplace}
          onKeyInputChange={onKeyInputChange}
          onSave={onSave}
        />
      )}
      <AuthErrorAlert message={error} title='API key update failed.' />
    </SettingsSection>
  )
}

function ConfiguredKeyState({
  fromEnv,
  isPending,
  last4,
  onClear,
  onReplace,
}: {
  fromEnv: boolean
  isPending: boolean
  last4: string
  onClear: () => void
  onReplace: () => void
}) {
  return (
    <div className='flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-background p-3 text-sm'>
      <span className='font-mono text-foreground'>
        configured &middot;&middot;&middot;&middot; {last4}
        {fromEnv ? (
          <span className='ml-2 font-sans text-muted-foreground'>
            from environment
          </span>
        ) : null}
      </span>
      <div className='flex gap-2'>
        <Button
          disabled={isPending}
          onClick={onReplace}
          type='button'
          variant='outline'
        >
          Replace key
        </Button>
        {fromEnv ? (
          <span className='self-center text-muted-foreground'>
            Set in server environment; clear it there.
          </span>
        ) : (
          <Button
            disabled={isPending}
            onClick={onClear}
            type='button'
            variant='outline'
          >
            Clear key
          </Button>
        )}
      </div>
    </div>
  )
}

function KeyEntryForm({
  isPending,
  isReplacing,
  keyInput,
  onCancelReplace,
  onKeyInputChange,
  onSave,
}: {
  isPending: boolean
  isReplacing: boolean
  keyInput: string
  onCancelReplace: () => void
  onKeyInputChange: (value: string) => void
  onSave: () => void
}) {
  const handleSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      onSave()
    },
    [onSave],
  )
  const handleKeyChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      onKeyInputChange(event.target.value)
    },
    [onKeyInputChange],
  )

  return (
    <form className='flex flex-col gap-3 sm:flex-row' onSubmit={handleSubmit}>
      <Input
        aria-label='OpenRouter API key'
        autoComplete='off'
        onChange={handleKeyChange}
        placeholder='sk-or-...'
        type='password'
        value={keyInput}
      />
      <Button disabled={isPending} type='submit'>
        Save key
      </Button>
      {isReplacing ? (
        <Button
          disabled={isPending}
          onClick={onCancelReplace}
          type='button'
          variant='outline'
        >
          Cancel
        </Button>
      ) : null}
    </form>
  )
}

function FeatureFlagsSection({
  error,
  flags,
  isPending,
  onReset,
  onToggle,
  sources,
}: {
  error: string
  flags: Record<string, boolean>
  isPending: boolean
  onReset: (name: string) => void
  onToggle: (name: string, enabled: boolean) => void
  sources: Record<string, string>
}) {
  // Object.keys already returns a fresh array, so sorting in place is safe.
  const names = Object.keys(flags).sort()

  return (
    <SettingsSection
      description='Turn platform-wide AI behaviors on or off for every tenant.'
      label='AI feature flags'
      title='Feature flags'
    >
      {names.length === 0 ? (
        <StatePanel title='No feature flags'>
          The backend did not report any platform AI feature flags.
        </StatePanel>
      ) : (
        <ul className='m-0 grid list-none gap-2 p-0'>
          {names.map((name) => (
            <FeatureFlagItem
              enabled={flags[name] === true}
              isPending={isPending}
              key={name}
              name={name}
              onReset={onReset}
              onToggle={onToggle}
              source={sources[name] ?? 'none'}
            />
          ))}
        </ul>
      )}
      <AuthErrorAlert message={error} title='Flag update failed.' />
    </SettingsSection>
  )
}

function FeatureFlagItem({
  enabled,
  isPending,
  name,
  onReset,
  onToggle,
  source,
}: {
  enabled: boolean
  isPending: boolean
  name: string
  onReset: (name: string) => void
  onToggle: (name: string, enabled: boolean) => void
  source: string
}) {
  const handleToggle = useCallback(() => {
    onToggle(name, enabled)
  }, [enabled, name, onToggle])
  const handleReset = useCallback(() => {
    onReset(name)
  }, [name, onReset])

  return (
    <li className='flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-background p-3 text-sm'>
      <span className='break-all text-foreground'>
        {name}
        <span className='ml-2 text-muted-foreground'>
          {enabled ? 'Enabled' : 'Disabled'}
        </span>
        <span className='ml-2 rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground'>
          {source}
        </span>
      </span>
      <div className='flex gap-2'>
        {source === 'db' ? (
          <Button
            aria-label={`Reset ${name}`}
            disabled={isPending}
            onClick={handleReset}
            type='button'
            variant='outline'
          >
            Reset
          </Button>
        ) : null}
        <Button
          disabled={isPending}
          onClick={handleToggle}
          type='button'
          variant={enabled ? 'outline' : 'default'}
        >
          {enabled ? 'Disable' : 'Enable'}
        </Button>
      </div>
    </li>
  )
}
