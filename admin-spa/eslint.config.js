// @ts-check

import { tanstackConfig } from '@tanstack/eslint-config'
import neostandard, { resolveIgnoresFromGitignore } from 'neostandard'
import { fileURLToPath } from 'node:url'
import tailwindcss from 'eslint-plugin-tailwindcss'
import unusedImports from 'eslint-plugin-unused-imports'
import tseslint from 'typescript-eslint'

const tsconfigRootDir = fileURLToPath(new URL('.', import.meta.url))
const tailwindCSSPath = fileURLToPath(
  new URL('./src/styles.css', import.meta.url),
)

const genericGuardRestrictions = [
  'error',
  {
    selector:
      'Identifier[name=/^(isRecord|hasObjectProperties|hasStringProps|hasNumberProps|optionalStringOrNull)$/]',
    message:
      'Define an Effect Schema and decode unknown boundary input instead of using a generic property guard.',
  },
]

export const effectBoundaryPlugin = {
  rules: {
    'require-schema-import': {
      meta: {
        type: 'problem',
        schema: [],
        messages: {
          missing:
            'Boundary modules must import Schema from Effect and model unknown payloads with explicit schemas.',
        },
      },
      create(context) {
        let importsEffectSchema = false

        return {
          ImportDeclaration(node) {
            if (
              (node.source.value === 'effect' &&
                node.specifiers.some(
                  (specifier) =>
                    specifier.type === 'ImportNamespaceSpecifier' ||
                    (specifier.type === 'ImportSpecifier' &&
                      specifier.imported.type === 'Identifier' &&
                      specifier.imported.name === 'Schema'),
                )) ||
              (node.source.value === 'effect/Schema' &&
                node.specifiers.some(
                  (specifier) => specifier.type === 'ImportNamespaceSpecifier',
                ))
            ) {
              importsEffectSchema = true
            }
          },
          'Program:exit'(node) {
            if (!importsEffectSchema) {
              context.report({ node, messageId: 'missing' })
            }
          },
        }
      },
    },
  },
}

export default tseslint.config(
  ...tanstackConfig,
  ...neostandard({
    ts: true,
    noStyle: true,
  }),
  ...tseslint.configs.recommended,
  ...tailwindcss.configs['flat/recommended'],
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      parserOptions: {
        projectService: true,
        tsconfigRootDir,
      },
    },
  },
  {
    ignores: [
      ...resolveIgnoresFromGitignore(),
      'node_modules/**',
      'dist/**',
      'build/**',
      '.source/**',
      '**/fixtures/**',
      'eslint.config.js',
      'src/routeTree.gen.ts',
      '**/__index__.tsx',
    ],
  },
  {
    name: 'admin-spa/rules',
    plugins: {
      'unused-imports': unusedImports,
    },
    rules: {
      '@typescript-eslint/no-unsafe-function-type': 'off',
      '@typescript-eslint/consistent-type-imports': [
        'error',
        {
          fixStyle: 'inline-type-imports',
          prefer: 'type-imports',
        },
      ],
      '@typescript-eslint/no-unused-vars': 'off',
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: './type-guards',
              message:
                'Boundary contracts belong in Effect Schemas, not generic type-guard modules.',
            },
          ],
        },
      ],
      'no-shadow': 'off',
      'tailwindcss/classnames-order': 'error',
      'tailwindcss/no-custom-classname': 'off',
      'unused-imports/no-unused-imports': 'error',
      'unused-imports/no-unused-vars': [
        'warn',
        {
          args: 'after-used',
          argsIgnorePattern: '^_',
          vars: 'all',
          varsIgnorePattern: '^_',
        },
      ],
    },
    settings: {
      tailwindcss: {
        callees: ['cn', 'cva'],
        config: tailwindCSSPath,
      },
    },
  },
  {
    name: 'admin-spa/effect-boundaries',
    files: ['src/lib/*-types.ts', 'src/lib/auth-client.ts'],
    plugins: {
      'effect-boundaries': effectBoundaryPlugin,
    },
    rules: {
      'effect-boundaries/require-schema-import': 'error',
      'no-restricted-syntax': [
        ...genericGuardRestrictions,
        {
          selector:
            'BinaryExpression[operator=/^(===|!==)$/][left.type="UnaryExpression"][left.operator="typeof"][right.value="object"]',
          message:
            'Model object boundaries with Effect Schema instead of hand-written typeof object guards.',
        },
        {
          selector:
            'BinaryExpression[operator=/^(===|!==)$/][right.type="UnaryExpression"][right.operator="typeof"][left.value="object"]',
          message:
            'Model object boundaries with Effect Schema instead of hand-written typeof object guards.',
        },
      ],
    },
  },
)
