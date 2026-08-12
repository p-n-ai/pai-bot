import type { ReactNode } from 'react'

export function StatePanel({
  children,
  role,
  title,
}: {
  children: ReactNode
  role?: 'alert'
  title: string
}) {
  return (
    <div
      className='mt-6 max-w-3xl rounded-xl border border-border bg-card p-5 sm:p-6'
      role={role}
    >
      <h2 className='text-lg leading-snug font-semibold tracking-[-0.015em]'>
        {title}
      </h2>
      <div className='mt-2 max-w-[65ch] text-sm leading-6 text-muted-foreground'>
        {children}
      </div>
    </div>
  )
}

export function LoadingStatus({ children }: { children: ReactNode }) {
  return <output>{children}</output>
}
