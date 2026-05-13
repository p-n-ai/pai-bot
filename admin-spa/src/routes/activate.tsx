import { createFileRoute, redirect, useNavigate } from '@tanstack/react-router'
import { useCallback } from 'react'

import type { AuthSession } from '@/lib/auth-types'
import { useAuth } from '@/auth-provider'
import { InviteActivationForm } from '@/components/invite-activation-form'
import { parseActivationSearch } from '@/lib/activation-search'
import { getDefaultRouteForUser } from '@/lib/rbac'

export const Route = createFileRoute('/activate')({
  validateSearch: parseActivationSearch,
  beforeLoad: ({ context }) => {
    if (context.auth.status !== 'authenticated') {
      return
    }

    throw redirect({
      to: getDefaultRouteForUser(context.auth.session.user),
    })
  },
  component: ActivateRoute,
})

function ActivateRoute() {
  const search = Route.useSearch()
  const navigate = useNavigate()
  const { setAuthenticatedSession } = useAuth()

  const finishActivation = useCallback(
    async (session: AuthSession) => {
      setAuthenticatedSession(session)
      await navigate({
        to: getDefaultRouteForUser(session.user),
      })
    },
    [navigate, setAuthenticatedSession],
  )

  return (
    <InviteActivationForm
      onAuthenticated={finishActivation}
      token={search.token ?? ''}
    />
  )
}
