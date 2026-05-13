import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useCallback, useEffect } from 'react'

import type { AuthSession } from '@/lib/auth-types'
import { useAuth } from '@/auth-provider'
import { AdminHomePanel } from '@/components/admin-home-panel'
import { getDefaultRouteForUser } from '@/lib/rbac'
import { parseRootSearch } from '@/lib/root-search'
import { getAuthenticatedRootRedirectTarget } from '@/lib/root-redirect-target'

export const Route = createFileRoute('/')({
  validateSearch: parseRootSearch,
  component: AdminHome,
})

function AdminHome() {
  const search = Route.useSearch()
  const navigate = useNavigate()
  const { auth, setAuthenticatedSession } = useAuth()

  useEffect(() => {
    if (auth.status !== 'authenticated') {
      return
    }

    navigate({
      replace: true,
      to: getAuthenticatedRootRedirectTarget(auth.session.user, search.next),
    }).catch(() => {})
  }, [auth, navigate, search.next])

  const finishLogin = useCallback(
    async (session: AuthSession) => {
      setAuthenticatedSession(session)
      await navigate({
        to: search.next ?? getDefaultRouteForUser(session.user),
      })
    },
    [navigate, search.next, setAuthenticatedSession],
  )

  if (auth.status === 'authenticated') {
    return null
  }

  return (
    <AdminHomePanel
      auth={auth}
      nextPath={search.next}
      onAuthenticated={finishLogin}
    />
  )
}
