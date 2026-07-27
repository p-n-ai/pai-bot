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

import { EmbedConfigPanel } from './embed-config-panel'
import type * as AdminAPI from '@/lib/admin-api'

const getEmbedConfig = vi.hoisted(() => vi.fn())
const updateEmbedConfig = vi.hoisted(() => vi.fn())
const addEmbedOrigin = vi.hoisted(() => vi.fn())
const removeEmbedOrigin = vi.hoisted(() => vi.fn())
const writeText = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminAPI>()
  return {
    ...actual,
    addEmbedOrigin,
    getEmbedConfig,
    removeEmbedOrigin,
    updateEmbedConfig,
  }
})

vi.mock('@/auth-provider', () => ({
  useAuth: () => ({
    auth: {
      status: 'authenticated',
      session: {
        expires_at: '2027-01-01T00:00:00Z',
        user: {
          user_id: 'admin-a',
          tenant_id: 'tenant-a',
          tenant_slug: 'school-a',
          role: 'admin',
        },
      },
      error: null,
    },
  }),
}))

const config = {
  id: 'config-a',
  tenant_id: 'tenant-a',
  enabled: true,
  allowed_origins: ['https://school.example'],
  theme_config: {
    color: '#123456',
    language: 'en',
    position: 'bottom-right',
  },
}

describe('EmbedConfigPanel', () => {
  beforeEach(() => {
    getEmbedConfig.mockReset()
    updateEmbedConfig.mockReset()
    addEmbedOrigin.mockReset()
    removeEmbedOrigin.mockReset()
    writeText.mockReset()
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })
  })

  afterEach(cleanup)

  it('renders loading and error states', async () => {
    let rejectLoad: (reason: Error) => void = () => {}
    getEmbedConfig.mockReturnValue(
      new Promise((_, reject) => {
        rejectLoad = reject
      }),
    )
    render(<EmbedConfigPanel />)
    expect(screen.getByText('Loading embed settings')).toBeInTheDocument()
    rejectLoad(new Error('Service unavailable'))
    expect(
      await screen.findByText('Could not load embed settings'),
    ).toBeInTheDocument()
    expect(screen.getByText('Service unavailable')).toBeInTheDocument()
  })

  it('saves enabled and presentation state', async () => {
    getEmbedConfig.mockResolvedValue(config)
    updateEmbedConfig.mockImplementation(async (input) => ({
      ...config,
      enabled: input.enabled,
      theme_config: input.theme_config,
    }))
    render(<EmbedConfigPanel />)
    await screen.findByText('Widget configuration')

    fireEvent.click(screen.getByRole('switch', { name: 'Enable widget' }))
    fireEvent.change(screen.getByLabelText('Language'), {
      target: { value: 'ms' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save configuration' }))

    await waitFor(() =>
      expect(updateEmbedConfig).toHaveBeenCalledWith({
        enabled: false,
        theme_config: {
          color: '#123456',
          language: 'ms',
          position: 'bottom-right',
        },
      }),
    )
  })

  it('adds and removes normalized origin entries through the API', async () => {
    getEmbedConfig.mockResolvedValue(config)
    addEmbedOrigin.mockResolvedValue(undefined)
    removeEmbedOrigin.mockResolvedValue(undefined)
    render(<EmbedConfigPanel />)
    await screen.findByText('Allowed origins')

    fireEvent.change(screen.getByLabelText('Allowed origin'), {
      target: { value: 'https://new.example' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Add origin' }))
    await waitFor(() =>
      expect(addEmbedOrigin).toHaveBeenCalledWith('https://new.example'),
    )

    fireEvent.click(
      screen.getByRole('button', {
        name: 'Remove https://school.example',
      }),
    )
    await waitFor(() =>
      expect(removeEmbedOrigin).toHaveBeenCalledWith('https://school.example'),
    )
  })

  it('copies a tenant-scoped snippet using the current browser origin', async () => {
    getEmbedConfig.mockResolvedValue(config)
    writeText.mockResolvedValue(undefined)
    render(<EmbedConfigPanel />)
    await screen.findByText('Install snippet')
    fireEvent.click(screen.getByRole('button', { name: 'Copy snippet' }))

    await waitFor(() => expect(writeText).toHaveBeenCalledOnce())
    expect(writeText.mock.calls[0][0]).toContain(
      'src="http://localhost:3000/embed/pai-chat.js"',
    )
    expect(writeText.mock.calls[0][0]).toContain('data-tenant="school-a"')
  })

  it('updates the static preview without creating guest auth', async () => {
    getEmbedConfig.mockResolvedValue(config)
    render(<EmbedConfigPanel />)
    const preview = await screen.findByLabelText('Chat preview')
    expect(preview).toBeInTheDocument()
    expect(
      screen.getByText('Static preview. It does not create a guest session.'),
    ).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Position'), {
      target: { value: 'bottom-left' },
    })
    fireEvent.change(screen.getByLabelText('Theme color'), {
      target: { value: '#654321' },
    })
    expect(screen.getByLabelText('Chat preview')).toHaveClass('left-4')
    expect(screen.getByText('P&AI Tutor')).toHaveStyle({
      backgroundColor: '#654321',
    })
    expect(addEmbedOrigin).not.toHaveBeenCalled()
  })
})
