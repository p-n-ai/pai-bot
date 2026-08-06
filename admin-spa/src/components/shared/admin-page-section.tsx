import type { ReactNode } from 'react'

export function AdminPageSection({
  children,
  description,
  eyebrow,
  hideEyebrow = false,
  title,
}: {
  children: ReactNode
  description: string
  eyebrow: string
  hideEyebrow?: boolean
  title: string
}) {
  return (
    <section className='relative mx-auto w-full max-w-[1240px] px-5 py-8 sm:px-8 sm:py-10 lg:px-10 lg:py-12'>
      <header className='max-w-3xl'>
        {hideEyebrow ? null : (
          <div className='mb-3 flex items-center gap-2.5'>
            <span
              aria-hidden='true'
              className='size-1.5 rounded-full bg-[var(--admin-accent-muted)]'
            />
            <p className='text-[0.6875rem] font-semibold tracking-[0.16em] text-[var(--admin-muted)] uppercase'>
              {eyebrow}
            </p>
          </div>
        )}
        <h1 className='text-[clamp(1.875rem,3vw,2.5rem)] leading-[1.08] font-semibold tracking-[-0.035em] text-balance text-[var(--admin-ink)]'>
          {title}
        </h1>
        <p className='mt-2.5 max-w-2xl text-[0.9375rem] leading-6 text-pretty text-[var(--admin-muted)]'>
          {description}
        </p>
      </header>
      {children}
    </section>
  )
}
