import { useCallback, useEffect, useMemo, useState } from 'react'
import { Check, Copy, MessageCircle, X } from 'lucide-react'
import type { ChangeEvent, Dispatch, FormEvent, SetStateAction } from 'react'

import type { EmbedConfig } from '@/lib/embed-config-types'
import type {
  EmbedLanguage,
  EmbedPosition,
  EmbedTheme,
} from '@/lib/embed-widget'
import { useAuth } from '@/auth-provider'
import { AuthErrorAlert } from '@/components/shared/auth-error-alert'
import {
  AdminSurface,
  AdminSurfaceHeader,
} from '@/components/shared/admin-surface'
import { LoadState } from '@/components/shared/load-state'
import { StatePanel } from '@/components/shared/state-panel'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import {
  addEmbedOrigin,
  getEmbedConfig,
  removeEmbedOrigin,
  updateEmbedConfig,
} from '@/lib/admin-api'
import {
  buildEmbedSnippet,
  defaultEmbedTheme,
  getEmbedCopy,
  readEmbedTheme,
  readableForeground,
} from '@/lib/embed-widget'

type LoadStatus = 'loading' | 'ready' | 'error'
type ErrorScope = 'configuration' | 'origins' | ''

export function EmbedConfigPanel() {
  const { auth } = useAuth()
  const tenantSlug =
    auth.status === 'authenticated' ? (auth.session.user.tenant_slug ?? '') : ''
  const [config, setConfig] = useState<EmbedConfig | null>(null)
  const [theme, setTheme] = useState<EmbedTheme>(defaultEmbedTheme)
  const [enabled, setEnabled] = useState(false)
  const [status, setStatus] = useState<LoadStatus>('loading')
  const [error, setError] = useState('')
  const [errorScope, setErrorScope] = useState<ErrorScope>('')
  const [saving, setSaving] = useState(false)
  const [origin, setOrigin] = useState('')
  const [copied, setCopied] = useState(false)
  const [copyError, setCopyError] = useState('')
  const [saveSuccess, setSaveSuccess] = useState('')

  const load = useCallback(async () => {
    setStatus('loading')
    try {
      const next = await getEmbedConfig()
      setConfig(next)
      setEnabled(next.enabled)
      setTheme(readEmbedTheme(next.theme_config))
      setError('')
      setErrorScope('')
      setStatus('ready')
    } catch (caught) {
      setError(
        errorMessage(
          caught,
          'Unable to load website chat settings. Check your connection and try again.',
        ),
      )
      setErrorScope('configuration')
      setStatus('error')
    }
  }, [])

  useEffect(() => {
    load().catch(() => {})
  }, [load])

  const save = useCallback(async () => {
    setSaving(true)
    setError('')
    setErrorScope('configuration')
    setSaveSuccess('')
    try {
      const next = await updateEmbedConfig({
        enabled,
        theme_config: { ...theme },
      })
      setConfig(next)
      setEnabled(next.enabled)
      setTheme(readEmbedTheme(next.theme_config))
      setSaveSuccess('Website chat settings saved.')
    } catch (caught) {
      setError(
        errorMessage(
          caught,
          'Unable to save website chat settings. Try again.',
        ),
      )
      setErrorScope('configuration')
    } finally {
      setSaving(false)
    }
  }, [enabled, theme])

  const addOrigin = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      if (!origin.trim()) {
        setError('Enter the website origin, including http:// or https://.')
        setErrorScope('origins')
        return
      }
      setSaving(true)
      setError('')
      setErrorScope('origins')
      try {
        await addEmbedOrigin(origin.trim())
        setOrigin('')
        const next = await getEmbedConfig()
        setConfig(next)
        setError('')
      } catch (caught) {
        setError(
          errorMessage(
            caught,
            'Unable to add this website. Check the address and try again.',
          ),
        )
      } finally {
        setSaving(false)
      }
    },
    [origin],
  )

  const removeOrigin = useCallback(async (target: string) => {
    setSaving(true)
    setError('')
    setErrorScope('origins')
    try {
      await removeEmbedOrigin(target)
      const next = await getEmbedConfig()
      setConfig(next)
      setError('')
    } catch (caught) {
      setError(
        errorMessage(caught, 'Unable to remove this website. Try again.'),
      )
    } finally {
      setSaving(false)
    }
  }, [])

  const snippet = useMemo(
    () =>
      buildEmbedSnippet({
        apiBase: config?.public_embed_base_url ?? '',
        tenantSlug,
        theme,
      }),
    [config?.public_embed_base_url, tenantSlug, theme],
  )
  const savedTheme = useMemo(
    () => readEmbedTheme(config?.theme_config ?? {}),
    [config?.theme_config],
  )
  const hasConfigurationChanges =
    config !== null &&
    (enabled !== config.enabled ||
      theme.color !== savedTheme.color ||
      theme.language !== savedTheme.language ||
      theme.position !== savedTheme.position)

  const copySnippet = useCallback(async () => {
    setCopyError('')
    try {
      const clipboard = Reflect.get(navigator, 'clipboard') as
        | Clipboard
        | undefined
      if (!clipboard) {
        throw new Error(
          'Clipboard access is unavailable. Select and copy the install code manually.',
        )
      }
      await clipboard.writeText(snippet)
      setCopied(true)
    } catch (caught) {
      setCopied(false)
      setCopyError(
        errorMessage(
          caught,
          'Unable to copy the install code. Select and copy it manually.',
        ),
      )
    }
  }, [snippet])
  useEffect(() => {
    setCopied(false)
    setCopyError('')
  }, [snippet])
  const handleEnabledChange = useCallback((value: boolean) => {
    setEnabled(value)
    setSaveSuccess('')
  }, [])
  const handleColorChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      setThemeField(setTheme, 'color', event.target.value)
      setSaveSuccess('')
    },
    [],
  )
  const handleLanguageChange = useCallback(
    (event: ChangeEvent<HTMLSelectElement>) => {
      setThemeField(setTheme, 'language', event.target.value as EmbedLanguage)
      setSaveSuccess('')
    },
    [],
  )
  const handlePositionChange = useCallback(
    (event: ChangeEvent<HTMLSelectElement>) => {
      setThemeField(setTheme, 'position', event.target.value as EmbedPosition)
      setSaveSuccess('')
    },
    [],
  )

  if (status !== 'ready') {
    return (
      <LoadState
        error={error}
        errorTitle='Unable to load website chat settings'
        loadingTitle='Loading website chat settings'
        loadingVariant='settings'
        status={status}
      />
    )
  }
  if (!config) {
    return (
      <StatePanel title='Website chat settings unavailable'>
        Reload the page. If the problem continues, contact your platform
        administrator.
      </StatePanel>
    )
  }

  return (
    <div className='mt-7 grid gap-7'>
      <EmbedSetupGuide
        config={config}
        copied={copied}
        hasConfigurationChanges={hasConfigurationChanges}
      />

      <section
        aria-label='Website chat settings'
        className='overflow-hidden rounded-[1.5rem] bg-[var(--admin-surface)] ring-1 ring-[var(--admin-line)]'
      >
        <div className='grid lg:grid-cols-[minmax(21rem,0.9fr)_minmax(25rem,1.1fr)]'>
          <WidgetPreview theme={theme} />
          <section
            aria-labelledby='embed-configuration-title'
            className='grid content-center gap-7 p-6 sm:p-8'
          >
            <header>
              <p className='mb-2.5 text-xs font-semibold tracking-[0.12em] text-[var(--admin-muted)] uppercase'>
                Appearance
              </p>
              <h2
                className='m-0 text-xl leading-snug font-semibold tracking-[-0.02em] text-[var(--admin-ink)]'
                id='embed-configuration-title'
              >
                Chat appearance
              </h2>
              <p className='mt-2 mb-0 max-w-md text-sm leading-6 text-[var(--admin-muted)]'>
                Choose how chat looks and where it appears on approved websites.
              </p>
            </header>

            <PublishedState config={config} />

            <div className='flex items-center justify-between gap-5 rounded-2xl bg-[var(--admin-surface-muted)] p-4 sm:p-5'>
              <div>
                <Label htmlFor='embed-enabled'>Show chat widget</Label>
                <p className='mt-1 mb-0 text-sm leading-5 text-[var(--admin-muted)]'>
                  Only approved websites can open guest chats.
                </p>
              </div>
              <Switch
                checked={enabled}
                disabled={saving}
                id='embed-enabled'
                onCheckedChange={handleEnabledChange}
              />
            </div>

            <div className='grid gap-4 sm:grid-cols-2'>
              <div className='grid gap-2 sm:col-span-2'>
                <Label htmlFor='embed-color'>Theme color</Label>
                <div className='flex h-12 items-center gap-3 rounded-xl border border-input px-2.5'>
                  <Input
                    className='size-8 shrink-0 cursor-pointer rounded-lg border-0 p-0 shadow-none focus-visible:ring-2'
                    id='embed-color'
                    onChange={handleColorChange}
                    type='color'
                    value={theme.color}
                  />
                  <span className='text-sm font-medium tracking-[0.06em] text-[var(--admin-ink)] uppercase'>
                    {theme.color}
                  </span>
                  <span className='ms-auto text-xs text-[var(--admin-muted)]'>
                    Widget accent
                  </span>
                </div>
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='embed-language'>Language</Label>
                <NativeSelect
                  className='w-full'
                  id='embed-language'
                  onChange={handleLanguageChange}
                  value={theme.language}
                >
                  <NativeSelectOption value='en'>English</NativeSelectOption>
                  <NativeSelectOption value='ms'>
                    Bahasa Melayu
                  </NativeSelectOption>
                  <NativeSelectOption value='zh'>中文</NativeSelectOption>
                </NativeSelect>
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='embed-position'>Position</Label>
                <NativeSelect
                  className='w-full'
                  id='embed-position'
                  onChange={handlePositionChange}
                  value={theme.position}
                >
                  <NativeSelectOption value='bottom-right'>
                    Bottom right
                  </NativeSelectOption>
                  <NativeSelectOption value='bottom-left'>
                    Bottom left
                  </NativeSelectOption>
                </NativeSelect>
              </div>
            </div>
            <div className='flex flex-wrap items-center justify-between gap-3 border-t border-[var(--admin-line)] pt-5'>
              <p
                aria-live='polite'
                className={`m-0 flex items-center gap-2 text-sm font-medium ${hasConfigurationChanges ? 'text-[var(--status-warning-text)]' : 'text-[var(--admin-muted)]'}`}
              >
                <span
                  aria-hidden='true'
                  className={`size-2 rounded-full ${hasConfigurationChanges ? 'bg-[var(--status-warning-border)]' : 'bg-[var(--surface-primary-default)]'}`}
                />
                {hasConfigurationChanges
                  ? 'Unsaved changes'
                  : 'All changes saved'}
              </p>
              <Button
                className='w-fit'
                disabled={saving || !hasConfigurationChanges}
                onClick={save}
                type='button'
              >
                {saving ? 'Saving…' : 'Save website chat'}
              </Button>
              <AuthErrorAlert
                message={errorScope === 'configuration' ? error : ''}
                title='Unable to save website chat settings'
              />
              {saveSuccess && (
                <p
                  className='w-full text-sm font-medium text-[var(--status-success-text)]'
                  role='status'
                >
                  {saveSuccess}
                </p>
              )}
            </div>
          </section>
        </div>
      </section>

      <section
        aria-label='Website chat installation'
        className='grid overflow-hidden rounded-[1.5rem] bg-[var(--admin-surface)] ring-1 ring-[var(--admin-line)] lg:grid-cols-2'
      >
        <OriginsSection
          config={config}
          disabled={saving}
          error={errorScope === 'origins' ? error : ''}
          onAdd={addOrigin}
          onChange={setOrigin}
          onRemove={removeOrigin}
          origin={origin}
        />

        <section
          aria-labelledby='install-snippet-title'
          className='grid content-start gap-5 p-6 sm:p-8'
        >
          <header>
            <p className='mb-2.5 text-xs font-semibold tracking-[0.12em] text-[var(--admin-muted)] uppercase'>
              Installation
            </p>
            <h2
              className='m-0 text-xl font-semibold tracking-[-0.025em] text-[var(--admin-ink)]'
              id='install-snippet-title'
            >
              Install website chat
            </h2>
            <p className='mt-2 mb-0 text-sm leading-6 text-[var(--admin-muted)]'>
              Paste this code before the closing &lt;/body&gt; tag on an
              approved website.
            </p>
          </header>
          {!tenantSlug ? (
            <StatePanel title='School account required'>
              Sign in to a school account before copying the install code.
            </StatePanel>
          ) : (
            <>
              <pre className='max-h-44 overflow-auto rounded-2xl bg-[var(--admin-ink)] p-4 text-xs leading-5 text-[var(--admin-nav-text)]'>
                <code>{snippet}</code>
              </pre>
              <Button
                className='w-fit'
                disabled={!tenantSlug}
                onClick={copySnippet}
                type='button'
                variant='outline'
              >
                {copied ? (
                  <Check aria-hidden='true' />
                ) : (
                  <Copy aria-hidden='true' />
                )}
                {copied ? 'Install code copied' : 'Copy install code'}
              </Button>
              <AuthErrorAlert
                message={copyError}
                title='Unable to copy install code'
              />
            </>
          )}
        </section>
      </section>

      <EmbedTroubleshooting />
    </div>
  )
}

function EmbedSetupGuide({
  config,
  copied,
  hasConfigurationChanges,
}: {
  config: EmbedConfig
  copied: boolean
  hasConfigurationChanges: boolean
}) {
  const hasApprovedOrigin = config.allowed_origins.length > 0
  const currentStep = !hasApprovedOrigin
    ? 0
    : hasConfigurationChanges
      ? 1
      : !config.enabled
        ? 2
        : copied
          ? 4
          : 3
  const steps = [
    ['Approve a website', 'Enter the complete website origin.'],
    ['Configure appearance', 'Set color, language, and position.'],
    ['Show the chat widget', 'Turn it on and save your changes.'],
    ['Install website chat', 'Copy the install code into your website.'],
    ['Test website chat', 'Send a test message from the website.'],
  ] as const

  return (
    <AdminSurface contentClassName='grid gap-7 p-6 sm:p-8'>
      <AdminSurfaceHeader
        description={`Step ${currentStep + 1} of ${steps.length}. Finish the current task before moving on.`}
        title='Setup guide'
      />
      <ol
        aria-label='Website chat setup steps'
        className='m-0 grid list-none p-0 lg:grid-cols-5'
      >
        {steps.map(([title, description], index) => {
          const completed = index < currentStep
          const current = index === currentStep
          const stateLabel = completed
            ? 'Complete'
            : current
              ? 'Current step'
              : 'Upcoming'

          return (
            <li
              aria-current={current ? 'step' : undefined}
              className='relative grid grid-cols-[2.5rem_minmax(0,1fr)] gap-3 pb-7 last:pb-0 lg:block lg:pe-5 lg:pb-0'
              key={title}
            >
              {index < steps.length - 1 && (
                <span
                  aria-hidden='true'
                  className={`absolute top-10 bottom-0 left-[1.21875rem] w-px lg:top-5 lg:right-0 lg:bottom-auto lg:left-10 lg:h-px lg:w-auto ${completed ? 'bg-[var(--admin-navigation)]' : 'bg-[var(--admin-line)]'}`}
                />
              )}
              <span
                aria-hidden='true'
                className={`relative z-10 flex size-10 shrink-0 items-center justify-center rounded-full text-sm font-semibold ${completed ? 'bg-[var(--admin-navigation)] text-[var(--admin-nav-text)]' : current ? 'bg-[var(--admin-accent)] text-[var(--admin-ink)] ring-4 ring-[var(--admin-surface-muted)]' : 'bg-[var(--admin-surface-muted)] text-[var(--admin-muted)] ring-1 ring-[var(--admin-line)]'}`}
              >
                {completed ? <Check className='size-4' /> : index + 1}
              </span>
              <div className='min-w-0 lg:mt-4'>
                <p
                  className={`m-0 text-xs font-semibold tracking-[0.1em] uppercase ${current ? 'text-[var(--admin-ink)]' : 'text-[var(--admin-muted)]'}`}
                >
                  {stateLabel}
                </p>
                <h3 className='mt-1 text-sm font-semibold'>
                  <span className='sr-only'>Step {index + 1}: </span>
                  {title}
                </h3>
                <p className='mt-1 text-sm leading-5 text-[var(--admin-muted)]'>
                  {description}
                </p>
              </div>
            </li>
          )
        })}
      </ol>
    </AdminSurface>
  )
}

function EmbedTroubleshooting() {
  return (
    <AdminSurface contentClassName='grid gap-6 p-6 sm:p-8'>
      <AdminSurfaceHeader
        description='Fast checks for the two most common setup failures.'
        title='Troubleshooting'
      />
      <div className='grid gap-5 sm:grid-cols-2'>
        <section className='rounded-2xl bg-[var(--admin-surface-muted)] p-5 sm:p-6'>
          <h3 className='text-sm font-semibold'>
            Website address does not match
          </h3>
          <p className='mt-2 text-sm leading-6 text-[var(--admin-muted)]'>
            Match the approved origin to your website’s protocol, host, and
            port. Remove any path, save again, then reload the website.
          </p>
        </section>
        <section className='rounded-2xl bg-[var(--admin-surface-muted)] p-5 sm:p-6'>
          <h3 className='text-sm font-semibold'>Chat widget remains hidden</h3>
          <p className='mt-2 text-sm leading-6 text-[var(--admin-muted)]'>
            Confirm the website is approved, turn on Show chat widget, and save.
            Refresh the website after installing the code.
          </p>
        </section>
      </div>
    </AdminSurface>
  )
}

function OriginsSection({
  config,
  disabled,
  error,
  onAdd,
  onChange,
  onRemove,
  origin,
}: {
  config: EmbedConfig
  disabled: boolean
  error: string
  onAdd: (event: FormEvent<HTMLFormElement>) => void
  onChange: (origin: string) => void
  onRemove: (origin: string) => void
  origin: string
}) {
  const handleChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => onChange(event.target.value),
    [onChange],
  )

  return (
    <section
      aria-labelledby='allowed-origins-title'
      className='grid content-start gap-5 p-6 sm:p-8 lg:border-e lg:border-[var(--admin-line)]'
    >
      <header>
        <p className='mb-2.5 text-xs font-semibold tracking-[0.12em] text-[var(--admin-muted)] uppercase'>
          Website access
        </p>
        <h2
          className='m-0 text-xl font-semibold tracking-[-0.025em] text-[var(--admin-ink)]'
          id='allowed-origins-title'
        >
          Approved websites
        </h2>
        <p className='mt-2 mb-0 text-sm leading-6 text-[var(--admin-muted)]'>
          Enter complete HTTP or HTTPS origins without paths.
        </p>
      </header>
      <form className='flex flex-col gap-3 sm:flex-row' onSubmit={onAdd}>
        <Label className='sr-only' htmlFor='embed-origin'>
          Website origin
        </Label>
        <Input
          id='embed-origin'
          onChange={handleChange}
          placeholder='https://school.example'
          value={origin}
        />
        <Button disabled={disabled} type='submit'>
          Add website
        </Button>
      </form>
      <AuthErrorAlert
        message={error}
        title='Unable to update approved websites'
      />
      {config.allowed_origins.length === 0 ? (
        <div className='rounded-2xl bg-[var(--admin-surface-muted)] p-4'>
          <h3 className='text-sm font-semibold text-[var(--admin-ink)]'>
            No websites approved
          </h3>
          <p className='mt-1 text-sm leading-5 text-[var(--admin-muted)]'>
            Add a website before showing the chat widget.
          </p>
        </div>
      ) : (
        <ul className='m-0 grid list-none gap-2 p-0'>
          {config.allowed_origins.map((allowedOrigin) => (
            <OriginItem
              disabled={disabled}
              key={allowedOrigin}
              onRemove={onRemove}
              origin={allowedOrigin}
            />
          ))}
        </ul>
      )}
    </section>
  )
}

function OriginItem({
  disabled,
  onRemove,
  origin,
}: {
  disabled: boolean
  onRemove: (origin: string) => void
  origin: string
}) {
  const handleRemove = useCallback(() => onRemove(origin), [onRemove, origin])

  return (
    <li className='flex items-center justify-between gap-3 rounded-xl bg-[var(--admin-surface-muted)] p-3 text-sm'>
      <span className='break-all'>{origin}</span>
      <Button
        aria-label={`Remove website ${origin}`}
        disabled={disabled}
        onClick={handleRemove}
        size='sm'
        type='button'
        variant='outline'
      >
        Remove website
      </Button>
    </li>
  )
}

function PublishedState({ config }: { config: EmbedConfig }) {
  const originCount = config.allowed_origins.length
  const published = config.enabled && originCount > 0
  const needsOrigin = config.enabled && originCount === 0
  const label = published ? 'Live' : needsOrigin ? 'Website required' : 'Hidden'
  const detail = published
    ? `${originCount} approved ${originCount === 1 ? 'website' : 'websites'}`
    : needsOrigin
      ? 'Chat widget is on, but no website is approved'
      : 'Chat widget is hidden'

  return (
    <div className='flex items-center gap-3 border-y border-[var(--admin-line)] py-4'>
      <span
        aria-hidden='true'
        className={`size-2.5 shrink-0 rounded-full ${published ? 'bg-[var(--surface-primary-default)]' : needsOrigin ? 'bg-[var(--status-warning-border)]' : 'bg-[var(--admin-muted)]'}`}
      />
      <div className='min-w-0'>
        <p className='m-0 text-sm font-semibold text-[var(--admin-ink)]'>
          {label}
        </p>
        <p className='m-0 text-xs text-[var(--admin-muted)]'>{detail}</p>
      </div>
      <p className='m-0 ms-auto text-xs font-semibold tracking-[0.1em] text-[var(--admin-muted)] uppercase'>
        Publication status
      </p>
    </div>
  )
}

function WidgetPreview({ theme }: { theme: EmbedTheme }) {
  const [open, setOpen] = useState(true)
  const left = theme.position === 'bottom-left'
  const copy = getEmbedCopy(theme.language)
  const foreground = readableForeground(theme.color)
  const themeStyle = useMemo(
    () => ({ backgroundColor: theme.color, color: foreground }),
    [foreground, theme.color],
  )
  const toggleOpen = useCallback(() => {
    setOpen((value) => !value)
  }, [])

  return (
    <section
      aria-labelledby='widget-preview-title'
      className='relative min-h-[30rem] overflow-hidden bg-[var(--admin-ink)] p-6 text-[var(--admin-surface)] sm:p-8 lg:min-h-[34rem]'
    >
      <p className='mb-2.5 flex items-center gap-2 text-xs font-semibold tracking-[0.12em] text-[var(--admin-nav-muted)] uppercase'>
        <span
          aria-hidden='true'
          className='size-2 rounded-full bg-[var(--admin-accent)]'
        />
        Visual preview
      </p>
      <h2
        className='m-0 text-xl leading-snug font-semibold tracking-[-0.02em]'
        id='widget-preview-title'
      >
        Preview
      </h2>
      <p className='mt-2 max-w-xs text-sm leading-6 text-[var(--admin-nav-text)]'>
        Static preview. It does not create a guest session.
      </p>
      {open && (
        <div
          aria-label='Chat preview'
          lang={theme.language}
          className={`absolute bottom-24 w-[min(20rem,calc(100%-2rem))] overflow-hidden rounded-2xl bg-[var(--admin-surface)] text-[var(--admin-ink)] ring-1 ring-[var(--admin-line)] ${left ? 'left-4 sm:left-8' : 'right-4 sm:right-8'}`}
        >
          <div className='p-4 text-sm font-semibold' style={themeStyle}>
            P&amp;AI Tutor
          </div>
          <div className='grid min-h-64 content-end gap-3 p-4'>
            <div className='max-w-[85%] rounded-2xl bg-[var(--admin-surface-muted)] p-3 text-sm leading-5'>
              {copy.greeting}
            </div>
            <div className='flex gap-2 border-t pt-3'>
              <Input
                aria-label='Preview message'
                disabled
                placeholder={copy.placeholder}
              />
              <Button disabled size='sm' type='button'>
                {copy.send}
              </Button>
            </div>
          </div>
        </div>
      )}
      <Button
        aria-expanded={open}
        aria-label={open ? 'Close chat preview' : 'Open chat preview'}
        className={`absolute bottom-6 size-12 rounded-2xl ${left ? 'left-4 sm:left-8' : 'right-4 sm:right-8'}`}
        onClick={toggleOpen}
        size='icon'
        style={themeStyle}
        type='button'
      >
        {open ? <X aria-hidden='true' /> : <MessageCircle aria-hidden='true' />}
      </Button>
    </section>
  )
}

function setThemeField<TKey extends keyof EmbedTheme>(
  setTheme: Dispatch<SetStateAction<EmbedTheme>>,
  key: TKey,
  value: EmbedTheme[TKey],
) {
  setTheme((current) => ({ ...current, [key]: value }))
}

function errorMessage(caught: unknown, fallback: string) {
  return caught instanceof Error ? caught.message : fallback
}
