import type { ReactNode } from 'react'

export function AdminPageSection({
  children,
  description,
  title,
}: {
  children: ReactNode
  description: string
  title: string
}) {
  return (
    <section className='relative mx-auto w-full max-w-[1180px] px-4 py-7 sm:px-6 sm:py-8 lg:px-8 lg:py-10'>
      <header className='max-w-2xl'>
        <h1 className='text-[clamp(2rem,2.4vw,2.25rem)] leading-[1.1] font-semibold tracking-[-0.025em] text-balance text-[var(--admin-ink)]'>
          {title}
        </h1>
        <p className='mt-2 max-w-[65ch] text-base leading-6 text-pretty text-[var(--admin-muted)]'>
          {description}
        </p>
      </header>
      {children}
    </section>
  )
}
