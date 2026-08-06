import type { ReactNode } from 'react'

import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { cn } from '@/lib/utils'

export function AdminSurface({
  children,
  className,
  contentClassName,
}: {
  children: ReactNode
  className?: string
  contentClassName?: string
}) {
  return (
    <Card
      className={cn(
        'rounded-2xl border border-[var(--admin-line)] bg-[var(--admin-surface)] shadow-[0_14px_40px_oklch(0.25_0.015_150/0.045)] ring-0',
        className,
      )}
    >
      <CardContent className={cn('p-6', contentClassName)}>
        {children}
      </CardContent>
    </Card>
  )
}

export function AdminSurfaceHeader({
  action,
  className,
  description,
  title,
}: {
  title: ReactNode
  description?: ReactNode
  action?: ReactNode
  className?: string
}) {
  return (
    <CardHeader
      className={cn(
        'flex flex-row items-center justify-between gap-3 p-0',
        className,
      )}
    >
      <div>
        <h2 className='text-lg leading-tight font-semibold tracking-[-0.02em] text-balance text-[var(--admin-ink)]'>
          {title}
        </h2>
        {description ? (
          <p className='mt-1.5 text-sm leading-5 text-pretty text-[var(--admin-muted)]'>
            {description}
          </p>
        ) : null}
      </div>
      {action}
    </CardHeader>
  )
}
