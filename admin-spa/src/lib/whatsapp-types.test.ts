import { describe, expect, it } from 'vitest'

import { isWhatsAppStatus } from './whatsapp-types'

describe('isWhatsAppStatus', () => {
  it('accepts connected and QR status payloads', () => {
    expect(isWhatsAppStatus({ connected: true })).toBe(true)
    expect(isWhatsAppStatus({ connected: false, enabled: false })).toBe(true)
    expect(
      isWhatsAppStatus({
        connected: false,
        qr: 'raw-qr',
        qr_image: 'data:image/png;base64,abc',
      }),
    ).toBe(true)
  })

  it('rejects malformed status payloads', () => {
    expect(isWhatsAppStatus({ connected: 'true' })).toBe(false)
    expect(isWhatsAppStatus({ connected: false, qr_image: 123 })).toBe(false)
  })
})
