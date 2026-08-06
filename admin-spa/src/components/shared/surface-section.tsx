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
      className='grid gap-6 rounded-2xl border border-[var(--admin-line)] bg-[var(--admin-surface)] p-5 shadow-[0_14px_40px_oklch(0.25_0.015_150/0.045)] sm:p-7'
      aria-label={title}
    >
      <div className='flex items-start justify-between gap-3'>
        <div>
          <h2 className='text-xl leading-tight font-semibold tracking-[-0.025em] text-balance text-[var(--admin-ink)]'>
            {title}
          </h2>
          <p className='mt-1.5 max-w-2xl text-sm leading-6 text-pretty text-[var(--admin-muted)]'>
            {description}
          </p>
        </div>
      </div>
      <div className='min-w-0'>{children}</div>
    </section>
  )
}
