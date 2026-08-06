export function StatItem({
  label,
  note,
  value,
}: {
  label: string
  note?: string
  value: string
}) {
  return (
    <div className='flex min-h-22 flex-col justify-start gap-2.5 rounded-xl bg-[var(--admin-surface-muted)] p-4'>
      <span className='text-[var(--admin-muted)]'>{label}</span>
      <strong className='text-[1.75rem] leading-none tracking-[-0.03em] tabular-nums text-[var(--admin-ink)]'>
        {value}
      </strong>
      {note ? (
        <p className='m-0 text-[0.8125rem] leading-snug text-[var(--admin-muted)]'>
          {note}
        </p>
      ) : null}
    </div>
  )
}
