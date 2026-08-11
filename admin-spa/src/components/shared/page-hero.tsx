import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

export function PageHero({
  aside,
  children,
  className,
  contentClassName,
  description,
  eyebrow,
  surface = 'card',
  title,
}: {
  eyebrow: string
  title: ReactNode
  description: ReactNode
  children?: ReactNode
  aside?: ReactNode
  surface?: 'card' | 'plain'
  className?: string
  contentClassName?: string
}) {
  return (
    <header
      className={cn(
        'grid gap-5 text-card-foreground',
        surface === 'card'
          ? 'rounded-[18px] bg-card p-5 ring-1 ring-[var(--admin-line,var(--border))] sm:p-6'
          : 'p-0',
        aside
          ? 'lg:grid-cols-[minmax(0,1fr)_minmax(16rem,0.7fr)] lg:items-center'
          : undefined,
        className,
      )}
    >
      <div className={cn('min-w-0 space-y-2.5', contentClassName)}>
        <p className='text-xs font-semibold tracking-[0.12em] text-muted-foreground uppercase'>
          {eyebrow}
        </p>
        <h2 className='max-w-[28ch] text-[clamp(1.5rem,2.1vw,1.875rem)] leading-[1.15] font-semibold tracking-[-0.025em] text-balance break-words text-foreground'>
          {title}
        </h2>
        <div className='max-w-[65ch] text-base leading-6 text-pretty [overflow-wrap:anywhere] text-muted-foreground'>
          {description}
        </div>
        {children}
      </div>
      {aside}
    </header>
  )
}
