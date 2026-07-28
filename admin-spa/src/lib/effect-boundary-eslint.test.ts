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
      'effect-boundaries/no-legacy-type-guards': 'error',
      'effect-boundaries/no-object-property-guards': 'error',
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

describe('no-legacy-type-guards', () => {
  it.each([
    "import { isRecord } from './type-guards'",
    "import { isRecord } from '@/lib/type-guards.ts'",
  ])('rejects a legacy guard import: %s', (code) => {
    expect(lint(`${code}\nimport { Schema } from 'effect'`)).toEqual([
      expect.objectContaining({
        messageId: 'forbidden',
        ruleId: 'effect-boundaries/no-legacy-type-guards',
      }),
    ])
  })

  it('allows a domain-specific schema module', () => {
    expect(
      lint(
        "import { Schema } from 'effect'\nimport { UserSchema } from './user-schema'",
      ),
    ).toEqual([])
  })
})

describe('no-object-property-guards', () => {
  it.each([
    'const isRecord = (value) => Boolean(value)',
    'const hasObjectProperties = (value) => Boolean(value)',
    'const hasStringProps = (value) => Boolean(value)',
    'const hasNumberProps = (value) => Boolean(value)',
    'const hasBooleanProps = (value) => Boolean(value)',
    'const optionalStringOrNull = (value) => Boolean(value)',
    "const check = (value) => typeof value === 'object'",
    "const check = (value) => 'object' !== typeof value",
  ])('rejects a generic property guard: %s', (code) => {
    expect(lint(`import { Schema } from 'effect'\n${code}`)).toEqual([
      expect.objectContaining({
        ruleId: 'effect-boundaries/no-object-property-guards',
      }),
    ])
  })

  it('allows domain-specific guards backed by Schema', () => {
    expect(
      lint(
        "import { Schema } from 'effect'\nconst isAuthSession = Schema.is(Schema.Struct({ id: Schema.String }))",
      ),
    ).toEqual([])
  })
})
