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
)
