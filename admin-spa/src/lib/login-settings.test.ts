import { describe, expect, it } from 'vitest'

import { isGoogleLoginEnabled } from './login-settings'

describe('isGoogleLoginEnabled', () => {
  it('enables Google login from the Vite public env flag', () => {
    expect(
      isGoogleLoginEnabled({
        VITE_PAI_AUTH_GOOGLE_LOGIN_ENABLED: 'true',
      }),
    ).toBe(true)
  })

  it('keeps the legacy Next.js public env flag working during migration', () => {
    expect(
      isGoogleLoginEnabled({
        NEXT_PUBLIC_PAI_AUTH_GOOGLE_LOGIN_ENABLED: 'true',
      }),
    ).toBe(true)
  })

  it('does not enable Google login for unset or non-true values', () => {
    expect(isGoogleLoginEnabled({})).toBe(false)
    expect(
      isGoogleLoginEnabled({
        VITE_PAI_AUTH_GOOGLE_LOGIN_ENABLED: 'false',
      }),
    ).toBe(false)
  })
})
