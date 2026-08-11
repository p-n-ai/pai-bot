import { useEffect, useState } from 'react'

import type {
  PublicStatusComponent,
  PublicStatusSnapshot,
} from '@/lib/public-status-types'
import { readPublicStatus } from '@/lib/public-status-client'

type StatusViewState =
  | { readonly kind: 'loading' }
  | { readonly kind: 'ready'; readonly snapshot: PublicStatusSnapshot }
  | { readonly kind: 'error' }

interface StatusRowModel {
  readonly id: PublicStatusComponent['id']
  readonly status: PublicStatusComponent['status'] | 'checking' | 'unknown'
}

const componentCopy = {
  application: {
    name: 'P&AI application',
    description: 'Admin and chat services',
  },
  ai_provider: {
    name: 'AI provider',
    description: 'AI response availability',
  },
} as const

export function PublicStatusPage() {
  const [viewState, setViewState] = useState<StatusViewState>({
    kind: 'loading',
  })

  useEffect(() => {
    const controller = new AbortController()

    readPublicStatus((input, init) =>
      fetch(input, { ...init, signal: controller.signal }),
    )
      .then((snapshot) => {
        setViewState({ kind: 'ready', snapshot })
      })
      .catch((error: unknown) => {
        if (error instanceof DOMException && error.name === 'AbortError') {
          return
        }
        setViewState({ kind: 'error' })
      })

    return () => {
      controller.abort()
    }
  }, [])

  const presentation = statusPresentation(viewState)

  return (
    <div className='min-h-svh bg-[var(--surface-secondary-default-hover)] text-[var(--text-default-heading)]'>
      <div className='mx-auto w-[min(calc(100%-2rem),42rem)] px-[max(0rem,env(safe-area-inset-left))] py-8 sm:py-10'>
        <header className='mb-10 sm:mb-12'>
          <a
            className='inline-flex min-h-11 items-center gap-3 rounded-lg text-sm font-semibold tracking-[-0.01em] no-underline outline-offset-4 focus-visible:outline-2'
            href='/health'
          >
            <span
              aria-hidden='true'
              className='grid size-8 place-items-center rounded-lg border border-[var(--border-primary-default)] bg-[var(--surface-tertiary-default)] text-[11px] font-bold tracking-[-0.04em] text-[var(--text-primary-on-color)]'
            >
              P&amp;
            </span>
            <span>P&amp;AI Status</span>
          </a>
        </header>

        <div aria-live='polite' className='grid gap-10 sm:gap-12'>
          <section aria-labelledby='status-heading' className='max-w-2xl'>
            <p className='mb-4 inline-flex items-center gap-2 text-sm font-medium text-[var(--text-default-body)]'>
              <span
                aria-hidden='true'
                className={
                  presentation.tone === 'operational'
                    ? 'size-2.5 rounded-full bg-[var(--surface-primary-default)]'
                    : presentation.tone === 'degraded'
                      ? 'size-2.5 rounded-full bg-[var(--status-warning-border)]'
                      : 'size-2.5 rounded-full bg-[var(--text-disabled-default)]'
                }
              />
              {presentation.liveLabel}
            </p>
            <h1
              className='m-0 max-w-[18ch] text-[clamp(2rem,5vw,2.5rem)] leading-[1.1] font-semibold tracking-[-0.03em] text-balance'
              id='status-heading'
            >
              {presentation.headline}
            </h1>
            <p className='mt-4 max-w-[60ch] text-base leading-7 text-pretty text-[var(--text-default-body)]'>
              {presentation.summary}
            </p>
          </section>

          <section
            aria-busy={viewState.kind === 'loading'}
            aria-labelledby='systems-heading'
          >
            <h2
              className='mb-6 text-sm leading-tight font-semibold'
              id='systems-heading'
            >
              System status
            </h2>
            <ul className='m-0 grid list-none gap-6 p-0'>
              {presentation.components.map((component) => (
                <StatusRow component={component} key={component.id} />
              ))}
            </ul>
          </section>
        </div>

        <footer className='mt-10 text-[13px] leading-5 text-[var(--text-default-body)]'>
          <a
            className='inline-flex min-h-11 items-center rounded-sm underline decoration-from-font underline-offset-4 outline-offset-4 focus-visible:outline-2'
            href='/health/status'
          >
            View JSON status
          </a>
        </footer>
      </div>
    </div>
  )
}

function StatusRow({ component }: { component: StatusRowModel }) {
  const copy = componentCopy[component.id]
  const operational = component.status === 'operational'
  const unavailable = component.status === 'unavailable'
  const label = operational
    ? 'Operational'
    : unavailable
      ? 'Unavailable'
      : component.status === 'checking'
        ? 'Checking'
        : 'Unknown'

  return (
    <li className='grid items-start gap-3 min-[32rem]:grid-cols-[minmax(0,1fr)_auto]'>
      <div className='min-w-0'>
        <h3 className='m-0 text-base leading-[1.3] font-semibold'>
          {copy.name}
        </h3>
        <p className='mt-1.5 text-sm leading-6 text-pretty text-[var(--text-default-body)]'>
          {copy.description}
        </p>
      </div>
      <span
        className={
          operational
            ? 'inline-flex w-fit items-center gap-2 text-[13px] font-semibold whitespace-nowrap text-[var(--text-tertiary-default)]'
            : unavailable
              ? 'inline-flex w-fit items-center gap-2 text-[13px] font-semibold whitespace-nowrap text-[var(--status-warning-text)]'
              : 'inline-flex w-fit items-center gap-2 text-[13px] font-semibold whitespace-nowrap text-[var(--text-default-body)]'
        }
      >
        <span aria-hidden='true' className='size-2 rounded-full bg-current' />
        {label}
      </span>
    </li>
  )
}

function statusPresentation(viewState: StatusViewState): {
  readonly headline: string
  readonly summary: string
  readonly components: ReadonlyArray<StatusRowModel>
  readonly liveLabel: string
  readonly tone: 'operational' | 'degraded' | 'neutral'
} {
  if (viewState.kind === 'loading') {
    return {
      headline: 'Checking system status',
      summary: 'Reading the latest availability from P&AI services.',
      components: [
        { id: 'application', status: 'checking' },
        { id: 'ai_provider', status: 'checking' },
      ],
      liveLabel: 'Checking status',
      tone: 'neutral',
    }
  }

  if (viewState.kind === 'error') {
    return {
      headline: 'Status currently unavailable',
      summary:
        'Unable to check the latest status. P&AI may still be available. Try again shortly.',
      components: [
        { id: 'application', status: 'unknown' },
        { id: 'ai_provider', status: 'unknown' },
      ],
      liveLabel: 'Status unavailable',
      tone: 'neutral',
    }
  }

  return viewState.snapshot.status === 'ok'
    ? {
        headline: 'All systems operational',
        summary: 'P&AI is online and serving requests normally.',
        components: viewState.snapshot.components,
        liveLabel: 'Live status',
        tone: 'operational',
      }
    : {
        headline: 'Some systems unavailable',
        summary:
          'The application is online, but one or more supporting services may be affected.',
        components: viewState.snapshot.components,
        liveLabel: 'Service advisory',
        tone: 'degraded',
      }
}
