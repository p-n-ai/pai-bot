import { describe, expect, it } from 'vitest'

import { readAISettings } from './ai-settings-types'

const readiness = {
  supported: true,
  configured: true,
  registrable: true,
  effective: false,
  managedBy: 'runtime',
}

const model = {
  baseline: 'baseline-model',
  override: null,
  effective: 'baseline-model',
  source: 'env',
}

const credential = {
  baseline: { set: false, last4: '' },
  override: { set: true, last4: 'a1b2' },
  effective: { set: true, last4: 'a1b2' },
  source: 'db',
  health: {
    stored: true,
    readable: true,
    version: 'v1',
    algorithm: 'a256gcm',
    keyId: 'test-safe-key-id',
    migrationNeeded: false,
  },
}

export const aiSettingsFixture = {
  defaultProvider: {
    baseline: { type: 'api_key', name: 'openai' },
    override: { type: 'api_key', name: 'openrouter' },
    effective: { type: 'api_key', name: 'openrouter' },
    source: 'db',
  },
  providers: [
    {
      type: 'api_key',
      name: 'openai',
      model,
      credential,
      readiness,
    },
    {
      type: 'api_key',
      name: 'anthropic',
      model,
      credential,
      readiness,
    },
    {
      type: 'api_key',
      name: 'deepseek',
      model,
      credential,
      readiness,
    },
    {
      type: 'api_key',
      name: 'google',
      model,
      credential,
      readiness,
    },
    {
      type: 'api_key',
      name: 'openrouter',
      model: { ...model, override: 'runtime-model', source: 'db' },
      credential,
      readiness: { ...readiness, effective: true },
    },
    {
      type: 'ollama',
      enabled: {
        baseline: true,
        override: null,
        effective: true,
        source: 'env',
      },
      model,
      readiness,
    },
    {
      type: 'managed_codex',
      enabled: {
        baseline: true,
        override: null,
        effective: true,
        source: 'env',
      },
      model,
      readiness,
    },
  ],
  flags: {
    baseline: { turn_hooks: false, proactive_nudges: false },
    override: { turn_hooks: true },
    effective: { turn_hooks: true, proactive_nudges: false },
    sources: { turn_hooks: 'db', proactive_nudges: 'env' },
  },
  revision: 3,
  appliedRevision: 3,
  drift: false,
} as const

describe('AI settings response guard', () => {
  it('accepts all three closed provider variants', () => {
    expect(readAISettings(aiSettingsFixture)).toEqual(aiSettingsFixture)
  })

  it('accepts nullable reconciliation values without weakening identity', () => {
    expect(
      readAISettings({
        ...aiSettingsFixture,
        defaultProvider: {
          baseline: null,
          override: null,
          effective: null,
          source: 'none',
        },
        providers: aiSettingsFixture.providers.map((provider) =>
          provider.type === 'api_key'
            ? {
                ...provider,
                model: {
                  baseline: null,
                  override: null,
                  effective: null,
                  source: 'none',
                },
              }
            : provider,
        ),
      }),
    ).not.toBeNull()
  })

  it('rejects unknown, partial, and cross-provider variants', () => {
    const replaceProvider = (provider: unknown) => ({
      ...aiSettingsFixture,
      providers: [provider, ...aiSettingsFixture.providers.slice(1)],
    })

    expect(
      readAISettings(replaceProvider({ type: 'api_key', model, credential })),
    ).toBeNull()
    expect(
      readAISettings(
        replaceProvider({
          ...aiSettingsFixture.providers[0],
          type: 'bedrock',
        }),
      ),
    ).toBeNull()
    expect(
      readAISettings(
        replaceProvider({
          ...aiSettingsFixture.providers[0],
          name: 'azure',
        }),
      ),
    ).toBeNull()
    expect(
      readAISettings(
        replaceProvider({
          ...aiSettingsFixture.providers[5],
          credential,
        }),
      ),
    ).toBeNull()
    expect(
      readAISettings(
        replaceProvider({
          ...aiSettingsFixture.providers[6],
          apiKey: 'must-never-appear',
        }),
      ),
    ).toBeNull()
  })

  it('rejects excess fields at top-level and nested boundaries', () => {
    expect(
      readAISettings({ ...aiSettingsFixture, openrouterModel: 'legacy' }),
    ).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        defaultProvider: {
          ...aiSettingsFixture.defaultProvider,
          legacyName: 'openrouter',
        },
      }),
    ).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        providers: aiSettingsFixture.providers.map((provider, index) =>
          index === 0
            ? {
                ...provider,
                credential: {
                  ...credential,
                  plaintext: 'must-never-appear',
                },
              }
            : provider,
        ),
      }),
    ).toBeNull()
  })

  it('rejects malformed projections and ownership sources', () => {
    expect(readAISettings(null)).toBeNull()
    expect(readAISettings('settings')).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        defaultProvider: {
          ...aiSettingsFixture.defaultProvider,
          source: 'client',
        },
      }),
    ).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        providers: aiSettingsFixture.providers.map((provider, index) =>
          index === 0
            ? { ...provider, model: { ...model, effective: 7 } }
            : provider,
        ),
      }),
    ).toBeNull()
    expect(
      readAISettings({
        ...aiSettingsFixture,
        flags: {
          ...aiSettingsFixture.flags,
          effective: { turn_hooks: 'yes' },
        },
      }),
    ).toBeNull()
  })
})
