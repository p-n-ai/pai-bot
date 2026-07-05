import { isRecord, isString } from './type-guards'

export interface AISettingsKeyStatus {
  set: boolean
  last4: string
}

export interface AISettings {
  defaultProvider: string
  openrouterModel: string
  openrouterKey: AISettingsKeyStatus
  flags: Record<string, boolean>
  availableProviders: Array<string>
}

export interface UpdateAISettingsInput {
  defaultProvider?: string
  openrouterModel?: string
  openrouterApiKey?: string
  flags?: Record<string, boolean>
}

export function readAISettings(value: unknown): AISettings | null {
  return isRecord(value) ? readAISettingsRecord(value) : null
}

function readAISettingsRecord(
  value: Record<string, unknown>,
): AISettings | null {
  const openrouterKey = readKeyStatus(value.openrouterKey)
  const flags = readFlags(value.flags)
  const availableProviders = readProviderList(value.availableProviders)

  if (
    !isString(value.defaultProvider) ||
    !isString(value.openrouterModel) ||
    openrouterKey === null ||
    flags === null ||
    availableProviders === null
  ) {
    return null
  }

  return {
    defaultProvider: value.defaultProvider,
    openrouterModel: value.openrouterModel,
    openrouterKey,
    flags,
    availableProviders,
  }
}

function readKeyStatus(value: unknown): AISettingsKeyStatus | null {
  if (!isRecord(value)) {
    return null
  }

  return typeof value.set === 'boolean' && isString(value.last4)
    ? { set: value.set, last4: value.last4 }
    : null
}

function readFlags(value: unknown): Record<string, boolean> | null {
  if (!isRecord(value)) {
    return null
  }

  const flags: Record<string, boolean> = {}

  for (const [name, enabled] of Object.entries(value)) {
    if (typeof enabled !== 'boolean') {
      return null
    }

    flags[name] = enabled
  }

  return flags
}

function readProviderList(value: unknown): Array<string> | null {
  return Array.isArray(value) && value.every(isString) ? value : null
}
