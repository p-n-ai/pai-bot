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
  readEmbedTheme,
} from '@/lib/embed-widget'

type LoadStatus = 'loading' | 'ready' | 'error'

export function EmbedConfigPanel() {
  const { auth } = useAuth()
  const tenantSlug =
    auth.status === 'authenticated' ? (auth.session.user.tenant_slug ?? '') : ''
  const [config, setConfig] = useState<EmbedConfig | null>(null)
  const [theme, setTheme] = useState<EmbedTheme>(defaultEmbedTheme)
  const [enabled, setEnabled] = useState(false)
  const [status, setStatus] = useState<LoadStatus>('loading')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [origin, setOrigin] = useState('')
  const [copied, setCopied] = useState(false)

  const load = useCallback(async () => {
    setStatus('loading')
    try {
      const next = await getEmbedConfig()
      setConfig(next)
      setEnabled(next.enabled)
      setTheme(readEmbedTheme(next.theme_config))
      setError('')
      setStatus('ready')
    } catch (caught) {
      setError(errorMessage(caught, 'Embed config could not be loaded.'))
      setStatus('error')
    }
  }, [])

  useEffect(() => {
    load().catch(() => {})
  }, [load])

  const save = useCallback(async () => {
    setSaving(true)
    setError('')
    try {
      const next = await updateEmbedConfig({
        enabled,
        theme_config: { ...theme },
      })
      setConfig(next)
      setEnabled(next.enabled)
      setTheme(readEmbedTheme(next.theme_config))
    } catch (caught) {
      setError(errorMessage(caught, 'Embed settings could not be saved.'))
    } finally {
      setSaving(false)
    }
  }, [enabled, theme])

  const addOrigin = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      if (!origin.trim()) {
        setError('Origin is required.')
        return
      }
      setSaving(true)
      setError('')
      try {
        await addEmbedOrigin(origin)
        setOrigin('')
        const next = await getEmbedConfig()
        setConfig(next)
      } catch (caught) {
        setError(errorMessage(caught, 'Origin could not be added.'))
      } finally {
        setSaving(false)
      }
    },
    [origin],
  )

  const removeOrigin = useCallback(async (target: string) => {
    setSaving(true)
    setError('')
    try {
      await removeEmbedOrigin(target)
      const next = await getEmbedConfig()
      setConfig(next)
    } catch (caught) {
      setError(errorMessage(caught, 'Origin could not be removed.'))
    } finally {
      setSaving(false)
    }
  }, [])

  const snippet = useMemo(
    () =>
      buildEmbedSnippet({
        apiBase: window.location.origin,
        tenantSlug,
        theme,
      }),
    [tenantSlug, theme],
  )

  const copySnippet = useCallback(async () => {
    await navigator.clipboard.writeText(snippet)
    setCopied(true)
  }, [snippet])
  const handleColorChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      setThemeField(setTheme, 'color', event.target.value)
    },
    [],
  )
  const handleLanguageChange = useCallback(
    (event: ChangeEvent<HTMLSelectElement>) => {
      setThemeField(setTheme, 'language', event.target.value as EmbedLanguage)
    },
    [],
  )
  const handlePositionChange = useCallback(
    (event: ChangeEvent<HTMLSelectElement>) => {
      setThemeField(setTheme, 'position', event.target.value as EmbedPosition)
    },
    [],
  )

  if (status !== 'ready') {
    return (
      <LoadState
        error={error}
        errorTitle='Could not load embed settings'
        loadingTitle='Loading embed settings'
        status={status}
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
    <div className='grid gap-5 xl:grid-cols-[minmax(0,1fr)_24rem]'>
      <div className='grid gap-5'>
        <section
          aria-labelledby='embed-configuration-title'
          className='grid gap-5 rounded-lg border border-border bg-card p-6'
        >
          <header>
            <h2
              className='m-0 text-2xl leading-tight'
              id='embed-configuration-title'
            >
              Widget configuration
            </h2>
            <p className='mt-2 mb-0 text-muted-foreground'>
              Control availability and presentation for approved host sites.
            </p>
          </header>

          <div className='flex items-center justify-between gap-4 rounded-md border p-4'>
            <div>
              <Label htmlFor='embed-enabled'>Enable widget</Label>
              <p className='mt-1 mb-0 text-sm text-muted-foreground'>
                Guest sessions remain limited to allowed origins.
              </p>
            </div>
            <Switch
              checked={enabled}
              disabled={saving}
              id='embed-enabled'
              onCheckedChange={setEnabled}
            />
          </div>

          <div className='grid gap-4 sm:grid-cols-3'>
            <div className='grid gap-2'>
              <Label htmlFor='embed-color'>Theme color</Label>
              <Input
                id='embed-color'
                onChange={handleColorChange}
                type='color'
                value={theme.color}
              />
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
          <Button disabled={saving} onClick={save} type='button'>
            {saving ? 'Saving…' : 'Save configuration'}
          </Button>
          <AuthErrorAlert message={error} title='Update failed.' />
        </section>

        <OriginsSection
          config={config}
          disabled={saving}
          onAdd={addOrigin}
          onChange={setOrigin}
          onRemove={removeOrigin}
          origin={origin}
        />

        <section
          aria-labelledby='install-snippet-title'
          className='grid gap-4 rounded-lg border border-border bg-card p-6'
        >
          <header>
            <h2 className='m-0 text-2xl' id='install-snippet-title'>
              Install snippet
            </h2>
            <p className='mt-2 mb-0 text-muted-foreground'>
              Add this before the closing body tag on an approved site.
            </p>
          </header>
          {!tenantSlug ? (
            <StatePanel title='Tenant slug unavailable'>
              Switch to a tenant session before copying the widget snippet.
            </StatePanel>
          ) : (
            <>
              <pre className='overflow-x-auto rounded-md bg-muted p-4 text-xs'>
                <code>{snippet}</code>
              </pre>
              <Button
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
                {copied ? 'Copied' : 'Copy snippet'}
              </Button>
            </>
          )}
        </section>
      </div>

      <WidgetPreview enabled={enabled} theme={theme} />
    </div>
  )
}

function OriginsSection({
  config,
  disabled,
  onAdd,
  onChange,
  onRemove,
  origin,
}: {
  config: EmbedConfig
  disabled: boolean
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
      className='grid gap-4 rounded-lg border border-border bg-card p-6'
    >
      <header>
        <h2 className='m-0 text-2xl' id='allowed-origins-title'>
          Allowed origins
        </h2>
        <p className='mt-2 mb-0 text-muted-foreground'>
          Use complete HTTP or HTTPS origins without paths.
        </p>
      </header>
      <form className='flex flex-col gap-3 sm:flex-row' onSubmit={onAdd}>
        <Label className='sr-only' htmlFor='embed-origin'>
          Allowed origin
        </Label>
        <Input
          id='embed-origin'
          onChange={handleChange}
          placeholder='https://school.example'
          value={origin}
        />
        <Button disabled={disabled} type='submit'>
          Add origin
        </Button>
      </form>
      {config.allowed_origins.length === 0 ? (
        <StatePanel title='No origins yet'>
          Add an origin before enabling the widget.
        </StatePanel>
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
    <li className='flex items-center justify-between gap-3 rounded-md border p-3 text-sm'>
      <span className='break-all'>{origin}</span>
      <Button
        aria-label={`Remove ${origin}`}
        disabled={disabled}
        onClick={handleRemove}
        size='sm'
        type='button'
        variant='outline'
      >
        Remove
      </Button>
    </li>
  )
}

function WidgetPreview({
  enabled,
  theme,
}: {
  enabled: boolean
  theme: EmbedTheme
}) {
  const [open, setOpen] = useState(true)
  const left = theme.position === 'bottom-left'
  const themeStyle = useMemo(
    () => ({ backgroundColor: theme.color }),
    [theme.color],
  )
  const toggleOpen = useCallback(() => {
    setOpen((value) => !value)
  }, [])

  return (
    <section
      aria-labelledby='widget-preview-title'
      className='relative min-h-[34rem] overflow-hidden rounded-lg border bg-muted/30 p-5'
    >
      <h2 className='m-0 text-lg' id='widget-preview-title'>
        Preview
      </h2>
      <p className='mt-1 text-sm text-muted-foreground'>
        Static preview. It does not create a guest session.
      </p>
      {open && (
        <div
          aria-label='Chat preview'
          className={`absolute bottom-20 w-[min(20rem,calc(100%-2rem))] overflow-hidden rounded-xl border bg-background shadow-xl ${left ? 'left-4' : 'right-4'}`}
        >
          <div
            className='p-4 text-sm font-semibold text-white'
            style={themeStyle}
          >
            P&amp;AI Tutor
          </div>
          <div className='grid min-h-64 content-end gap-3 p-4'>
            <div className='max-w-[85%] rounded-xl border bg-card p-3 text-sm'>
              Hi! What would you like to learn today?
            </div>
            <div className='flex gap-2 border-t pt-3'>
              <Input
                aria-label='Preview message'
                disabled
                placeholder='Ask a question…'
              />
              <Button disabled size='sm' type='button'>
                Send
              </Button>
            </div>
          </div>
        </div>
      )}
      <Button
        aria-expanded={open}
        aria-label={open ? 'Close chat preview' : 'Open chat preview'}
        className={`absolute bottom-4 size-12 rounded-xl text-white ${left ? 'left-4' : 'right-4'}`}
        disabled={!enabled}
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
