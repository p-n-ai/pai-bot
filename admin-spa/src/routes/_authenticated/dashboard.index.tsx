import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { Option, Schema, flow } from 'effect'
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

const DashboardSearchSchema = Schema.Struct({
  class: Schema.optionalKey(Schema.String),
  student: Schema.optionalKey(Schema.String),
})

type DashboardSearch = typeof DashboardSearchSchema.Type

function normalizeDashboardSearch(search: DashboardSearch): DashboardSearch {
  const classID = search.class?.trim()
  const studentID = search.student?.trim()
  return {
    class: classID || undefined,
    student: studentID || undefined,
  }
}

const parseDashboardSearch = flow(
  Schema.decodeUnknownOption(DashboardSearchSchema),
  Option.map(normalizeDashboardSearch),
  Option.getOrElse((): DashboardSearch => ({})),
)

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
      .catch((cause: unknown) =>
        runWhenActive(active, () =>
          setClassesError(
            cause instanceof Error
              ? cause.message
              : 'Unable to load classes. Check your connection and try again.',
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
      .catch((cause: unknown) => {
        runWhenActive(isCurrent(), () => {
          setState({
            status: 'error',
            progress: null,
            error: getDashboardErrorMessage(cause),
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
      .catch((cause: unknown) =>
        runWhenActive(isCurrent(), () =>
          setLeaderboardState({
            status: 'error',
            message:
              cause instanceof Error
                ? cause.message
                : 'Unable to load weekly progress. Try again.',
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
      .catch((cause: unknown) => {
        setNudgeMessage(
          cause instanceof Error
            ? cause.message
            : 'Unable to send the nudge. Check your connection and try again.',
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

function getDashboardErrorMessage(cause: unknown): string {
  return cause instanceof Error
    ? cause.message
    : 'Unable to load class progress. Check your connection and try again.'
}
