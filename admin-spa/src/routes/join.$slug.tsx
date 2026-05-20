import { createFileRoute } from '@tanstack/react-router'
import { useEffect, useState } from 'react'

import type { JoinClassView } from '@/lib/join-types'
import { JoinClassCard } from '@/components/auth/join-class-card'
import { LoadState } from '@/components/shared/load-state'
import { getJoinClass } from '@/lib/admin-api'

export const Route = createFileRoute('/join/$slug')({
  component: JoinRoute,
})

type JoinState =
  | { status: 'loading'; joinClass: null; error: null }
  | { status: 'ready'; joinClass: JoinClassView; error: null }
  | { status: 'error'; joinClass: null; error: string }

function JoinRoute() {
  const { slug } = Route.useParams()
  const [state, setState] = useState<JoinState>({
    status: 'loading',
    joinClass: null,
    error: null,
  })

  useEffect(() => {
    let active = true

    getJoinClass(slug)
      .then((joinClass) => {
        if (active) {
          setState({ status: 'ready', joinClass, error: null })
        }
      })
      .catch((caught: unknown) => {
        if (active) {
          setState({
            status: 'error',
            joinClass: null,
            error:
              caught instanceof Error
                ? caught.message
                : 'Class join link unavailable',
          })
        }
      })

    return () => {
      active = false
    }
  }, [slug])

  return (
    <main className='join-page'>
      {state.status === 'ready' ? (
        <JoinClassCard joinClass={state.joinClass} />
      ) : (
        <LoadState
          error={state.error}
          errorTitle='Class join link unavailable'
          loadingTitle='Loading class join link...'
          status={state.status}
        />
      )}
    </main>
  )
}
