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
    <section className='mx-auto w-full max-w-[1240px] px-4 py-7 sm:px-6 sm:py-9 lg:px-10 lg:py-11'>
      {hideEyebrow ? null : (
        <p className='mb-2 text-[0.6875rem] font-semibold tracking-[0.16em] text-muted-foreground uppercase'>
          {eyebrow}
        </p>
      )}
      <h1 className='max-w-3xl text-[clamp(2rem,3vw,2.75rem)] leading-[1.06] font-semibold text-foreground'>
        {title}
      </h1>
      <p className='mt-2.5 max-w-2xl text-[0.9375rem] leading-6 text-muted-foreground'>
        {description}
      </p>
      {children}
    </section>
  )
}
