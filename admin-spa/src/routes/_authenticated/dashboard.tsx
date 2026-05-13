import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useCallback, useEffect, useState } from 'react'

import type { DashboardState } from '@/components/dashboard/dashboard-page-view'
import { DashboardPageView } from '@/components/dashboard/dashboard-page-view'
import { runWhenActive } from '@/lib/active-guard'
import { sendStudentNudge } from '@/lib/admin-api'
import { getNudgeSuccessMessage } from '@/lib/dashboard-nudge-copy'
import { fetchDashboardProgress } from '@/lib/dashboard-progress'

export const Route = createFileRoute('/_authenticated/dashboard')({
  validateSearch: parseDashboardSearch,
  component: DashboardRoute,
})

function DashboardRoute() {
  const navigate = useNavigate()
  const search = Route.useSearch()
  const [state, setState] = useState<DashboardState>({
    status: 'loading',
    progress: null,
    error: null,
  })
  const [nudgeMessage, setNudgeMessage] = useState('')
  const [sendingStudentID, setSendingStudentID] = useState('')

  useEffect(() => {
    let active = true

    fetchDashboardProgress()
      .then((result) => {
        runWhenActive(active, () => {
          setState({ status: 'ready', result, error: null })
        })
      })
      .catch((caught: unknown) => {
        runWhenActive(active, () => {
          setState({
            status: 'error',
            progress: null,
            error: getDashboardErrorMessage(caught),
          })
        })
      })

    return () => {
      active = false
    }
  }, [])

  const handleNudge = useCallback((studentID: string, studentName: string) => {
    setSendingStudentID(studentID)
    setNudgeMessage('')

    sendStudentNudge(studentID)
      .then(() => {
        setNudgeMessage(getNudgeSuccessMessage(studentName))
      })
      .catch((caught: unknown) => {
        setNudgeMessage(
          caught instanceof Error ? caught.message : 'Nudge failed',
        )
      })
      .finally(() => {
        setSendingStudentID('')
      })
  }, [])

  const handleSelectStudent = useCallback(
    (studentID: string) => {
      navigate({
        search: (previous) => ({ ...previous, student: studentID }),
        to: '/dashboard',
      }).catch(() => {})
    },
    [navigate],
  )

  const handleCloseStudent = useCallback(() => {
    navigate({
      search: (previous) => ({ ...previous, student: undefined }),
      to: '/dashboard',
    }).catch(() => {})
  }, [navigate])

  return (
    <DashboardPageView
      nudgeMessage={nudgeMessage}
      onCloseStudent={handleCloseStudent}
      onNudge={handleNudge}
      onSelectStudent={handleSelectStudent}
      selectedStudentID={search.student}
      sendingStudentID={sendingStudentID}
      state={state}
    />
  )
}

function parseDashboardSearch(search: Record<string, unknown>) {
  return {
    student: typeof search.student === 'string' ? search.student : undefined,
  }
}

function getDashboardErrorMessage(caught: unknown): string {
  return caught instanceof Error ? caught.message : 'Class data failed'
}
