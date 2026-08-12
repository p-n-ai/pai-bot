/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import {
  ChevronUpIcon,
  MailPlusIcon,
  TrophyIcon,
} from '@/components/ui/pandai-icons'

afterEach(cleanup)

describe('Pandai icon mappings', () => {
  it('renders application semantics from the Pandai sprite', () => {
    render(
      <>
        <MailPlusIcon data-testid='mail' />
        <TrophyIcon data-testid='trophy' />
      </>,
    )

    expect(screen.getByTestId('mail').querySelector('use')).toHaveAttribute(
      'href',
      '/icons/pandai.svg#ic-mail',
    )
    expect(screen.getByTestId('trophy').querySelector('use')).toHaveAttribute(
      'href',
      '/icons/pandai.svg#ic-award',
    )
  })

  it('derives the upward chevron from the Pandai downward chevron', () => {
    render(<ChevronUpIcon className='size-4' data-testid='chevron-up' />)

    expect(screen.getByTestId('chevron-up')).toHaveClass('rotate-180', 'size-4')
    expect(
      screen.getByTestId('chevron-up').querySelector('use'),
    ).toHaveAttribute('href', '/icons/pandai.svg#ic-chevron-down')
  })
})
