import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

export function AdminHighlightPanel({
  children,
  className,
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        'grid gap-3 rounded-[24px] bg-[var(--admin-navigation)] p-4 text-[var(--admin-nav-text)]',
        className,
      )}
    >
      {children}
    </div>
  )
}
