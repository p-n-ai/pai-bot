/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { JoinClassCard } from './join-class-card'

describe('JoinClassCard', () => {
  it('renders the source admin public join summary', () => {
    render(<JoinClassCard joinClass={joinClass} />)

    expect(screen.getByLabelText('Class join link')).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Form 1 Algebra' }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/Student enrollment and invite completion/u),
    ).toBeInTheDocument()
    expect(screen.getByText('Sekolah Harapan')).toBeInTheDocument()
    expect(screen.getByText('KSSM Form 1')).toBeInTheDocument()
  })
})

const joinClass = {
  class_id: 'class-1',
  class_name: 'Form 1 Algebra',
  class_slug: 'form-1-algebra',
  curriculum_label: 'KSSM Form 1',
  school_name: 'Sekolah Harapan',
} as const
