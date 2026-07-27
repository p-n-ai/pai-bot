import { createFileRoute } from '@tanstack/react-router'
import { useCallback, useEffect, useRef, useState } from 'react'

import type { ClassesState } from '@/components/classes/classes-management'
import { useAuth } from '@/auth-provider'
import { AdminPageSection } from '@/components/shared/admin-page-section'
import { ClassesContent } from '@/components/classes/classes-management'
import { listGroups } from '@/lib/admin-api'

export const Route = createFileRoute('/_authenticated/dashboard/classes')({
  component: ClassesRoute,
})

function ClassesRoute() {
  const { auth } = useAuth()
  const [state, setState] = useState<ClassesState>({
    status: 'loading',
    groups: [],
    error: null,
  })
  const [selectedClassID, setSelectedClassID] = useState('')
  const requestVersion = useRef(0)

  const reloadClasses = useCallback(() => {
    const currentRequest = requestVersion.current + 1
    requestVersion.current = currentRequest
    setState((current) => ({
      status: 'loading',
      groups: current.groups,
      error: null,
    }))

    listGroups()
      .then((groups) => {
        if (requestVersion.current !== currentRequest) {
          return
        }

        setState({ status: 'ready', groups, error: null })
        setSelectedClassID((current) =>
          groups.some((group) => group.id === current)
            ? current
            : (groups[0]?.id ?? ''),
        )
      })
      .catch((caught: unknown) => {
        if (requestVersion.current !== currentRequest) {
          return
        }

        setState((current) => ({
          status: 'error',
          groups: current.groups,
          error: caught instanceof Error ? caught.message : 'Classes failed',
        }))
      })
  }, [])

  useEffect(() => {
    reloadClasses()

    return () => {
      requestVersion.current += 1
    }
  }, [reloadClasses])

  return (
    <AdminPageSection
      description='Create classes, share student join codes, and manage each roster.'
      eyebrow='Teaching'
      title='Classes'
    >
      <ClassesContent
        canInviteAdults={
          auth.session?.user.role === 'admin' ||
          auth.session?.user.role === 'platform_admin'
        }
        onCreated={reloadClasses}
        onRetry={reloadClasses}
        onSelectClass={setSelectedClassID}
        selectedClassID={selectedClassID}
        state={state}
      />
    </AdminPageSection>
  )
}
