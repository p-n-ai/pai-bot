import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

import type { AuthSession } from '@/lib/auth-types'
import { readAuthSession } from '@/lib/auth-client'

export type AuthState =
  | {
      status: 'pending'
      session: null
      error: null
    }
  | {
      status: 'authenticated'
      session: AuthSession
      error: null
    }
  | {
      status: 'anonymous'
      session: null
      error: null
    }
  | {
      status: 'error'
      session: null
      error: Error
    }

export interface AuthContextValue {
  auth: AuthState
  setAnonymousSession: () => void
  setAuthenticatedSession: (session: AuthSession) => void
}

const pendingAuth: AuthState = {
  status: 'pending',
  session: null,
  error: null,
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [auth, setAuth] = useState<AuthState>(pendingAuth)

  useEffect(() => {
    let mounted = true

    readAuthSession()
      .then((session) => {
        if (!mounted) {
          return
        }

        setAuth(
          session
            ? { status: 'authenticated', session, error: null }
            : { status: 'anonymous', session: null, error: null },
        )
      })
      .catch((error: unknown) => {
        if (!mounted) {
          return
        }

        setAuth({
          status: 'error',
          session: null,
          error: error instanceof Error ? error : new Error('Auth failed'),
        })
      })

    return () => {
      mounted = false
    }
  }, [])

  const value = useMemo(
    () => ({
      auth,
      setAnonymousSession: () =>
        setAuth({ status: 'anonymous', session: null, error: null }),
      setAuthenticatedSession: (session: AuthSession) =>
        setAuth({ status: 'authenticated', session, error: null }),
    }),
    [auth],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const value = useContext(AuthContext)

  if (!value) {
    throw new Error('useAuth must be used within AuthProvider')
  }

  return value
}

export const initialAuthContext: AuthContextValue = {
  auth: pendingAuth,
  setAnonymousSession: () => {},
  setAuthenticatedSession: () => {},
}
