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
      className='mt-6 rounded-lg border border-border bg-card p-4'
      role={role}
    >
      <h2 className='text-xl leading-tight font-semibold'>{title}</h2>
      <div className='mt-3 text-muted-foreground'>{children}</div>
    </div>
  )
}

export function LoadingStatus({ children }: { children: ReactNode }) {
  return <p role='status'>{children}</p>
}
