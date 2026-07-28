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

const genericPropertyGuardName =
  /^(?:isRecord|hasObjectProperties|has[A-Z][A-Za-z0-9]*Props|optional[A-Z][A-Za-z0-9]*OrNull)$/
const legacyTypeGuardsModule = /(?:^|\/)type-guards(?:\.[cm]?[jt]sx?)?$/

/** ESLint rules that keep unknown admin response parsing on Effect Schema. */
export const effectBoundaryPlugin = {
  rules: {
    'no-legacy-type-guards': {
      meta: {
        type: 'problem',
        schema: [],
        messages: {
          forbidden:
            'Boundary contracts belong in Effect Schemas, not the legacy type-guards module.',
        },
      },
      create(context) {
        return {
          ImportDeclaration(node) {
            if (
              typeof node.source.value === 'string' &&
              legacyTypeGuardsModule.test(node.source.value)
            ) {
              context.report({ node, messageId: 'forbidden' })
            }
          },
        }
      },
    },
    'no-object-property-guards': {
      meta: {
        type: 'problem',
        schema: [],
        messages: {
          generic:
            'Define an Effect Schema instead of using a generic object-property guard.',
          objectTypeof:
            'Model object boundaries with Effect Schema instead of hand-written typeof object guards.',
        },
      },
      create(context) {
        return {
          BinaryExpression(node) {
            if (
              (node.operator === '===' || node.operator === '!==') &&
              ((node.left.type === 'UnaryExpression' &&
                node.left.operator === 'typeof' &&
                node.right.type === 'Literal' &&
                node.right.value === 'object') ||
                (node.right.type === 'UnaryExpression' &&
                  node.right.operator === 'typeof' &&
                  node.left.type === 'Literal' &&
                  node.left.value === 'object'))
            ) {
              context.report({ node, messageId: 'objectTypeof' })
            }
          },
          Identifier(node) {
            if (
              node.parent.type !== 'ImportSpecifier' &&
              genericPropertyGuardName.test(node.name)
            ) {
              context.report({ node, messageId: 'generic' })
            }
          },
        }
      },
    },
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
      'effect-boundaries': effectBoundaryPlugin,
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
      '@typescript-eslint/no-empty-object-type': [
        'error',
        {
          allowInterfaces: 'with-single-extends',
        },
      ],
      '@typescript-eslint/no-unused-vars': 'off',
      'effect-boundaries/no-legacy-type-guards': 'error',
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
    rules: {
      'effect-boundaries/no-object-property-guards': 'error',
      'effect-boundaries/require-schema-import': 'error',
    },
  },
)
