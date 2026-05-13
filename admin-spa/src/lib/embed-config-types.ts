import { isRecord, isString } from './type-guards'

export interface EmbedConfig {
  id: string
  tenant_id: string
  enabled: boolean
  allowed_origins: Array<string>
  theme_config: Record<string, unknown>
  created_at?: string
  updated_at?: string
}

export interface UpdateEmbedConfigInput {
  enabled?: boolean
  theme_config?: Record<string, unknown>
}

export function readEmbedConfig(value: unknown): EmbedConfig | null {
  return isRecord(value) ? readEmbedConfigRecord(value) : null
}

function readRequiredEmbedConfig(
  value: Record<string, unknown>,
): { enabled: boolean; tenantID: string } | null {
  const enabled = readBoolean(readField(value, 'enabled', 'Enabled'))
  const tenantID = readOptionalString(readField(value, 'tenant_id', 'TenantID'))

  return enabled === null || tenantID === undefined
    ? null
    : { enabled, tenantID }
}

function readField(
  value: Record<string, unknown>,
  snakeName: string,
  goName: string,
): unknown {
  return value[snakeName] ?? value[goName]
}

function readEmbedConfigRecord(
  value: Record<string, unknown>,
): EmbedConfig | null {
  const required = readRequiredEmbedConfig(value)

  return required ? buildEmbedConfig(value, required) : null
}

function buildEmbedConfig(
  value: Record<string, unknown>,
  required: { enabled: boolean; tenantID: string },
): EmbedConfig {
  return {
    id: readOptionalString(readField(value, 'id', 'ID')) ?? '',
    tenant_id: required.tenantID,
    enabled: required.enabled,
    allowed_origins: readStringArray(
      readField(value, 'allowed_origins', 'AllowedOrigins'),
    ),
    theme_config: readThemeConfig(
      readField(value, 'theme_config', 'ThemeConfig'),
    ),
    created_at: readOptionalString(readField(value, 'created_at', 'CreatedAt')),
    updated_at: readOptionalString(readField(value, 'updated_at', 'UpdatedAt')),
  }
}

function readBoolean(value: unknown): boolean | null {
  return typeof value === 'boolean' ? value : null
}

function readOptionalString(value: unknown): string | undefined {
  return isString(value) ? value : undefined
}

function readStringArray(value: unknown): Array<string> {
  return Array.isArray(value) && value.every(isString) ? value : []
}

function readThemeConfig(value: unknown): Record<string, unknown> {
  return isRecord(value) ? value : {}
}
