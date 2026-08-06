import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/dashboard/retrieval-lab')(
  {
    beforeLoad: () => {
      throw redirect({
        to: '/dashboard',
        search: { class: undefined, student: undefined },
      })
    },
  },
)
