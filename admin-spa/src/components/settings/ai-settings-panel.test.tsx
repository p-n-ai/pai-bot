/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AISettingsPanel } from './ai-settings-panel'
import type * as AdminAPI from '@/lib/admin-api'
import { aiSettingsFixture } from '@/lib/ai-settings-types.test'

const getAISettings = vi.hoisted(() => vi.fn())
const updateAISettings = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminAPI>()

  return {
    ...actual,
    getAISettings,
    updateAISettings,
  }
})

const unsetKeySettings = {
  ...aiSettingsFixture,
  openrouterKey: { set: false, last4: '' },
  sources: { ...aiSettingsFixture.sources, openrouterKey: 'none' },
}

describe('AISettingsPanel', () => {
  beforeEach(() => {
    getAISettings.mockReset()
    updateAISettings.mockReset()
  })

  afterEach(() => {
    cleanup()
  })

  it('saves a new key, clears the input, and shows the masked state', async () => {
    getAISettings.mockResolvedValue(unsetKeySettings)
    updateAISettings.mockResolvedValue({
      ...aiSettingsFixture,
      openrouterKey: { set: true, last4: 'z9y8' },
    })

    render(<AISettingsPanel />)

    const input = await screen.findByPlaceholderText('sk-or-...')
    fireEvent.change(input, { target: { value: 'sk-or-secret' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save key' }))

    expect(await screen.findByText(/configured .+ z9y8/)).toBeInTheDocument()
    expect(updateAISettings).toHaveBeenCalledWith({
      openrouterApiKey: 'sk-or-secret',
    })
    expect(screen.queryByPlaceholderText('sk-or-...')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Replace key' }))
    expect(screen.getByPlaceholderText('sk-or-...')).toHaveValue('')
  })

  it('cancels a key replacement without calling the API', async () => {
    getAISettings.mockResolvedValue(aiSettingsFixture)

    render(<AISettingsPanel />)

    fireEvent.click(await screen.findByRole('button', { name: 'Replace key' }))
    expect(screen.getByPlaceholderText('sk-or-...')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.getByText(/configured .+ a1b2/)).toBeInTheDocument()
    expect(updateAISettings).not.toHaveBeenCalled()
  })

  it('keeps the typed key and shows the error alert when saving fails', async () => {
    getAISettings.mockResolvedValue(unsetKeySettings)
    updateAISettings.mockRejectedValue(new Error('key rejected'))

    render(<AISettingsPanel />)

    const input = await screen.findByPlaceholderText('sk-or-...')
    fireEvent.change(input, { target: { value: 'sk-or-secret' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save key' }))

    expect(await screen.findByText('key rejected')).toBeInTheDocument()
    expect(screen.getByText('API key update failed.')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('sk-or-...')).toHaveValue('sk-or-secret')
  })

  it('marks an env-sourced key and offers no Clear action', async () => {
    getAISettings.mockResolvedValue({
      ...aiSettingsFixture,
      sources: { ...aiSettingsFixture.sources, openrouterKey: 'env' },
    })

    render(<AISettingsPanel />)

    expect(await screen.findByText('from environment')).toBeInTheDocument()
    expect(
      screen.getByText('Set in server environment; clear it there.'),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Clear key' }),
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Replace key' }),
    ).toBeInTheDocument()
  })

  it('resets a db-sourced flag by sending a null override', async () => {
    getAISettings.mockResolvedValue(aiSettingsFixture)
    updateAISettings.mockResolvedValue({
      ...aiSettingsFixture,
      sources: {
        ...aiSettingsFixture.sources,
        flags: { ...aiSettingsFixture.sources.flags, turn_hooks: 'env' },
      },
    })

    render(<AISettingsPanel />)

    const reset = await screen.findByRole('button', {
      name: 'Reset turn_hooks',
    })
    expect(
      screen.queryByRole('button', { name: 'Reset proactive_nudges' }),
    ).not.toBeInTheDocument()

    fireEvent.click(reset)

    await waitFor(() => {
      expect(updateAISettings).toHaveBeenCalledWith({
        flags: { turn_hooks: null },
      })
    })
    await waitFor(() => {
      expect(
        screen.queryByRole('button', { name: 'Reset turn_hooks' }),
      ).not.toBeInTheDocument()
    })
  })
})
