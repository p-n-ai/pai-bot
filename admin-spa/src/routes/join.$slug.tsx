import { createFileRoute } from '@tanstack/react-router'
import { ArrowLeftIcon, UsersRoundIcon } from 'lucide-react'
import { useEffect, useState } from 'react'

import type { JoinClassView } from '@/lib/join-types'
import { JoinClassCard } from '@/components/auth/join-class-card'
import { LoadState } from '@/components/shared/load-state'
import { getJoinClass } from '@/lib/admin-api'

export const Route = createFileRoute('/join/$slug')({
  component: JoinRoute,
})

type JoinState =
  | { status: 'loading'; joinClass: null; error: null }
  | { status: 'ready'; joinClass: JoinClassView; error: null }
  | { status: 'error'; joinClass: null; error: string }

function JoinRoute() {
  const { slug } = Route.useParams()
  const [state, setState] = useState<JoinState>({
    status: 'loading',
    joinClass: null,
    error: null,
  })

  useEffect(() => {
    let active = true

    getJoinClass(slug)
      .then((joinClass) => {
        if (active) {
          setState({ status: 'ready', joinClass, error: null })
        }
      })
      .catch((caught: unknown) => {
        if (active) {
          setState({
            status: 'error',
            joinClass: null,
            error:
              caught instanceof Error
                ? caught.message
                : 'Class join link unavailable',
          })
        }
      })

    return () => {
      active = false
    }
  }, [slug])

  return (
    <main className='min-h-svh bg-[var(--surface-secondary-default-hover)] px-4 text-[var(--text-default-heading)] sm:px-6 lg:px-8'>
      <div className='mx-auto grid min-h-svh max-w-[1180px] grid-rows-[64px_minmax(0,1fr)]'>
        <header className='flex items-center border-b border-[var(--border-default)]'>
          <a
            className='inline-flex min-h-11 items-center gap-2.5 text-sm font-semibold tracking-tight text-[var(--text-default-heading)] no-underline outline-none focus-visible:rounded-lg focus-visible:ring-2 focus-visible:ring-[var(--border-primary-focus)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-secondary-default-hover)]'
            href='/'
          >
            <span
              aria-hidden='true'
              className='grid size-8 place-items-center rounded-lg border border-[var(--border-primary-default)] bg-[var(--surface-tertiary-default)] text-xs font-bold text-[var(--text-primary-on-color)]'
            >
              P
            </span>
            <span>P&amp;AI Bot</span>
          </a>
        </header>

        <section
          aria-labelledby='join-page-title'
          className='grid items-start gap-8 py-10 md:grid-cols-[minmax(0,0.8fr)_minmax(360px,1fr)] md:items-center md:py-14 lg:gap-14'
        >
          <div className='max-w-lg'>
            <span className='grid size-11 place-items-center rounded-xl border border-[var(--border-primary-focus)] bg-[var(--surface-primary-default)] text-[var(--icon-tertiary-default)]'>
              <UsersRoundIcon aria-hidden='true' className='size-5' />
            </span>
            <p className='mt-6 text-xs font-semibold tracking-[0.16em] text-[var(--text-tertiary-default)] uppercase'>
              Student access
            </p>
            <h1
              className='mt-3 text-[42px] leading-[0.98] font-semibold tracking-[-0.025em] text-balance sm:text-[56px]'
              id='join-page-title'
            >
              Join your class.
            </h1>
            <p className='mt-5 max-w-[38ch] text-base leading-7 text-pretty text-[var(--text-default-body)]'>
              Open the class link from your school to confirm the learning space
              before you continue.
            </p>
            <a
              className='mt-7 inline-flex min-h-11 items-center gap-2 rounded-full px-1 text-sm font-semibold text-[var(--text-tertiary-default)] underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-[var(--border-primary-focus)]'
              href='/'
            >
              <ArrowLeftIcon aria-hidden='true' className='size-4' />
              Back to sign in
            </a>
          </div>

          <div className='w-full min-w-0'>
            {state.status === 'ready' ? (
              <JoinClassCard joinClass={state.joinClass} />
            ) : (
              <LoadState
                error={state.error}
                errorTitle='Class join link unavailable'
                loadingTitle='Loading class join link...'
                status={state.status}
              />
            )}
          </div>
        </section>
      </div>
    </main>
  )
}
