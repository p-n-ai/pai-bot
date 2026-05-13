/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { RetrievalLab } from './retrieval-lab'
import type * as TanStackRouter from '@tanstack/react-router'

const navigate = vi.hoisted(() => vi.fn())
const setAnonymousSession = vi.hoisted(() => vi.fn())

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()

  return {
    ...actual,
    useNavigate: () => navigate,
  }
})

vi.mock('@/auth-provider', () => ({
  useAuth: () => ({
    setAnonymousSession,
  }),
}))

describe('RetrievalLab', () => {
  beforeEach(() => {
    navigate.mockReset()
    setAnonymousSession.mockReset()
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('runs repeated retrieval searches and exposes metrics plus raw output', async () => {
    vi.mocked(fetch).mockImplementation(() =>
      Promise.resolve(
        new Response(
          JSON.stringify([
            {
              document: {
                collection_id: 'curriculum:matematik-form-1',
                id: 'doc_1',
                kind: 'topic_card',
                title: 'Linear equations',
              },
              excerpt: 'Solve linear equations one step at a time.',
              high_signal_terms: 3,
              matched_terms: 2,
              score: 12.5,
            },
          ]),
        ),
      ),
    )

    render(<RetrievalLab />)

    fireEvent.change(screen.getByLabelText('Search retrieval content'), {
      target: { value: 'first step only' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Additional settings' }))
    fireEvent.change(screen.getByLabelText('Repeat runs'), {
      target: { value: '2' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Search' }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2))
    expect(
      await screen.findByLabelText('Search run summary'),
    ).toHaveTextContent('Done. 2 run(s).')
    expect(screen.getByText(/1 hit in about \d+ ms/)).toBeInTheDocument()
    expect(screen.queryByText('Status')).not.toBeInTheDocument()
    expect(screen.getByText('Linear equations')).toBeInTheDocument()
    expect(screen.getByRole('list')).toBeInTheDocument()
    expect(screen.getByRole('listitem')).toBeInTheDocument()
    expect(screen.getByText('Terms 2/3')).toBeInTheDocument()
    expect(screen.getByText('Rank 1')).toBeInTheDocument()
    expect(screen.getByText('doc_1')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Raw output' }))
    expect(screen.getByText(/"query": "first step only"/)).toBeInTheDocument()
  })

  it('renders the source retrieval-lab title and description', () => {
    render(<RetrievalLab />)

    expect(
      screen.getByRole('heading', { name: 'PaiBot Search', level: 1 }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Search retrieval content across collections and sources.',
      ),
    ).toBeInTheDocument()
  })

  it('keeps retrieval filters collapsed behind an accessible settings trigger', () => {
    render(<RetrievalLab />)

    const settingsButton = screen.getByRole('button', {
      name: 'Additional settings',
    })

    expect(settingsButton).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByLabelText('Repeat runs')).not.toBeInTheDocument()

    fireEvent.click(settingsButton)

    expect(settingsButton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByLabelText('Repeat runs')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Load example' }),
    ).toBeInTheDocument()
  })

  it('keeps retrieval results hidden before the first run', () => {
    render(<RetrievalLab />)

    expect(
      screen.getByLabelText('Search retrieval content'),
    ).toBeInTheDocument()
    expect(screen.queryByText('Status')).not.toBeInTheDocument()
    expect(screen.queryByText('No hits yet.')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Raw output' }),
    ).not.toBeInTheDocument()
  })

  it('keeps raw retrieval output collapsed after a run until requested', async () => {
    vi.mocked(fetch).mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify([]))),
    )

    render(<RetrievalLab />)

    fireEvent.click(screen.getByRole('button', { name: 'Search' }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(3))
    expect(
      await screen.findByLabelText('Search run summary'),
    ).toHaveTextContent('Done. 3 run(s).')

    const rawOutputButton = screen.getByRole('button', { name: 'Raw output' })

    expect(rawOutputButton).toHaveAttribute('aria-expanded', 'false')
    expect(
      screen.queryByText('Run a query to inspect the raw payload and hits.'),
    ).not.toBeInTheDocument()

    fireEvent.click(rawOutputButton)

    expect(rawOutputButton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(/"query": "linear equation"/)).toBeInTheDocument()
  })

  it('renders empty successful searches as a bordered result panel', async () => {
    vi.mocked(fetch).mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify([]))),
    )

    render(<RetrievalLab />)

    fireEvent.click(screen.getByRole('button', { name: 'Search' }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(3))
    expect(
      await screen.findByLabelText('Search run summary'),
    ).toHaveTextContent('Done. 3 run(s).')
    expect(screen.getByText(/0 hits in about \d+ ms/)).toBeInTheDocument()
    expect(screen.getByText('No hits yet.')).toBeInTheDocument()
  })
})
