import type { JoinClassView } from '@/lib/join-types'

export function JoinClassCard({ joinClass }: { joinClass: JoinClassView }) {
  return (
    <section
      className='rounded-[20px] border border-[var(--border-primary-default)] bg-[var(--surface-general-default)] p-6 sm:p-8'
      aria-label='Class join link'
    >
      <p className='text-xs font-semibold tracking-[0.16em] text-[var(--text-tertiary-default)] uppercase'>
        Class join link
      </p>
      <h2 className='mt-3 text-3xl leading-tight font-semibold tracking-[-0.02em] text-[var(--text-default-heading)]'>
        {joinClass.class_name}
      </h2>
      <p className='mt-3 max-w-[52ch] text-sm leading-6 text-[var(--text-default-body)]'>
        {joinClass.school_name} is ready to use this join route for{' '}
        <strong>{joinClass.class_slug}</strong>. Student enrollment and invite
        completion still land in the next slice.
      </p>
      <dl className='mt-6 grid gap-3'>
        <div className='flex items-center justify-between gap-4 rounded-xl bg-[var(--surface-secondary-default-hover)] px-4 py-3'>
          <dt className='text-sm text-[var(--text-default-body)]'>School</dt>
          <dd className='m-0 text-right text-sm font-semibold text-[var(--text-default-heading)]'>
            {joinClass.school_name}
          </dd>
        </div>
        <div className='flex items-center justify-between gap-4 rounded-xl bg-[var(--surface-secondary-default-hover)] px-4 py-3'>
          <dt className='text-sm text-[var(--text-default-body)]'>
            Curriculum
          </dt>
          <dd className='m-0 text-right text-sm font-semibold text-[var(--text-default-heading)]'>
            {joinClass.curriculum_label}
          </dd>
        </div>
      </dl>
    </section>
  )
}
