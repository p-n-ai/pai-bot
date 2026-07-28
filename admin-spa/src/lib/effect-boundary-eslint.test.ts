import { Linter } from 'eslint'
import { describe, expect, it } from 'vitest'

import { effectBoundaryPlugin } from '../../eslint.config.js'

const linter = new Linter()
const config: Array<Linter.Config> = [
  {
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
    },
    plugins: {
      'effect-boundaries': effectBoundaryPlugin,
    },
    rules: {
      'effect-boundaries/require-schema-import': 'error',
    },
  },
]

function lint(code: string): Array<Linter.LintMessage> {
  return linter.verify(code, config)
}

describe('require-schema-import', () => {
  it.each([
    "import { Schema } from 'effect'",
    "import { Schema as S } from 'effect'",
    "import * as Effect from 'effect'",
    "import * as Schema from 'effect/Schema'",
  ])('accepts a valid Schema import: %s', (code) => {
    expect(lint(code)).toEqual([])
  })

  it('rejects boundary modules without access to Schema', () => {
    expect(lint("import { Option } from 'effect'")).toEqual([
      expect.objectContaining({
        messageId: 'missing',
        ruleId: 'effect-boundaries/require-schema-import',
      }),
    ])
  })
})
