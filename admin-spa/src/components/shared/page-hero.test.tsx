/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import { PageHero } from './page-hero'

afterEach(() => {
  cleanup()
})

it('keeps the record title subordinate to the page title', () => {
  render(
    <PageHero
      description='Class 4A · Telegram'
      eyebrow='Learner record'
      title='Alya Sofea'
    />,
  )

  expect(
    screen.getByRole('heading', { level: 2, name: 'Alya Sofea' }),
  ).toBeInTheDocument()
  expect(screen.queryByRole('heading', { level: 1 })).not.toBeInTheDocument()
})
