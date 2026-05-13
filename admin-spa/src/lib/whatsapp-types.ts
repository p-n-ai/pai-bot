import { isRecord, isString } from './type-guards'

export interface WhatsAppStatus {
  connected: boolean
  enabled?: boolean
  qr?: string
  qr_image?: string
}

export function isWhatsAppStatus(value: unknown): value is WhatsAppStatus {
  if (!isRecord(value)) {
    return false
  }

  return [hasConnectionState, hasOptionalEnabled, hasOptionalQRCode].every(
    (check) => check(value),
  )
}

function hasConnectionState(value: Record<string, unknown>): boolean {
  return typeof value.connected === 'boolean'
}

function hasOptionalEnabled(value: Record<string, unknown>): boolean {
  return optionalBoolean(value.enabled)
}

function hasOptionalQRCode(value: Record<string, unknown>): boolean {
  return optionalString(value.qr) && optionalString(value.qr_image)
}

function optionalString(value: unknown): boolean {
  return value === undefined || isString(value)
}

function optionalBoolean(value: unknown): boolean {
  return value === undefined || typeof value === 'boolean'
}
