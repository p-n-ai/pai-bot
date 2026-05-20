/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { AIUsageProviderTable } from './ai-usage-provider-table'
import type { AIUsageSummary } from '@/lib/ai-usage-types'

describe('AIUsageProviderTable', () => {
  it('renders the source-admin empty provider traffic copy', () => {
    render(<AIUsageProviderTable providers={emptyProviders} />)

    expect(screen.getByText('No provider traffic recorded')).toBeInTheDocument()
    expect(
      screen.getByText(
        'Provider rows will populate after the first successful AI requests for this tenant.',
      ),
    ).toBeInTheDocument()
  })
})

const emptyProviders: AIUsageSummary['providers'] = []
