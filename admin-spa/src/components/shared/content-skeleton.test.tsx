/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { ContentSkeleton } from './content-skeleton'

afterEach(cleanup)

describe('ContentSkeleton', () => {
  it.each(['analytics', 'budget', 'classes', 'settings', 'users'] as const)(
    'announces and identifies the %s loading shape',
    (variant) => {
      const { container } = render(
        <ContentSkeleton label={`Loading ${variant}`} variant={variant} />,
      )

      expect(screen.getByRole('status')).toHaveAttribute('aria-busy', 'true')
      expect(screen.getByText(`Loading ${variant}`)).toHaveClass('sr-only')
      expect(
        container.querySelector(`[data-skeleton-variant="${variant}"]`),
      ).toBeInTheDocument()
    },
  )
})
