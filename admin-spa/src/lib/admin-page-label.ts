const pageLabels = [
  { prefix: '/dashboard/classes', label: 'My classes' },
  { prefix: '/dashboard/metrics', label: 'Learning progress' },
  { prefix: '/dashboard/ai-usage', label: 'AI activity' },
  { prefix: '/settings/users', label: 'Staff access' },
  { prefix: '/settings/budget', label: 'AI budget' },
  { prefix: '/settings/ai', label: 'AI settings' },
  { prefix: '/settings/embed', label: 'Website chat' },
  { prefix: '/students', label: 'Learner profile' },
  { prefix: '/parents', label: 'Family overview' },
  { prefix: '/setup', label: 'School setup' },
  { prefix: '/export', label: 'Download records' },
] as const

export function getAdminPageLabel(pathname: string): string {
  if (pathname === '/dashboard') {
    return 'Today'
  }

  return (
    pageLabels.find(
      ({ prefix }) => pathname === prefix || pathname.startsWith(`${prefix}/`),
    )?.label ?? 'School workspace'
  )
}
