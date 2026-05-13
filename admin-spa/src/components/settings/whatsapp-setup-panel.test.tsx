/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { WhatsAppSetupPanel } from './whatsapp-setup-panel'
import type * as AdminAPI from '@/lib/admin-api'

const getWhatsAppStatus = vi.hoisted(() => vi.fn())
const disconnectWhatsApp = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminAPI>()

  return {
    ...actual,
    disconnectWhatsApp,
    getWhatsAppStatus,
  }
})

describe('WhatsAppSetupPanel', () => {
  beforeEach(() => {
    getWhatsAppStatus.mockReset()
    disconnectWhatsApp.mockReset()
  })

  afterEach(() => {
    cleanup()
  })

  it('shows the QR setup state from the WhatsApp status endpoint', async () => {
    getWhatsAppStatus.mockResolvedValue({
      connected: false,
      qr_image: 'data:image/png;base64,abc',
    })

    render(<WhatsAppSetupPanel />)

    expect(
      await screen.findByRole('img', { name: 'WhatsApp QR Code' }),
    ).toHaveAttribute('src', 'data:image/png;base64,abc')
    expect(
      screen.getByText('Scan the QR code below with your phone to connect.'),
    ).toBeInTheDocument()
    expect(screen.getByText(/Settings/i)).toBeInTheDocument()
    expect(screen.getByText(/Linked Devices/i)).toBeInTheDocument()
    expect(screen.getByText(/Link a Device/i)).toBeInTheDocument()
    expect(
      screen.getByText('Page refreshes automatically every 5 seconds.'),
    ).toBeInTheDocument()
  })

  it('shows the source waiting state while the QR code is unavailable', async () => {
    getWhatsAppStatus.mockResolvedValue({
      connected: false,
    })

    render(<WhatsAppSetupPanel />)

    expect(
      await screen.findByLabelText('Waiting for QR code'),
    ).toBeInTheDocument()
    expect(
      screen.getByText('Waiting for QR code from server...'),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })

  it('disconnects an active WhatsApp session', async () => {
    getWhatsAppStatus.mockResolvedValue({
      connected: true,
    })
    disconnectWhatsApp.mockResolvedValue(undefined)

    render(<WhatsAppSetupPanel />)

    const button = await screen.findByRole('button', { name: 'Disconnect' })
    expect(screen.getByLabelText('WhatsApp connected')).toBeInTheDocument()
    expect(screen.getByLabelText('WhatsApp active session')).toBeInTheDocument()
    expect(screen.getByText('Session active')).toBeInTheDocument()
    expect(
      screen.getByText('Messages are being sent and received via WhatsApp.'),
    ).toBeInTheDocument()
    button.click()

    await waitFor(() => {
      expect(disconnectWhatsApp).toHaveBeenCalledOnce()
    })
  })
})
