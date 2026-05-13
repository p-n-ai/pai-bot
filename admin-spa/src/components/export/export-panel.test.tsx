/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { ExportPanel } from './export-panel'

describe('ExportPanel', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders the tenant-scoped export downloads from the current admin app', () => {
    render(<ExportPanel />)

    expect(screen.getByLabelText('Tenant export downloads')).toBeInTheDocument()
    expectDownloadLink('Students CSV', '/api/admin/export/students')
    expectDownloadLink('Conversations JSON', '/api/admin/export/conversations')
    expectDownloadLink('Progress CSV', '/api/admin/export/progress')
  })

  it('renders the source-admin descriptions for each export download', () => {
    render(<ExportPanel />)

    expect(
      screen.getByRole('heading', { name: 'Students CSV' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Conversations JSON' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Progress CSV' }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Download student roster details with form, channel, tracked topics, and average mastery.',
      ),
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Download structured conversation transcripts with message metadata for audit and review workflows.',
      ),
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Download per-topic mastery records with scheduling fields for spreadsheet or warehouse import.',
      ),
    ).toBeInTheDocument()
  })
})

function expectDownloadLink(name: string, href: string) {
  const link = screen.getByRole('link', { name: new RegExp(name, 'u') })

  expect(link).toHaveAttribute('download')
  expect(link).toHaveAttribute('href', href)
}
