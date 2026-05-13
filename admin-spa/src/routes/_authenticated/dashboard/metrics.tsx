import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/dashboard/metrics')({
  beforeLoad: () => {
    throw redirect({
      search: { student: undefined },
      to: '/dashboard/ai-usage',
    })
  },
})
