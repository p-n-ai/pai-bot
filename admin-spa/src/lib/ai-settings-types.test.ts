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
})
