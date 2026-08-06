import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useCallback, useEffect, useRef, useState } from 'react'

import type { DashboardState } from '@/components/dashboard/dashboard-page-view'
import type { LeaderboardState } from '@/components/dashboard/class-leaderboard'
import type { GroupRecord } from '@/lib/group-types'
import { DashboardPageView } from '@/components/dashboard/dashboard-page-view'
import { runWhenActive } from '@/lib/active-guard'
import {
  getGroupLeaderboard,
  listGroups,
  sendStudentNudge,
} from '@/lib/admin-api'
import { getNudgeSuccessMessage } from '@/lib/dashboard-nudge-copy'
import { fetchDashboardProgress } from '@/lib/dashboard-progress'

export const Route = createFileRoute('/_authenticated/dashboard/')({
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
  const [classes, setClasses] = useState<ReadonlyArray<GroupRecord>>([])
  const [classesLoading, setClassesLoading] = useState(true)
  const [classesError, setClassesError] = useState('')
  const [leaderboardState, setLeaderboardState] = useState<LeaderboardState>({
    status: search.class ? 'loading' : 'unavailable',
  })
  const [leaderboardRetryVersion, setLeaderboardRetryVersion] = useState(0)
  const progressGeneration = useRef(0)
  const leaderboardGeneration = useRef(0)

  useEffect(() => {
    let active = true
    listGroups('class')
      .then((groups) => runWhenActive(active, () => setClasses(groups)))
      .catch((caught: unknown) =>
        runWhenActive(active, () =>
          setClassesError(
            caught instanceof Error ? caught.message : 'Classes failed',
          ),
        ),
      )
      .finally(() => runWhenActive(active, () => setClassesLoading(false)))
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    const generation = ++progressGeneration.current
    const isCurrent = () => generation === progressGeneration.current
    const classID = search.class ?? 'all-students'
    setState({ status: 'loading', progress: null, error: null })

    fetchDashboardProgress(classID)
      .then((result) => {
        runWhenActive(isCurrent(), () => {
          setState({ status: 'ready', result, error: null })
        })
      })
      .catch((caught: unknown) => {
        runWhenActive(isCurrent(), () => {
          setState({
            status: 'error',
            progress: null,
            error: getDashboardErrorMessage(caught),
          })
        })
      })

    return () => {
      if (progressGeneration.current === generation) {
        progressGeneration.current += 1
      }
    }
  }, [search.class])

  useEffect(() => {
    const generation = ++leaderboardGeneration.current
    const isCurrent = () => generation === leaderboardGeneration.current
    if (!search.class) {
      setLeaderboardState({ status: 'unavailable' })
      return
    }

    setLeaderboardState({ status: 'loading' })
    getGroupLeaderboard(search.class)
      .then((entries) =>
        runWhenActive(isCurrent(), () =>
          setLeaderboardState({ status: 'ready', entries }),
        ),
      )
      .catch((caught: unknown) =>
        runWhenActive(isCurrent(), () =>
          setLeaderboardState({
            status: 'error',
            message:
              caught instanceof Error ? caught.message : 'Leaderboard failed',
          }),
        ),
      )

    return () => {
      if (leaderboardGeneration.current === generation) {
        leaderboardGeneration.current += 1
      }
    }
  }, [leaderboardRetryVersion, search.class])

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
        search: (previous) => ({
          class: previous.class,
          student: studentID,
        }),
        to: '/dashboard',
      }).catch(() => {})
    },
    [navigate],
  )

  const handleSelectClass = useCallback(
    (classID: string | undefined) => {
      navigate({
        search: (previous) => ({
          ...previous,
          class: classID,
          student: undefined,
        }),
        to: '/dashboard',
      }).catch(() => {})
    },
    [navigate],
  )

  const handleRetryLeaderboard = useCallback(() => {
    setLeaderboardRetryVersion((version) => version + 1)
  }, [])

  const handleCloseStudent = useCallback(() => {
    navigate({
      search: (previous) => ({
        class: previous.class,
        student: undefined,
      }),
      to: '/dashboard',
    }).catch(() => {})
  }, [navigate])

  return (
    <DashboardPageView
      classes={classes}
      classesError={classesError}
      classesLoading={classesLoading}
      leaderboardState={leaderboardState}
      nudgeMessage={nudgeMessage}
      onCloseStudent={handleCloseStudent}
      onNudge={handleNudge}
      onRetryLeaderboard={handleRetryLeaderboard}
      onSelectClass={handleSelectClass}
      onSelectStudent={handleSelectStudent}
      selectedStudentID={search.student}
      selectedClassID={search.class}
      sendingStudentID={sendingStudentID}
      state={state}
    />
  )
}

function parseDashboardSearch(search: Record<string, unknown>) {
  return {
    class: readNonEmptySearchValue(search.class),
    student: readNonEmptySearchValue(search.student),
  }
}

function readNonEmptySearchValue(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  return trimmed || undefined
}

function getDashboardErrorMessage(caught: unknown): string {
  return caught instanceof Error ? caught.message : 'Class data failed'
}
