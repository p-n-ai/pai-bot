import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

export function AdminInsetPanel({
  children,
  className,
  title,
}: {
  title?: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        'rounded-[24px] border border-[var(--admin-line)] bg-[var(--admin-surface)] p-4',
        className,
      )}
    >
      {title ? (
        <p className='text-xs font-semibold tracking-[0.2em] text-[var(--admin-muted)] uppercase'>
          {title}
        </p>
      ) : null}
      <div className={title ? 'mt-3' : undefined}>{children}</div>
    </div>
  )
}
