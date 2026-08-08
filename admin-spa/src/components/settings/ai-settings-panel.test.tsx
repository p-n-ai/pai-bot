/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AISettingsPanel } from './ai-settings-panel'
import type * as AdminAPI from '@/lib/admin-api'
import type { AISettings } from '@/lib/ai-settings-types'
import { AdminAPIError } from '@/lib/admin-api'
import { aiSettingsFixture } from '@/lib/ai-settings-types.test'

const getAISettings = vi.hoisted(() => vi.fn())
const getCodexAuthStatus = vi.hoisted(() => vi.fn())
const startCodexDeviceAuth = vi.hoisted(() => vi.fn())
const updateAISettings = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminAPI>()
  return {
    ...actual,
    getAISettings,
    getCodexAuthStatus,
    startCodexDeviceAuth,
    updateAISettings,
  }
})

function providerSection(name: string) {
  return screen.getByRole('region', { name })
}

function responseWith(
  update: (settings: typeof aiSettingsFixture) => unknown,
): unknown {
  return update(aiSettingsFixture)
}

describe('AISettingsPanel', () => {
  beforeEach(() => {
    window.HTMLElement.prototype.scrollIntoView = vi.fn()
    getAISettings.mockReset()
    getAISettings.mockResolvedValue(aiSettingsFixture)
    getCodexAuthStatus.mockReset()
    getCodexAuthStatus.mockResolvedValue({
      state: 'disconnected',
      verificationUrl: '',
      userCode: '',
      message: '',
    })
    startCodexDeviceAuth.mockReset()
    updateAISettings.mockReset()
    updateAISettings.mockResolvedValue(aiSettingsFixture)
  })

  afterEach(cleanup)

  it('renders every closed provider with only its legal controls', async () => {
    render(<AISettingsPanel />)

    expect(await screen.findByText('Default provider')).toBeInTheDocument()
    for (const label of [
      'OpenAI provider',
      'Anthropic provider',
      'DeepSeek provider',
      'Groq provider',
      'xAI provider',
      'Mistral provider',
      'Cerebras provider',
      'Google provider',
      'OpenRouter provider',
      'Ollama provider',
      'Managed Codex provider',
    ]) {
      expect(providerSection(label)).toBeInTheDocument()
    }

    expect(
      within(providerSection('Ollama provider')).queryByText('API key'),
    ).not.toBeInTheDocument()
    expect(
      within(providerSection('Managed Codex provider')).queryByText('API key'),
    ).not.toBeInTheDocument()
    expect(
      within(providerSection('Ollama provider')).queryByLabelText(/URL/i),
    ).not.toBeInTheDocument()
  })

  it('switches the default with a closed provider selector', async () => {
    render(<AISettingsPanel />)
    const section = await screen.findByRole('region', {
      name: 'Default AI provider',
    })
    fireEvent.click(within(section).getByRole('combobox'))
    fireEvent.click(await screen.findByRole('option', { name: 'Ollama' }))

    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        defaultProvider: { type: 'ollama' },
        expectedRevision: 3,
      })
    })
  })

  it.each([
    ['Groq', 'groq'],
    ['xAI', 'xai'],
    ['Mistral', 'mistral'],
    ['Cerebras', 'cerebras'],
  ] as const)('selects %s as the default provider', async (label, name) => {
    render(<AISettingsPanel />)
    const section = await screen.findByRole('region', {
      name: 'Default AI provider',
    })
    fireEvent.click(within(section).getByRole('combobox'))
    fireEvent.click(await screen.findByRole('option', { name: label }))

    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        defaultProvider: { type: 'api_key', name },
        expectedRevision: 3,
      })
    })
  })

  it('saves and resets an API-key provider model through its variant', async () => {
    render(<AISettingsPanel />)
    const section = await screen.findByRole('region', {
      name: 'OpenRouter provider',
    })
    const input = within(section).getByRole('textbox', {
      name: 'OpenRouter model',
    })

    fireEvent.change(input, { target: { value: ' openrouter/new-model ' } })
    fireEvent.click(within(section).getByRole('button', { name: 'Save model' }))
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        expectedRevision: 3,
        provider: {
          type: 'api_key',
          name: 'openrouter',
          model: 'openrouter/new-model',
        },
      })
    })

    fireEvent.click(
      within(section).getByRole('button', {
        name: 'Use environment model',
      }),
    )
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        expectedRevision: 3,
        provider: {
          type: 'api_key',
          name: 'openrouter',
          model: null,
        },
      })
    })
  })

  it('replaces and resets an API key without retaining plaintext', async () => {
    render(<AISettingsPanel />)
    const section = await screen.findByRole('region', {
      name: 'OpenRouter provider',
    })
    fireEvent.click(
      within(section).getByRole('button', { name: 'Replace key' }),
    )
    const input = within(section).getByLabelText('OpenRouter API key')
    fireEvent.change(input, { target: { value: 'sk-or-secret' } })
    fireEvent.click(within(section).getByRole('button', { name: 'Save key' }))

    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        expectedRevision: 3,
        provider: {
          type: 'api_key',
          name: 'openrouter',
          apiKey: 'sk-or-secret',
        },
      })
    })
    expect(
      within(section).queryByDisplayValue('sk-or-secret'),
    ).not.toBeInTheDocument()

    fireEvent.click(
      within(section).getByRole('button', {
        name: 'Use environment key',
      }),
    )
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        expectedRevision: 3,
        provider: {
          type: 'api_key',
          name: 'openrouter',
          apiKey: null,
        },
      })
    })
  })

  it('keeps an API key input when the write fails', async () => {
    updateAISettings.mockRejectedValue(new Error('key rejected'))
    render(<AISettingsPanel />)
    const section = await screen.findByRole('region', {
      name: 'OpenRouter provider',
    })
    fireEvent.click(
      within(section).getByRole('button', { name: 'Replace key' }),
    )
    const input = within(section).getByLabelText('OpenRouter API key')
    fireEvent.change(input, { target: { value: 'sk-or-secret' } })
    fireEvent.click(within(section).getByRole('button', { name: 'Save key' }))

    expect(await within(section).findByText('key rejected')).toBeInTheDocument()
    expect(input).toHaveValue('sk-or-secret')
  })

  it('does not offer a reset action for an environment-owned key', async () => {
    getAISettings.mockResolvedValue(
      responseWith((settings) => ({
        ...settings,
        providers: settings.providers.map((provider) =>
          provider.type === 'api_key' && provider.name === 'openrouter'
            ? {
                ...provider,
                credential: {
                  ...provider.credential,
                  source: 'env',
                  override: { set: false, last4: '' },
                },
              }
            : provider,
        ),
      })),
    )
    render(<AISettingsPanel />)
    const section = await screen.findByRole('region', {
      name: 'OpenRouter provider',
    })
    expect(within(section).getByText('from environment')).toBeInTheDocument()
    expect(
      within(section).queryByRole('button', {
        name: 'Use environment key',
      }),
    ).not.toBeInTheDocument()
  })

  it('updates and resets Ollama without exposing a custom endpoint', async () => {
    const dbSettings = responseWith((settings) => ({
      ...settings,
      providers: settings.providers.map((provider) =>
        provider.type === 'ollama'
          ? {
              ...provider,
              enabled: {
                ...provider.enabled,
                override: true,
                source: 'db',
              },
              model: { ...provider.model, override: 'llama3', source: 'db' },
            }
          : provider,
      ),
    }))
    getAISettings.mockResolvedValue(dbSettings)
    updateAISettings.mockResolvedValue(dbSettings)
    render(<AISettingsPanel />)
    const section = await screen.findByRole('region', {
      name: 'Ollama provider',
    })

    fireEvent.click(
      within(section).getByRole('button', {
        name: 'Use environment setting',
      }),
    )
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        expectedRevision: 3,
        provider: { type: 'ollama', enabled: null },
      })
    })
    fireEvent.click(
      within(section).getByRole('button', {
        name: 'Use environment model',
      }),
    )
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        expectedRevision: 3,
        provider: { type: 'ollama', model: null },
      })
    })
    fireEvent.click(
      within(section).getByRole('button', { name: 'Disable Ollama' }),
    )
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        expectedRevision: 3,
        provider: { type: 'ollama', enabled: false },
      })
    })
    expect(within(section).queryByLabelText(/URL/i)).not.toBeInTheDocument()
  })

  it('updates managed Codex model and starts device authorization', async () => {
    getAISettings.mockResolvedValue(
      responseWith((settings) => ({
        ...settings,
        providers: settings.providers.map((provider) =>
          provider.type === 'managed_codex'
            ? { ...provider, model: { ...provider.model, source: 'db' } }
            : provider,
        ),
      })),
    )
    startCodexDeviceAuth.mockResolvedValue({
      state: 'awaiting_authorization',
      verificationUrl: 'https://auth.openai.com/codex/device',
      userCode: 'ABCD-1234',
      message: '',
    })
    render(<AISettingsPanel />)
    const section = await screen.findByRole('region', {
      name: 'Managed Codex provider',
    })
    fireEvent.click(
      within(section).getByRole('button', {
        name: 'Use environment model',
      }),
    )
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        expectedRevision: 3,
        provider: { type: 'managed_codex', model: null },
      })
    })

    fireEvent.click(
      within(section).getByRole('button', { name: 'Connect Codex' }),
    )
    expect(await within(section).findByText('ABCD-1234')).toBeInTheDocument()
    expect(
      within(section).getByRole('link', {
        name: 'Open Codex verification page',
      }),
    ).toHaveAttribute('href', 'https://auth.openai.com/codex/device')
  })

  it('hides managed Codex auth when the server capability is absent', async () => {
    getCodexAuthStatus.mockRejectedValue(
      new AdminAPIError('Request failed: 404', 404),
    )
    render(<AISettingsPanel />)
    const section = await screen.findByRole('region', {
      name: 'Managed Codex provider',
    })
    await act(async () => Promise.resolve())
    expect(
      within(section).queryByRole('button', { name: 'Connect Codex' }),
    ).not.toBeInTheDocument()
  })

  it('uses null to reset default-provider and feature-flag overrides', async () => {
    render(<AISettingsPanel />)
    fireEvent.click(
      await screen.findByRole('button', { name: 'Use environment default' }),
    )
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        defaultProvider: null,
        expectedRevision: 3,
      })
    })

    fireEvent.click(screen.getByRole('button', { name: 'Reset turn_hooks' }))
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        expectedRevision: 3,
        flags: { turn_hooks: null },
      })
    })
  })

  it('rejects a blank model before calling the API', async () => {
    render(<AISettingsPanel />)
    const section = await screen.findByRole('region', {
      name: 'OpenRouter provider',
    })
    fireEvent.change(
      within(section).getByRole('textbox', { name: 'OpenRouter model' }),
      { target: { value: '   ' } },
    )
    fireEvent.click(within(section).getByRole('button', { name: 'Save model' }))

    expect(await within(section).findByText('Model is required.')).toBeVisible()
    expect(updateAISettings).not.toHaveBeenCalled()
  })

  it('serializes cross-section saves against the latest confirmed revision', async () => {
    const flagSave = deferred<AISettings>()
    const revision4 = {
      ...aiSettingsFixture,
      revision: 4,
      appliedRevision: 4,
    } as unknown as AISettings
    const revision5 = {
      ...aiSettingsFixture,
      revision: 5,
      appliedRevision: 5,
    } as unknown as AISettings
    updateAISettings
      .mockImplementationOnce(() => flagSave.promise)
      .mockResolvedValueOnce(revision5)
    render(<AISettingsPanel />)
    fireEvent.click(await screen.findByRole('button', { name: 'Disable' }))
    const section = providerSection('OpenRouter provider')
    fireEvent.click(within(section).getByRole('button', { name: 'Save model' }))

    await act(async () => Promise.resolve())
    expect(updateAISettings).toHaveBeenCalledTimes(1)
    await act(async () => {
      flagSave.resolve(revision4)
      await flagSave.promise
    })
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenNthCalledWith(2, {
        expectedRevision: 4,
        provider: {
          type: 'api_key',
          name: 'openrouter',
          model: 'baseline-model',
        },
      })
    })
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Disable' })).toBeEnabled()
      expect(
        within(section).getByRole('button', { name: 'Save model' }),
      ).toBeEnabled()
    })
  })

  it('refreshes the revision after a conflict before the next save', async () => {
    const revision7 = {
      ...aiSettingsFixture,
      revision: 7,
      appliedRevision: 7,
    } as unknown as AISettings
    const revision8 = {
      ...aiSettingsFixture,
      revision: 8,
      appliedRevision: 8,
    } as unknown as AISettings
    getAISettings
      .mockResolvedValueOnce(aiSettingsFixture)
      .mockResolvedValueOnce(revision7)
    updateAISettings
      .mockRejectedValueOnce(new AdminAPIError('revision conflict', 409))
      .mockResolvedValueOnce(revision8)

    render(<AISettingsPanel />)
    fireEvent.click(await screen.findByRole('button', { name: 'Disable' }))
    expect(await screen.findByText('revision conflict')).toBeInTheDocument()
    await waitFor(() => expect(getAISettings).toHaveBeenCalledTimes(2))

    fireEvent.click(screen.getByRole('button', { name: 'Disable' }))
    await waitFor(() => {
      expect(updateAISettings).toHaveBeenNthCalledWith(2, {
        expectedRevision: 7,
        flags: { turn_hooks: false },
      })
    })
  })
})

function deferred<T>() {
  let deferredResolve!: (value: T) => void
  const promise = new Promise<T>((resolve) => {
    deferredResolve = resolve
  })
  return { promise, resolve: deferredResolve }
}
