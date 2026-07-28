import { describe, expect, it } from 'vitest'

import { readCodexAuthStatus } from './codex-auth-types'

describe('readCodexAuthStatus', () => {
  it('accepts a bounded device authorization status', () => {
    expect(
      readCodexAuthStatus({
        state: 'awaiting_authorization',
        verificationUrl: 'https://auth.openai.com/codex/device',
        userCode: 'ABCD-1234',
      }),
    ).toEqual({
      state: 'awaiting_authorization',
      verificationUrl: 'https://auth.openai.com/codex/device',
      userCode: 'ABCD-1234',
      message: '',
    })
  })

  it('rejects unknown states and malformed fields', () => {
    expect(readCodexAuthStatus({ state: 'authorized' })).toBeNull()
    expect(
      readCodexAuthStatus({ state: 'connected', message: { secret: true } }),
    ).toBeNull()
  })
})
