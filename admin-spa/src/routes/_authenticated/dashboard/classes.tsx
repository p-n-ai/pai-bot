import { createFileRoute } from '@tanstack/react-router'
import { useCallback, useEffect, useState } from 'react'

import type { ClassesState } from '@/components/classes/classes-management'
import { AdminPageSection } from '@/components/shared/admin-page-section'
import { ClassesContent } from '@/components/classes/classes-management'
import { listGroups } from '@/lib/admin-api'

export const Route = createFileRoute('/_authenticated/dashboard/classes')({
  component: ClassesRoute,
})

function ClassesRoute() {
  const [state, setState] = useState<ClassesState>({
    status: 'loading',
    groups: [],
    error: null,
  })
  const [selectedClassID, setSelectedClassID] = useState('')

  const reloadClasses = useCallback(() => {
    setState((current) => ({
      status: 'loading',
      groups: current.groups,
      error: null,
    }))

    listGroups()
      .then((groups) => {
        setState({ status: 'ready', groups, error: null })
        // fallow-ignore-next-line complexity
        setSelectedClassID((current) => current || groups[0]?.id || '')
      })
      .catch((caught: unknown) => {
        setState({
          status: 'error',
          groups: [],
          error: caught instanceof Error ? caught.message : 'Classes failed',
        })
      })
  }, [])

  useEffect(() => {
    reloadClasses()
  }, [reloadClasses])

  return (
    <AdminPageSection
      description='Create and manage classes, join codes, member rosters, and topic assignments.'
      eyebrow='Teaching operations'
      title='Class management'
    >
      <ClassesContent
        onCreated={reloadClasses}
        onSelectClass={setSelectedClassID}
        selectedClassID={selectedClassID}
        state={state}
      />
    </AdminPageSection>
  )
}
