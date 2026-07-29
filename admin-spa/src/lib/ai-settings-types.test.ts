import { describe, expect, it } from 'vitest'

import { readAISettings } from './ai-settings-types'

export const aiSettingsFixture = {
  defaultProvider: 'openai',
  openrouterModel: 'anthropic/claude-sonnet-4.5',
  openrouterKey: {
    set: true,
    last4: 'a1b2',
  },
  flags: {
    turn_hooks: true,
    proactive_nudges: false,
  },
  availableProviders: ['openai', 'openrouter'],
  providers: [
    {
      name: 'openai',
      supported: true,
      configured: true,
      registrable: true,
      effective: true,
      managedBy: 'environment',
    },
    {
      name: 'openrouter',
      supported: true,
      configured: true,
      registrable: true,
      effective: false,
      managedBy: 'runtime',
    },
  ],
  health: {
    revision: 3,
    appliedRevision: 3,
    drift: false,
    openrouterKey: {
      stored: true,
      readable: true,
      version: 'v1',
      algorithm: 'a256gcm',
      keyId: 'test-safe-key-id',
      migrationNeeded: false,
    },
  },
  sources: {
    defaultProvider: 'db',
    openrouterModel: 'env',
    openrouterKey: 'db',
    flags: {
      turn_hooks: 'db',
      proactive_nudges: 'env',
    },
  },
  baseline: {
    defaultProvider: 'openai',
    openrouterModel: 'anthropic/claude-sonnet-4.5',
    openrouterKey: { set: false, last4: '' },
    flags: {
      turn_hooks: false,
      proactive_nudges: false,
    },
  },
  override: {
    defaultProvider: 'openai',
    openrouterModel: null,
    openrouterKey: { set: true, last4: 'a1b2' },
    flags: {
      turn_hooks: true,
    },
  },
  effective: {
    defaultProvider: 'openai',
    openrouterModel: 'anthropic/claude-sonnet-4.5',
    openrouterKey: { set: true, last4: 'a1b2' },
    flags: {
      turn_hooks: true,
      proactive_nudges: false,
    },
  },
  revision: 3,
  appliedRevision: 3,
  drift: false,
}

describe('AI settings response guard', () => {
  it('accepts the backend AI settings shape', () => {
    expect(readAISettings(aiSettingsFixture)).toEqual(aiSettingsFixture)
  })

  it('accepts an unset key and empty flag map', () => {
    expect(
      readAISettings({
        ...aiSettingsFixture,
        openrouterKey: { set: false, last4: '' },
        flags: {},
      }),
    ).toEqual({
      ...aiSettingsFixture,
      openrouterKey: { set: false, last4: '' },
      flags: {},
    })
  })

  it('rejects non-object payloads', () => {
    expect(readAISettings(null)).toBeNull()
    expect(readAISettings(undefined)).toBeNull()
    expect(readAISettings('settings')).toBeNull()
  })

  it('rejects payloads with missing or mistyped fields', () => {
    expect(
      readAISettings({ ...aiSettingsFixture, defaultProvider: 7 }),
    ).toBeNull()
    expect(
      readAISettings({ ...aiSettingsFixture, openrouterModel: undefined }),
    ).toBeNull()
    expect(
      readAISettings({ ...aiSettingsFixture, openrouterKey: {} }),
    ).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        openrouterKey: { set: 'yes', last4: 'a1b2' },
      }),
    ).toBeNull()
    expect(
      readAISettings({ ...aiSettingsFixture, flags: { turn_hooks: 'on' } }),
    ).toBeNull()
    expect(readAISettings({ ...aiSettingsFixture, flags: null })).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        availableProviders: ['openai', 3],
      }),
    ).toBeNull()
    expect(
      readAISettings({ ...aiSettingsFixture, availableProviders: 'openai' }),
    ).toBeNull()
  })

  it('rejects payloads with missing or mistyped sources', () => {
    expect(
      readAISettings({ ...aiSettingsFixture, sources: undefined }),
    ).toBeNull()
    expect(readAISettings({ ...aiSettingsFixture, sources: 'db' })).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        sources: { ...aiSettingsFixture.sources, openrouterKey: 7 },
      }),
    ).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        sources: { ...aiSettingsFixture.sources, flags: { turn_hooks: true } },
      }),
    ).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        sources: { ...aiSettingsFixture.sources, flags: null },
      }),
    ).toBeNull()
  })

  it('requires redacted baseline, override, and effective projections', () => {
    expect(
      readAISettings({ ...aiSettingsFixture, baseline: undefined }),
    ).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        override: {
          ...aiSettingsFixture.override,
          openrouterKey: { set: true, last4: 1234 },
        },
      }),
    ).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        override: {
          ...aiSettingsFixture.override,
          defaultProvider: null,
        },
      }),
    ).not.toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        effective: {
          ...aiSettingsFixture.effective,
          flags: { turn_hooks: 'true' },
        },
      }),
    ).toBeNull()
  })
})
