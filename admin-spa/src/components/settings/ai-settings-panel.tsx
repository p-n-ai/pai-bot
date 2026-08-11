import { useCallback, useEffect, useRef, useState } from 'react'
import type {
  ChangeEvent,
  Dispatch,
  FormEvent,
  ReactNode,
  SetStateAction,
} from 'react'

import type {
  AISettings,
  APIKeyProviderName,
  ProviderProjection,
  ProviderSelector,
  UpdateAISettingsInput,
} from '@/lib/ai-settings-types'
import type { CodexAuthStatus } from '@/lib/codex-auth-types'
import { AuthErrorAlert } from '@/components/shared/auth-error-alert'
import { LoadState } from '@/components/shared/load-state'
import { StatePanel } from '@/components/shared/state-panel'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useSubmitStatus } from '@/hooks/use-submit-status'
import {
  AdminAPIError,
  getAISettings,
  getCodexAuthStatus,
  startCodexDeviceAuth,
  updateAISettings,
} from '@/lib/admin-api'

type PanelState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'ready'; settings: AISettings }

type SubmitStatus = ReturnType<typeof useSubmitStatus>
type SubmitSection = 'provider' | 'model' | 'key' | 'enabled' | 'flags'
type SettingsMutation = Omit<UpdateAISettingsInput, 'expectedRevision'>
type CodexState =
  | { status: 'loading' }
  | { status: 'unavailable' }
  | { status: 'error'; message: string }
  | { status: 'ready'; auth: CodexAuthStatus }

const providerLabels: Record<APIKeyProviderName, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  deepseek: 'DeepSeek',
  google: 'Google',
  openrouter: 'OpenRouter',
}

export function AISettingsPanel() {
  const [state, setState] = useState<PanelState>({ status: 'loading' })
  const [modelInputs, setModelInputs] = useState<Record<string, string>>({})
  const [keyInputs, setKeyInputs] = useState<Record<string, string>>({})
  const [replacingKeys, setReplacingKeys] = useState<Record<string, boolean>>(
    {},
  )
  const [codexState, setCodexState] = useState<CodexState>({
    status: 'loading',
  })
  const [isStartingCodex, setIsStartingCodex] = useState(false)
  const nextRequestID = useRef(0)
  const sectionSeq = useRef<Record<string, number>>({})
  const settingsRef = useRef<AISettings | null>(null)
  const mutationQueue = useRef(Promise.resolve())
  const providerSubmit = useSubmitStatus('')
  const modelSubmit = useSubmitStatus('')
  const keySubmit = useSubmitStatus('')
  const enabledSubmit = useSubmitStatus('')
  const flagsSubmit = useSubmitStatus('')

  const acceptSettings = useCallback((settings: AISettings) => {
    settingsRef.current = settings
    setState({ status: 'ready', settings })
    setModelInputs(providerModels(settings.providers))
  }, [])

  useEffect(() => {
    getAISettings()
      .then(acceptSettings)
      .catch((caught: unknown) => {
        setState({
          status: 'error',
          message:
            caught instanceof Error
              ? caught.message
              : 'Unable to load AI settings. Check your connection and try again.',
        })
      })
  }, [acceptSettings])

  const loadCodexStatus = useCallback(() => {
    return getCodexAuthStatus()
      .then((auth) => setCodexState({ status: 'ready', auth }))
      .catch((caught: unknown) => {
        if (caught instanceof AdminAPIError && caught.status === 404) {
          setCodexState({ status: 'unavailable' })
          return
        }
        setCodexState({
          status: 'error',
          message:
            caught instanceof Error
              ? caught.message
              : 'Unable to check the Codex connection. Try again.',
        })
      })
  }, [])

  useEffect(() => {
    loadCodexStatus()
  }, [loadCodexStatus])

  useEffect(() => {
    if (
      codexState.status !== 'ready' ||
      (codexState.auth.state !== 'starting' &&
        codexState.auth.state !== 'awaiting_authorization')
    ) {
      return
    }
    const timer = window.setInterval(loadCodexStatus, 1500)
    return () => window.clearInterval(timer)
  }, [codexState, loadCodexStatus])

  const handleStartCodex = useCallback(() => {
    setIsStartingCodex(true)
    startCodexDeviceAuth()
      .then((auth) => setCodexState({ status: 'ready', auth }))
      .catch((caught: unknown) => {
        setCodexState({
          status: 'error',
          message:
            caught instanceof Error
              ? caught.message
              : 'Unable to start Codex verification. Try again.',
        })
      })
      .finally(() => setIsStartingCodex(false))
  }, [])

  const submitSettings = useCallback(
    (
      section: SubmitSection,
      input: SettingsMutation,
      submit: SubmitStatus,
      fallbackMessage: string,
      onSaved?: (next: AISettings) => void,
    ) => {
      const seq = ++nextRequestID.current
      sectionSeq.current[section] = seq
      submit.beginSubmit()
      mutationQueue.current = mutationQueue.current.then(async () => {
        try {
          const current = settingsRef.current
          if (!current) {
            throw new Error('Reload AI settings before making changes.')
          }
          const next = await updateAISettings({
            ...input,
            expectedRevision: current.revision,
          })
          acceptSettings(next)
          if (seq === sectionSeq.current[section]) {
            onSaved?.(next)
          }
        } catch (caught: unknown) {
          if (caught instanceof AdminAPIError && caught.status === 409) {
            try {
              acceptSettings(await getAISettings())
            } catch {
              // Keep the conflict as the actionable error; retrying will reload.
            }
          }
          if (seq !== sectionSeq.current[section]) return
          submit.setError(
            caught instanceof Error ? caught.message : fallbackMessage,
          )
        } finally {
          if (seq === sectionSeq.current[section]) submit.finishSubmit()
        }
      })
    },
    [acceptSettings],
  )

  const setDefaultProvider = useCallback(
    (value: string) => {
      submitSettings(
        'provider',
        { defaultProvider: decodeSelectorValue(value) },
        providerSubmit,
        'Unable to change the default provider. Try again.',
      )
    },
    [providerSubmit, submitSettings],
  )
  const resetDefaultProvider = useCallback(() => {
    submitSettings(
      'provider',
      { defaultProvider: null },
      providerSubmit,
      'Unable to use the environment default. Try again.',
    )
  }, [providerSubmit, submitSettings])
  const saveModel = useCallback(
    (provider: ProviderProjection) => {
      const id = providerID(provider)
      const model = (modelInputs[id] ?? '').trim()
      if (!model) {
        modelSubmit.setError('Model is required.')
        return
      }
      submitSettings(
        'model',
        { provider: modelPatch(provider, model) },
        modelSubmit,
        'Unable to save the model. Check the model name and try again.',
      )
    },
    [modelInputs, modelSubmit, submitSettings],
  )
  const resetModel = useCallback(
    (provider: ProviderProjection) => {
      submitSettings(
        'model',
        { provider: modelPatch(provider, null) },
        modelSubmit,
        'Unable to use the environment model. Try again.',
      )
    },
    [modelSubmit, submitSettings],
  )
  const saveKey = useCallback(
    (provider: Extract<ProviderProjection, { type: 'api_key' }>) => {
      const key = keyInputs[provider.name] ?? ''
      if (!key.trim()) {
        keySubmit.setError('API key is required.')
        return
      }
      submitSettings(
        'key',
        {
          provider: {
            type: 'api_key',
            name: provider.name,
            apiKey: key,
          },
        },
        keySubmit,
        `Unable to save the ${providerLabels[provider.name]} API key. Check the key and try again.`,
        () => {
          setKeyInputs((current) => ({ ...current, [provider.name]: '' }))
          setReplacingKeys((current) => ({
            ...current,
            [provider.name]: false,
          }))
        },
      )
    },
    [keyInputs, keySubmit, submitSettings],
  )
  const resetKey = useCallback(
    (provider: Extract<ProviderProjection, { type: 'api_key' }>) => {
      submitSettings(
        'key',
        {
          provider: {
            type: 'api_key',
            name: provider.name,
            apiKey: null,
          },
        },
        keySubmit,
        `Unable to reset the ${providerLabels[provider.name]} API key. Try again.`,
      )
    },
    [keySubmit, submitSettings],
  )
  const setOllamaEnabled = useCallback(
    (
      provider: Extract<ProviderProjection, { type: 'ollama' }>,
      enabled: boolean | null,
    ) => {
      submitSettings(
        'enabled',
        { provider: { type: 'ollama', enabled } },
        enabledSubmit,
        'Unable to change Ollama availability. Try again.',
      )
    },
    [enabledSubmit, submitSettings],
  )
  const toggleFlag = useCallback(
    (name: string, enabled: boolean) => {
      submitSettings(
        'flags',
        { flags: { [name]: !enabled } },
        flagsSubmit,
        'Unable to change this AI control. Try again.',
      )
    },
    [flagsSubmit, submitSettings],
  )
  const resetFlag = useCallback(
    (name: string) => {
      submitSettings(
        'flags',
        { flags: { [name]: null } },
        flagsSubmit,
        'Unable to use the default for this AI control. Try again.',
      )
    },
    [flagsSubmit, submitSettings],
  )

  if (state.status !== 'ready') {
    return (
      <LoadState
        error={state.status === 'error' ? state.message : null}
        errorTitle='Unable to load AI settings'
        loadingTitle='Loading AI settings'
        loadingVariant='settings'
        status={state.status}
      />
    )
  }

  const settings = state.settings

  return (
    <div className='mt-8 grid gap-6'>
      <div className='flex flex-wrap items-center gap-2 text-sm text-muted-foreground'>
        <Badge variant={settings.drift ? 'destructive' : 'secondary'}>
          {settings.drift ? 'Changes pending' : 'Settings in sync'}
        </Badge>
        <span>
          Requested version {settings.revision}; active version{' '}
          {settings.appliedRevision}
        </span>
      </div>
      <DefaultProviderSection
        error={providerSubmit.error}
        isPending={providerSubmit.isPending}
        onChange={setDefaultProvider}
        onReset={resetDefaultProvider}
        settings={settings}
      />
      {settings.providers.map((provider) => (
        <ProviderEditorController
          codexState={codexState}
          enabledError={enabledSubmit.error}
          isEnabledPending={enabledSubmit.isPending}
          isKeyPending={keySubmit.isPending}
          isModelPending={modelSubmit.isPending}
          isStartingCodex={isStartingCodex}
          keyError={keySubmit.error}
          keyInput={keyInputs[providerID(provider)] ?? ''}
          keyReplacing={replacingKeys[providerID(provider)] === true}
          key={providerID(provider)}
          model={modelInputs[providerID(provider)] ?? ''}
          modelError={modelSubmit.error}
          onResetKey={resetKey}
          onResetModel={resetModel}
          onSaveKey={saveKey}
          onSaveModel={saveModel}
          onSetOllamaEnabled={setOllamaEnabled}
          onStartCodex={handleStartCodex}
          provider={provider}
          setKeyInputs={setKeyInputs}
          setModelInputs={setModelInputs}
          setReplacingKeys={setReplacingKeys}
        />
      ))}
      <FeatureFlagsSection
        error={flagsSubmit.error}
        flags={settings.flags.effective}
        isPending={flagsSubmit.isPending}
        onReset={resetFlag}
        onToggle={toggleFlag}
        sources={settings.flags.sources}
      />
    </div>
  )
}

function DefaultProviderSection({
  error,
  isPending,
  onChange,
  onReset,
  settings,
}: {
  error: string
  isPending: boolean
  onChange: (value: string) => void
  onReset: () => void
  settings: AISettings
}) {
  return (
    <SettingsSection
      description='Use this provider for tutor conversations unless a specific task selects another.'
      label='Default AI provider'
      title='Default provider'
    >
      <div className='flex flex-wrap gap-2'>
        <Select
          disabled={isPending}
          onValueChange={onChange}
          value={
            settings.defaultProvider.effective
              ? selectorValue(settings.defaultProvider.effective)
              : ''
          }
        >
          <SelectTrigger className='sm:max-w-xs' id='ai-default-provider'>
            <SelectValue placeholder='Not set' />
          </SelectTrigger>
          <SelectContent>
            {settings.providers.map((provider) => (
              <SelectItem
                disabled={!provider.readiness.registrable}
                key={providerID(provider)}
                value={selectorValue(providerSelector(provider))}
              >
                {providerTitle(provider)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {settings.defaultProvider.source === 'db' ? (
          <Button
            disabled={isPending}
            onClick={onReset}
            type='button'
            variant='outline'
          >
            Use environment default
          </Button>
        ) : null}
      </div>
      <AuthErrorAlert message={error} title='Unable to update provider' />
    </SettingsSection>
  )
}

type ProviderEditorProps = {
  codexState: CodexState
  enabledError: string
  isEnabledPending: boolean
  isKeyPending: boolean
  isModelPending: boolean
  isStartingCodex: boolean
  keyError: string
  keyInput: string
  keyReplacing: boolean
  model: string
  modelError: string
  onCancelKey: () => void
  onKeyChange: (value: string) => void
  onModelChange: (value: string) => void
  onReplaceKey: () => void
  onResetKey: () => void
  onResetModel: () => void
  onSaveKey: () => void
  onSaveModel: () => void
  onSetOllamaEnabled: (enabled: boolean | null) => void
  onStartCodex: () => void
  provider: ProviderProjection
}

type ProviderEditorControllerProps = Omit<
  ProviderEditorProps,
  | 'onCancelKey'
  | 'onKeyChange'
  | 'onModelChange'
  | 'onReplaceKey'
  | 'onResetKey'
  | 'onResetModel'
  | 'onSaveKey'
  | 'onSaveModel'
  | 'onSetOllamaEnabled'
> & {
  onResetKey: (
    provider: Extract<ProviderProjection, { type: 'api_key' }>,
  ) => void
  onResetModel: (provider: ProviderProjection) => void
  onSaveKey: (
    provider: Extract<ProviderProjection, { type: 'api_key' }>,
  ) => void
  onSaveModel: (provider: ProviderProjection) => void
  onSetOllamaEnabled: (
    provider: Extract<ProviderProjection, { type: 'ollama' }>,
    enabled: boolean | null,
  ) => void
  setKeyInputs: Dispatch<SetStateAction<Record<string, string>>>
  setModelInputs: Dispatch<SetStateAction<Record<string, string>>>
  setReplacingKeys: Dispatch<SetStateAction<Record<string, boolean>>>
}

function ProviderEditorController({
  onResetKey,
  onResetModel,
  onSaveKey,
  onSaveModel,
  onSetOllamaEnabled,
  provider,
  setKeyInputs,
  setModelInputs,
  setReplacingKeys,
  ...props
}: ProviderEditorControllerProps) {
  const id = providerID(provider)
  const cancelKey = useCallback(() => {
    setKeyInputs((current) => ({ ...current, [id]: '' }))
    setReplacingKeys((current) => ({ ...current, [id]: false }))
  }, [id, setKeyInputs, setReplacingKeys])
  const changeKey = useCallback(
    (value: string) => {
      setKeyInputs((current) => ({ ...current, [id]: value }))
    },
    [id, setKeyInputs],
  )
  const changeModel = useCallback(
    (value: string) => {
      setModelInputs((current) => ({ ...current, [id]: value }))
    },
    [id, setModelInputs],
  )
  const replaceKey = useCallback(() => {
    setReplacingKeys((current) => ({ ...current, [id]: true }))
  }, [id, setReplacingKeys])
  const resetKey = useCallback(() => {
    if (provider.type === 'api_key') onResetKey(provider)
  }, [onResetKey, provider])
  const resetModel = useCallback(() => {
    onResetModel(provider)
  }, [onResetModel, provider])
  const saveKey = useCallback(() => {
    if (provider.type === 'api_key') onSaveKey(provider)
  }, [onSaveKey, provider])
  const saveModel = useCallback(() => {
    onSaveModel(provider)
  }, [onSaveModel, provider])
  const setEnabled = useCallback(
    (enabled: boolean | null) => {
      if (provider.type === 'ollama') onSetOllamaEnabled(provider, enabled)
    },
    [onSetOllamaEnabled, provider],
  )

  return (
    <ProviderEditor
      {...props}
      onCancelKey={cancelKey}
      onKeyChange={changeKey}
      onModelChange={changeModel}
      onReplaceKey={replaceKey}
      onResetKey={resetKey}
      onResetModel={resetModel}
      onSaveKey={saveKey}
      onSaveModel={saveModel}
      onSetOllamaEnabled={setEnabled}
      provider={provider}
    />
  )
}

function ProviderEditor(props: ProviderEditorProps) {
  const { provider } = props
  switch (provider.type) {
    case 'api_key':
      return <APIKeyProviderEditor {...props} provider={provider} />
    case 'ollama':
      return <OllamaProviderEditor {...props} provider={provider} />
    case 'managed_codex':
      return <ManagedCodexProviderEditor {...props} provider={provider} />
  }
  return assertNever(provider)
}

function APIKeyProviderEditor({
  isKeyPending,
  isModelPending,
  keyError,
  keyInput,
  keyReplacing,
  model,
  modelError,
  onCancelKey,
  onKeyChange,
  onModelChange,
  onReplaceKey,
  onResetKey,
  onResetModel,
  onSaveKey,
  onSaveModel,
  provider,
}: ProviderEditorProps & {
  provider: Extract<ProviderProjection, { type: 'api_key' }>
}) {
  const showMasked = provider.credential.effective.set && !keyReplacing
  return (
    <SettingsSection
      description='Choose the model and save a new API key.'
      label={`${providerLabels[provider.name]} provider`}
      title={providerLabels[provider.name]}
    >
      <ModelEditor
        isPending={isModelPending}
        model={model}
        onChange={onModelChange}
        onReset={onResetModel}
        onSave={onSaveModel}
        provider={provider}
      />
      <AuthErrorAlert message={modelError} title='Unable to update model' />
      <FieldHeading source={provider.credential.source} text='API key' />
      <p className='m-0 text-sm text-muted-foreground'>
        The key is encrypted after saving and cannot be viewed again.
      </p>
      {showMasked ? (
        <ConfiguredKeyState
          fromEnv={provider.credential.source === 'env'}
          isPending={isKeyPending}
          last4={provider.credential.effective.last4}
          onReplace={onReplaceKey}
          onReset={onResetKey}
        />
      ) : (
        <KeyEntryForm
          isPending={isKeyPending}
          isReplacing={keyReplacing}
          label={`${providerLabels[provider.name]} API key`}
          onCancel={onCancelKey}
          onChange={onKeyChange}
          onSave={onSaveKey}
          value={keyInput}
        />
      )}
      <AuthErrorAlert message={keyError} title='Unable to update API key' />
    </SettingsSection>
  )
}

function OllamaProviderEditor({
  enabledError,
  isEnabledPending,
  isModelPending,
  model,
  modelError,
  onModelChange,
  onResetModel,
  onSaveModel,
  onSetOllamaEnabled,
  provider,
}: ProviderEditorProps & {
  provider: Extract<ProviderProjection, { type: 'ollama' }>
}) {
  const toggleEnabled = useCallback(() => {
    onSetOllamaEnabled(!provider.enabled.effective)
  }, [onSetOllamaEnabled, provider.enabled.effective])
  const resetEnabled = useCallback(() => {
    onSetOllamaEnabled(null)
  }, [onSetOllamaEnabled])
  return (
    <SettingsSection
      description='Use the Ollama endpoint configured for this deployment.'
      label='Ollama provider'
      title='Ollama'
    >
      <div className='flex flex-wrap items-center gap-2'>
        <SourceBadge source={provider.enabled.source} />
        <Button
          disabled={isEnabledPending}
          onClick={toggleEnabled}
          type='button'
        >
          {provider.enabled.effective ? 'Disable Ollama' : 'Enable Ollama'}
        </Button>
        {provider.enabled.source === 'db' ? (
          <Button
            disabled={isEnabledPending}
            onClick={resetEnabled}
            type='button'
            variant='outline'
          >
            Use environment setting
          </Button>
        ) : null}
      </div>
      <AuthErrorAlert message={enabledError} title='Unable to update Ollama' />
      <ModelEditor
        isPending={isModelPending}
        model={model}
        onChange={onModelChange}
        onReset={onResetModel}
        onSave={onSaveModel}
        provider={provider}
      />
      <AuthErrorAlert message={modelError} title='Unable to update model' />
    </SettingsSection>
  )
}

function ManagedCodexProviderEditor({
  codexState,
  isModelPending,
  isStartingCodex,
  model,
  modelError,
  onModelChange,
  onResetModel,
  onSaveModel,
  onStartCodex,
  provider,
}: ProviderEditorProps & {
  provider: Extract<ProviderProjection, { type: 'managed_codex' }>
}) {
  return (
    <SettingsSection
      description='Connect the managed Codex account and choose its model.'
      label='Managed Codex provider'
      title='Managed Codex'
    >
      <CodexAuthControls
        isStarting={isStartingCodex}
        onStart={onStartCodex}
        state={codexState}
      />
      <ModelEditor
        isPending={isModelPending}
        model={model}
        onChange={onModelChange}
        onReset={onResetModel}
        onSave={onSaveModel}
        provider={provider}
      />
      <AuthErrorAlert message={modelError} title='Unable to update model' />
    </SettingsSection>
  )
}

function ModelEditor({
  isPending,
  model,
  onChange,
  onReset,
  onSave,
  provider,
}: {
  isPending: boolean
  model: string
  onChange: (value: string) => void
  onReset: () => void
  onSave: () => void
  provider: ProviderProjection
}) {
  const handleSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      onSave()
    },
    [onSave],
  )
  const handleChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      onChange(event.target.value)
    },
    [onChange],
  )
  return (
    <div className='grid gap-2'>
      <FieldHeading source={provider.model.source} text='Model' />
      <form className='flex flex-col gap-3 sm:flex-row' onSubmit={handleSubmit}>
        <Input
          aria-label={`${providerTitle(provider)} model`}
          onChange={handleChange}
          value={model}
        />
        <Button disabled={isPending} type='submit'>
          Save model
        </Button>
        {provider.model.source === 'db' ? (
          <Button
            disabled={isPending}
            onClick={onReset}
            type='button'
            variant='outline'
          >
            Use environment model
          </Button>
        ) : null}
      </form>
    </div>
  )
}

function CodexAuthControls({
  isStarting,
  onStart,
  state,
}: {
  isStarting: boolean
  onStart: () => void
  state: CodexState
}) {
  if (state.status === 'loading' || state.status === 'unavailable') return null
  const auth = state.status === 'ready' ? state.auth : null
  const awaiting = auth?.state === 'awaiting_authorization'
  const connected = auth?.state === 'connected'
  return (
    <div className='grid gap-3'>
      <div className='flex flex-wrap items-center gap-3'>
        <Badge variant={connected ? 'secondary' : 'outline'}>
          {connected
            ? 'Connected'
            : (auth?.state.replaceAll('_', ' ') ?? 'Unavailable')}
        </Badge>
        <Button
          disabled={
            isStarting ||
            auth?.state === 'starting' ||
            auth?.state === 'awaiting_authorization'
          }
          onClick={onStart}
          type='button'
        >
          {connected ? 'Reconnect Codex' : 'Connect Codex'}
        </Button>
      </div>
      {awaiting ? (
        <div className='grid gap-3 rounded-md border border-border bg-background p-4'>
          <code className='w-fit rounded bg-muted px-3 py-2 text-lg font-semibold tracking-widest text-foreground'>
            {auth.userCode}
          </code>
          <a
            className='w-fit text-sm font-medium text-primary underline'
            href={auth.verificationUrl}
            rel='noreferrer'
            target='_blank'
          >
            Open Codex verification page
          </a>
        </div>
      ) : null}
      <AuthErrorAlert
        message={
          state.status === 'error'
            ? state.message
            : auth?.state === 'failed'
              ? auth.message
              : ''
        }
        title='Unable to connect Codex'
      />
    </div>
  )
}

function ConfiguredKeyState({
  fromEnv,
  isPending,
  last4,
  onReplace,
  onReset,
}: {
  fromEnv: boolean
  isPending: boolean
  last4: string
  onReplace: () => void
  onReset: () => void
}) {
  return (
    <div className='flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-background p-3 text-sm'>
      <span className='font-mono text-foreground'>
        Saved key &middot;&middot;&middot;&middot; {last4}
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
        {!fromEnv ? (
          <Button
            disabled={isPending}
            onClick={onReset}
            type='button'
            variant='outline'
          >
            Use environment key
          </Button>
        ) : null}
      </div>
    </div>
  )
}

function KeyEntryForm({
  isPending,
  isReplacing,
  label,
  onCancel,
  onChange,
  onSave,
  value,
}: {
  isPending: boolean
  isReplacing: boolean
  label: string
  onCancel: () => void
  onChange: (value: string) => void
  onSave: () => void
  value: string
}) {
  const handleSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      onSave()
    },
    [onSave],
  )
  const handleChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      onChange(event.target.value)
    },
    [onChange],
  )
  return (
    <form className='flex flex-col gap-3 sm:flex-row' onSubmit={handleSubmit}>
      <Input
        aria-label={label}
        autoComplete='off'
        onChange={handleChange}
        placeholder='API key'
        type='password'
        value={value}
      />
      <Button disabled={isPending} type='submit'>
        Save key
      </Button>
      {isReplacing ? (
        <Button
          disabled={isPending}
          onClick={onCancel}
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
  const names = (
    Object.keys(flags) as Array<string> & {
      toSorted: () => Array<string>
    }
  ).toSorted()
  return (
    <SettingsSection
      description='Control AI features for every school.'
      label='AI feature controls'
      title='Feature flags'
    >
      {names.length === 0 ? (
        <StatePanel title='No AI feature controls available'>
          This deployment does not provide any AI feature controls.
        </StatePanel>
      ) : (
        <ul className='m-0 grid list-none gap-2 p-0'>
          {names.map((name) => (
            <FeatureFlagRow
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
      <AuthErrorAlert message={error} title='Unable to update AI feature' />
    </SettingsSection>
  )
}

function FeatureFlagRow({
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
  const reset = useCallback(() => {
    onReset(name)
  }, [name, onReset])
  const toggle = useCallback(() => {
    onToggle(name, enabled)
  }, [enabled, name, onToggle])
  return (
    <li className='flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-background p-3 text-sm'>
      <span className='flex flex-wrap items-center gap-2'>
        <span className='font-mono text-[13px]'>{name}</span>
        <SourceBadge source={source} />
      </span>
      <div className='flex gap-2'>
        {source === 'db' ? (
          <Button
            aria-label={`Reset ${name}`}
            disabled={isPending}
            onClick={reset}
            type='button'
            variant='outline'
          >
            Use default
          </Button>
        ) : null}
        <Button
          disabled={isPending}
          onClick={toggle}
          type='button'
          variant={enabled ? 'outline' : 'default'}
        >
          {enabled ? 'Disable' : 'Enable'}
        </Button>
      </div>
    </li>
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
      className='grid gap-5 rounded-2xl border border-[var(--admin-line)] bg-[var(--admin-surface)] p-5 sm:p-7'
    >
      <header>
        <h2 className='m-0 text-lg leading-tight font-semibold tracking-[-0.02em] text-foreground'>
          {title}
        </h2>
        <p className='mt-2 mb-0 max-w-2xl text-sm leading-6 text-muted-foreground'>
          {description}
        </p>
      </header>
      {children}
    </section>
  )
}

function SourceBadge({ source }: { source: string }) {
  if (source === 'db') return <Badge variant='secondary'>Override</Badge>
  if (source === 'env') return <Badge variant='outline'>Environment</Badge>
  return null
}

function FieldHeading({ source, text }: { source: string; text: string }) {
  return (
    <div className='flex items-center gap-2'>
      <Label>{text}</Label>
      <SourceBadge source={source} />
    </div>
  )
}

function providerModels(providers: ReadonlyArray<ProviderProjection>) {
  return Object.fromEntries(
    providers.map((provider) => [
      providerID(provider),
      provider.model.effective ?? '',
    ]),
  )
}

function providerID(provider: ProviderProjection): string {
  return provider.type === 'api_key' ? provider.name : provider.type
}

function providerTitle(provider: ProviderProjection): string {
  switch (provider.type) {
    case 'api_key':
      return providerLabels[provider.name]
    case 'ollama':
      return 'Ollama'
    case 'managed_codex':
      return 'Managed Codex'
  }
  return assertNever(provider)
}

function providerSelector(provider: ProviderProjection): ProviderSelector {
  switch (provider.type) {
    case 'api_key':
      return { type: 'api_key', name: provider.name }
    case 'ollama':
      return { type: 'ollama' }
    case 'managed_codex':
      return { type: 'managed_codex' }
  }
  return assertNever(provider)
}

function modelPatch(provider: ProviderProjection, model: string | null) {
  switch (provider.type) {
    case 'api_key':
      return { type: 'api_key' as const, name: provider.name, model }
    case 'ollama':
      return { type: 'ollama' as const, model }
    case 'managed_codex':
      return { type: 'managed_codex' as const, model }
  }
  return assertNever(provider)
}

function selectorValue(selector: ProviderSelector): string {
  return selector.type === 'api_key'
    ? `api_key:${selector.name}`
    : selector.type
}

function decodeSelectorValue(value: string): ProviderSelector {
  switch (value) {
    case 'api_key:openai':
      return { type: 'api_key', name: 'openai' }
    case 'api_key:anthropic':
      return { type: 'api_key', name: 'anthropic' }
    case 'api_key:deepseek':
      return { type: 'api_key', name: 'deepseek' }
    case 'api_key:google':
      return { type: 'api_key', name: 'google' }
    case 'api_key:openrouter':
      return { type: 'api_key', name: 'openrouter' }
    case 'ollama':
      return { type: 'ollama' }
    case 'managed_codex':
      return { type: 'managed_codex' }
    default:
      throw new Error('Unsupported provider selector')
  }
}

function assertNever(value: never): never {
  throw new Error(`Unsupported provider projection: ${String(value)}`)
}
