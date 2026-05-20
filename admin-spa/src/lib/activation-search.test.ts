import { describe, expect, it } from 'vitest'

import { parseActivationSearch } from './activation-search'

describe('parseActivationSearch', () => {
  it('keeps token search values as strings', () => {
    expect(parseActivationSearch({ token: 'invite-token' })).toEqual({
      token: 'invite-token',
    })
  })

  it('drops malformed token search values', () => {
    expect(parseActivationSearch({ token: ['invite-token'] })).toEqual({})
  })
})
