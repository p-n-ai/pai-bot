import type { ReactNode } from 'react'

import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

const FOUR_ITEMS = ['one', 'two', 'three', 'four'] as const
const FIVE_ITEMS = [...FOUR_ITEMS, 'five'] as const

export type ContentSkeletonVariant =
  | 'analytics'
  | 'budget'
  | 'classes'
  | 'settings'
  | 'users'

/** Reserves the shape of route content while its initial data is loading. */
export function ContentSkeleton({
  label,
  variant,
}: {
  label: string
  variant: ContentSkeletonVariant
}) {
  return (
    <output aria-busy='true' className='mt-6' data-skeleton-variant={variant}>
      <span className='sr-only'>{label}</span>
      {getSkeletonContent(variant)}
    </output>
  )
}

function getSkeletonContent(variant: ContentSkeletonVariant): ReactNode {
  const views = {
    analytics: (
      <div className='grid gap-5'>
        <StatGrid count={4} />
        <Surface rows={3} />
        <Surface className='min-h-64' rows={2} />
        <Surface rows={4} />
      </div>
    ),
    budget: (
      <Surface>
        <StatGrid count={3} />
        <div className='mt-7 grid gap-4'>
          <Skeleton className='h-4 w-36' />
          <Skeleton className='h-11 w-full' />
          <Skeleton className='h-11 w-32' />
        </div>
      </Surface>
    ),
    classes: (
      <div className='grid gap-5'>
        <StatGrid count={2} />
        <div className='grid items-start gap-5 lg:grid-cols-[minmax(220px,0.34fr)_minmax(0,1fr)]'>
          <Surface rows={4} />
          <Surface rows={5} />
        </div>
      </div>
    ),
    settings: (
      <div className='grid gap-5 xl:grid-cols-[minmax(0,1fr)_24rem]'>
        <Surface rows={5} />
        <div className='grid gap-5'>
          <Surface rows={3} />
          <Surface rows={2} />
        </div>
      </div>
    ),
    users: (
      <div className='grid gap-5'>
        <StatGrid count={5} />
        <Surface>
          <div className='flex flex-col gap-3 sm:flex-row sm:justify-between'>
            <Skeleton className='h-11 w-full sm:max-w-sm' />
            <Skeleton className='h-11 w-full sm:w-36' />
          </div>
          <div className='mt-6 grid gap-2'>
            {FIVE_ITEMS.map((item) => (
              <Skeleton className='h-12 w-full' key={item} />
            ))}
          </div>
        </Surface>
      </div>
    ),
  } satisfies Record<ContentSkeletonVariant, ReactNode>

  return views[variant]
}

function StatGrid({ count }: { count: number }) {
  const items = count === 5 ? FIVE_ITEMS : FOUR_ITEMS.slice(0, count)

  return (
    <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
      {items.map((item) => (
        <div className='rounded-xl border border-border bg-card p-5' key={item}>
          <Skeleton className='h-4 w-24' />
          <Skeleton className='mt-4 h-8 w-16' />
          <Skeleton className='mt-3 h-3 w-32 max-w-full' />
        </div>
      ))}
    </div>
  )
}

function Surface({
  children,
  className,
  rows = 0,
}: {
  children?: ReactNode
  className?: string
  rows?: number
}) {
  return (
    <div
      className={cn(
        'rounded-xl border border-border bg-card p-5 sm:p-6',
        className,
      )}
    >
      <Skeleton className='h-5 w-40' />
      <Skeleton className='mt-2 h-4 w-full max-w-md' />
      {children}
      {rows > 0 ? (
        <div className='mt-6 grid gap-3'>
          {FIVE_ITEMS.slice(0, rows).map((item) => (
            <Skeleton className='h-11 w-full' key={item} />
          ))}
        </div>
      ) : null}
    </div>
  )
}
