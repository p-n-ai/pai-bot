import { RouterProvider } from '@tanstack/react-router'
import { useMemo } from 'react'

import { useAuth } from './auth-provider'
import { Button } from './components/ui/button'
import { Skeleton } from './components/ui/skeleton'
import { router } from './router'
import { XIcon } from '@/components/ui/pandai-icons'

const SKELETON_LINE_KEYS = ['first', 'second', 'third', 'fourth']

export function AdminApp() {
  const { auth, setAnonymousSession } = useAuth()
  const routerContext = useMemo(() => ({ auth }), [auth])

  if (auth.status === 'pending') {
    return <AdminSessionSkeleton onClose={setAnonymousSession} />
  }

  if (auth.status === 'error') {
    return (
      <main
        className='grid min-h-screen place-items-center p-6'
        id='main-content'
      >
        <section className='w-full max-w-140' role='alert'>
          <h1>Unable to verify your session</h1>
          <p>Check your connection and reload the page.</p>
        </section>
      </main>
    )
  }

  return <RouterProvider context={routerContext} router={router} />
}

function AdminSessionSkeleton({ onClose }: { onClose: () => void }) {
  return (
    <main
      aria-busy='true'
      aria-label='Loading P&AI Bot'
      className='flex min-h-svh bg-[var(--surface-secondary-default-hover)] text-[var(--text-default-heading)]'
      id='main-content'
      role='status'
    >
      <Button
        aria-label='Continue to sign in'
        className='absolute top-4 right-4 z-10 size-9 rounded-full bg-[var(--surface-general-default)]'
        onClick={onClose}
        size='icon'
        type='button'
        variant='outline'
      >
        <XIcon aria-hidden='true' className='size-4' />
      </Button>
      <aside className='hidden h-svh w-[17rem] shrink-0 border-r border-[var(--border-primary-default)] bg-[var(--surface-general-default)] p-4 sm:block'>
        <div className='flex items-center gap-3 px-2 py-3'>
          <Skeleton className='size-9 rounded-lg' />
          <Skeleton className='h-4 w-24' />
        </div>
        <div className='mt-8 grid gap-6'>
          <SkeletonGroup lines={3} />
          <SkeletonGroup lines={4} />
        </div>
        <Skeleton className='mt-[45vh] h-10 rounded-full' />
      </aside>
      <section className='mx-auto w-full max-w-[1120px] px-4 py-6 sm:px-6 lg:px-8'>
        <Skeleton className='h-3 w-24' />
        <Skeleton className='mt-4 h-12 w-full max-w-md' />
        <Skeleton className='mt-4 h-5 w-full max-w-xl' />
        <div className='mt-8 grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
          <Skeleton className='h-28 rounded-lg' />
          <Skeleton className='h-28 rounded-lg' />
          <Skeleton className='h-28 rounded-lg' />
          <Skeleton className='h-28 rounded-lg' />
        </div>
        <Skeleton className='mt-5 h-56 rounded-lg' />
      </section>
      <span className='sr-only'>Loading P&amp;AI Bot</span>
    </main>
  )
}

function SkeletonGroup({ lines }: { lines: number }) {
  return (
    <div className='grid gap-3'>
      <Skeleton className='h-3 w-20' />
      {SKELETON_LINE_KEYS.slice(0, lines).map((key) => (
        <Skeleton className='h-10 rounded-lg' key={key} />
      ))}
    </div>
  )
}
