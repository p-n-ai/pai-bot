const adminDateTimeFormatter = new Intl.DateTimeFormat('en-GB', {
  day: '2-digit',
  hour: '2-digit',
  hour12: false,
  minute: '2-digit',
  month: 'short',
  timeZone: 'UTC',
  year: 'numeric',
})

export function formatAdminDateTime(value: string | null | undefined): string {
  if (!value) {
    return ''
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  return `${adminDateTimeFormatter.format(date)} UTC`
}

export function formatDayCount(days: number): string {
  return `${days} ${days === 1 ? 'day' : 'days'}`
}
