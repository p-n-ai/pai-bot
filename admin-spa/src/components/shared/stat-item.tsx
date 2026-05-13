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
    <div className='flex min-h-22 flex-col justify-start gap-2.5 rounded-lg border border-border bg-card p-3.5'>
      <span className='text-muted-foreground'>{label}</span>
      <strong className='text-[1.75rem] leading-none'>{value}</strong>
      {note ? (
        <p className='m-0 text-[0.8125rem] leading-snug text-muted-foreground'>
          {note}
        </p>
      ) : null}
    </div>
  )
}
