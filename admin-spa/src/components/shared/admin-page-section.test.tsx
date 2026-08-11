/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import { AdminPageSection } from './admin-page-section'

afterEach(() => {
  cleanup()
})

it('gives each admin page one concise identity block', () => {
  render(
    <AdminPageSection
      description='Download school records for reporting.'
      title='Download records'
    >
      <div>Page content</div>
    </AdminPageSection>,
  )

  expect(
    screen.getByRole('heading', { level: 1, name: 'Download records' }),
  ).toBeInTheDocument()
  expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1)
  expect(
    screen.getByText('Download school records for reporting.'),
  ).toBeInTheDocument()
})
