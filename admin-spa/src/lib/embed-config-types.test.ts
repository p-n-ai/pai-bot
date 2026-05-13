import { describe, expect, it } from 'vitest'

import { readEmbedConfig } from './embed-config-types'

describe('embed config response guard', () => {
  it('normalizes backend embed config field names', () => {
    expect(
      readEmbedConfig({
        TenantID: 'tenant_1',
        Enabled: true,
        AllowedOrigins: ['https://school.example'],
        ThemeConfig: {
          color: '#0f172a',
        },
      }),
    ).toEqual({
      id: '',
      tenant_id: 'tenant_1',
      enabled: true,
      allowed_origins: ['https://school.example'],
      theme_config: {
        color: '#0f172a',
      },
      created_at: undefined,
      updated_at: undefined,
    })
  })

  it('rejects responses without tenant and enabled fields', () => {
    expect(readEmbedConfig({ tenant_id: 'tenant_1' })).toBeNull()
    expect(readEmbedConfig({ enabled: true })).toBeNull()
    expect(readEmbedConfig(null)).toBeNull()
  })
})
