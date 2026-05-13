export function Metric({
  className,
  label,
  value,
}: {
  label: string
  value: string
  className?: string
}) {
  return (
    <div className={className}>
      <p className='text-xs tracking-[0.22em] text-slate-400 uppercase'>
        {label}
      </p>
      <p className='mt-2 text-2xl font-semibold'>{value}</p>
    </div>
  )
}
