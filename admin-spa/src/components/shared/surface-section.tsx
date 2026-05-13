import type { ReactNode } from 'react'

export function SurfaceSection({
  children,
  description,
  title,
}: {
  children: ReactNode
  description: string
  title: string
}) {
  return (
    <section
      className='grid gap-6 rounded-lg border border-border bg-card p-6'
      aria-label={title}
    >
      <div className='flex items-start justify-between gap-3'>
        <div>
          <h2 className='text-xl leading-tight font-semibold'>{title}</h2>
          <p className='mt-1 text-sm leading-6 text-muted-foreground'>
            {description}
          </p>
        </div>
      </div>
      <div className='min-w-0'>{children}</div>
    </section>
  )
}
