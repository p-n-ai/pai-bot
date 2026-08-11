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
  within,
} from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { EmbedConfigPanel } from './embed-config-panel'
import type * as AdminAPI from '@/lib/admin-api'

const getEmbedConfig = vi.hoisted(() => vi.fn())
const updateEmbedConfig = vi.hoisted(() => vi.fn())
const addEmbedOrigin = vi.hoisted(() => vi.fn())
const removeEmbedOrigin = vi.hoisted(() => vi.fn())
const writeText = vi.hoisted(() => vi.fn())
let rejectLoad: (reason: Error) => void = () => {}

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
  public_embed_base_url: 'https://chat.example',
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
    rejectLoad = () => {}
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })
  })

  afterEach(cleanup)

  it('renders loading and error states', async () => {
    getEmbedConfig.mockReturnValue(
      new Promise((_resolve, reject) => {
        rejectLoad = reject
      }),
    )
    render(<EmbedConfigPanel />)
    expect(
      screen.getByText('Loading website chat settings'),
    ).toBeInTheDocument()
    rejectLoad(new Error('Service unavailable'))
    expect(
      await screen.findByText('Unable to load website chat settings'),
    ).toBeInTheDocument()
    expect(screen.getByText('Service unavailable')).toBeInTheDocument()
  })

  it('explains the ordered setup flow and common recovery steps', async () => {
    getEmbedConfig.mockResolvedValue(config)
    render(<EmbedConfigPanel />)

    await screen.findByRole('heading', { name: 'Setup guide' })
    const steps = within(
      screen.getByRole('list', { name: 'Website chat setup steps' }),
    ).getAllByRole('listitem')
    expect(steps).toHaveLength(5)
    expect(steps.map((step) => step.textContent)).toEqual([
      expect.stringContaining('Approve a website'),
      expect.stringContaining('Configure appearance'),
      expect.stringContaining('Show the chat widget'),
      expect.stringContaining('Install website chat'),
      expect.stringContaining('Test website chat'),
    ])
    expect(
      screen.getByRole('heading', {
        name: 'Website address does not match',
      }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Chat widget remains hidden' }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/website’s protocol, host, and port/i),
    ).toBeInTheDocument()
  })

  it('saves enabled and presentation state', async () => {
    getEmbedConfig.mockResolvedValue(config)
    updateEmbedConfig.mockImplementation((input) =>
      Promise.resolve({
        ...config,
        enabled: input.enabled,
        theme_config: input.theme_config,
      }),
    )
    render(<EmbedConfigPanel />)
    await screen.findByText('Chat appearance')

    fireEvent.click(screen.getByRole('switch', { name: 'Show chat widget' }))
    fireEvent.change(screen.getByLabelText('Language'), {
      target: { value: 'ms' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save website chat' }))

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
    expect(screen.getByRole('status')).toHaveTextContent(
      'Website chat settings saved.',
    )
  })

  it('adds and removes normalized origin entries through the API', async () => {
    getEmbedConfig.mockResolvedValue(config)
    addEmbedOrigin.mockResolvedValue(undefined)
    removeEmbedOrigin.mockResolvedValue(undefined)
    render(<EmbedConfigPanel />)
    await screen.findByText('Approved websites')

    fireEvent.change(screen.getByLabelText('Website origin'), {
      target: { value: '  https://new.example  ' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Add website' }))
    await waitFor(() =>
      expect(addEmbedOrigin).toHaveBeenCalledWith('https://new.example'),
    )

    fireEvent.click(
      screen.getByRole('button', {
        name: 'Remove website https://school.example',
      }),
    )
    await waitFor(() =>
      expect(removeEmbedOrigin).toHaveBeenCalledWith('https://school.example'),
    )
  })

  it('places origin failures beside the origin controls and clears them after recovery', async () => {
    getEmbedConfig.mockResolvedValue(config)
    addEmbedOrigin
      .mockRejectedValueOnce(new Error('Origin rejected'))
      .mockResolvedValueOnce(undefined)
    render(<EmbedConfigPanel />)
    const originsTitle = await screen.findByText('Approved websites')
    const originsSection = originsTitle.closest('section')
    if (!originsSection) {
      throw new Error('allowed origins section not found')
    }

    fireEvent.change(screen.getByLabelText('Website origin'), {
      target: { value: 'https://new.example' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Add website' }))

    expect(
      await within(originsSection).findByText(
        'Unable to update approved websites',
      ),
    ).toBeInTheDocument()
    expect(
      within(originsSection).getByText('Origin rejected'),
    ).toBeInTheDocument()
    expect(
      screen.queryByText('Unable to save website chat settings'),
    ).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Add website' }))
    await waitFor(() =>
      expect(
        within(originsSection).queryByText('Origin rejected'),
      ).not.toBeInTheDocument(),
    )
  })

  it('copies a tenant-scoped snippet using the backend public embed origin', async () => {
    getEmbedConfig.mockResolvedValue(config)
    writeText.mockResolvedValue(undefined)
    render(<EmbedConfigPanel />)
    await screen.findByRole('heading', {
      level: 2,
      name: 'Install website chat',
    })
    fireEvent.click(screen.getByRole('button', { name: 'Copy install code' }))

    await waitFor(() => expect(writeText).toHaveBeenCalledOnce())
    expect(writeText.mock.calls[0][0]).toContain(
      'src="https://chat.example/embed/pai-chat.js"',
    )
    expect(writeText.mock.calls[0][0]).toContain('data-tenant="school-a"')

    fireEvent.change(screen.getByLabelText('Language'), {
      target: { value: 'zh' },
    })
    expect(
      screen.getByRole('button', { name: 'Copy install code' }),
    ).toBeInTheDocument()
  })

  it('reports clipboard rejection', async () => {
    getEmbedConfig.mockResolvedValue(config)
    writeText.mockRejectedValue(new Error('Clipboard permission denied'))
    render(<EmbedConfigPanel />)
    await screen.findByRole('heading', {
      level: 2,
      name: 'Install website chat',
    })
    fireEvent.click(screen.getByRole('button', { name: 'Copy install code' }))

    expect(
      await screen.findByText('Unable to copy install code'),
    ).toBeInTheDocument()
    expect(screen.getByText('Clipboard permission denied')).toBeInTheDocument()
  })

  it('offers manual recovery when clipboard access is unavailable', async () => {
    getEmbedConfig.mockResolvedValue(config)
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: undefined,
    })
    render(<EmbedConfigPanel />)
    await screen.findByRole('heading', {
      level: 2,
      name: 'Install website chat',
    })
    fireEvent.click(screen.getByRole('button', { name: 'Copy install code' }))

    expect(
      await screen.findByText('Unable to copy install code'),
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Clipboard access is unavailable. Select and copy the install code manually.',
      ),
    ).toBeInTheDocument()
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
    fireEvent.change(screen.getByLabelText('Language'), {
      target: { value: 'ms' },
    })
    expect(
      screen.getByText('Hai! Apakah yang ingin anda pelajari hari ini?'),
    ).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Tanya soalan…')).toBeInTheDocument()
    expect(screen.getByText('Hantar')).toBeInTheDocument()
    expect(screen.getByText('P&AI Tutor')).toHaveStyle({
      color: 'oklch(1 0 0)',
    })
    expect(addEmbedOrigin).not.toHaveBeenCalled()
  })
})
