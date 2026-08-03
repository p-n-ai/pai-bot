import { TrendingUpIcon } from 'lucide-react'
import { useCallback } from 'react'

import type { LeaderboardEntry } from '@/lib/leaderboard-types'
import {
  AdminSurface,
  AdminSurfaceHeader,
} from '@/components/shared/admin-surface'
import { StatePanel } from '@/components/shared/state-panel'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { getLeaderboardRowView } from '@/lib/dashboard-leaderboard-view'
import { cn } from '@/lib/utils'

export type LeaderboardState =
  | { readonly status: 'unavailable' }
  | { readonly status: 'loading' }
  | {
      readonly status: 'ready'
      readonly entries: ReadonlyArray<LeaderboardEntry>
    }
  | { readonly status: 'error'; readonly message: string }

export function ClassLeaderboard({
  onRetry,
  onSelectStudent,
  state,
}: {
  onRetry: () => void
  onSelectStudent: (studentID: string) => void
  state: LeaderboardState
}) {
  return (
    <AdminSurface aria-label='Weekly progress leaders'>
      <AdminSurfaceHeader
        title='Weekly progress leaders'
        description='Learners with the largest mastery gains in this class.'
      />
      <div className='mt-5'>
        <LeaderboardContent
          onRetry={onRetry}
          onSelectStudent={onSelectStudent}
          state={state}
        />
      </div>
    </AdminSurface>
  )
}

function LeaderboardContent({
  onRetry,
  onSelectStudent,
  state,
}: {
  onRetry: () => void
  onSelectStudent: (studentID: string) => void
  state: LeaderboardState
}) {
  if (state.status === 'unavailable') {
    return (
      <StatePanel title='Select a class'>
        Choose a class to see its weekly progress leaders.
      </StatePanel>
    )
  }
  if (state.status === 'loading') {
    return <LeaderboardSkeleton />
  }
  if (state.status === 'error') {
    return (
      <StatePanel role='alert' title='Leaderboard unavailable'>
        <p>{state.message}</p>
        <Button className='mt-4' onClick={onRetry} size='sm' type='button'>
          Try again
        </Button>
      </StatePanel>
    )
  }
  if (state.entries.length === 0) {
    return (
      <StatePanel title='No weekly gains yet'>
        Rankings appear after learners build mastery in this class.
      </StatePanel>
    )
  }

  return (
    <ol className='divide-y divide-slate-200 dark:divide-white/10'>
      {state.entries.map((entry) => (
        <LeaderboardRow
          entry={entry}
          key={entry.user_id}
          onSelectStudent={onSelectStudent}
        />
      ))}
    </ol>
  )
}

function LeaderboardRow({
  entry,
  onSelectStudent,
}: {
  entry: LeaderboardEntry
  onSelectStudent: (studentID: string) => void
}) {
  const row = getLeaderboardRowView(entry)
  const handleSelectStudent = useCallback(() => {
    onSelectStudent(row.studentID)
  }, [onSelectStudent, row.studentID])

  return (
    <li className='flex items-center gap-3 py-3'>
      <span className='w-9 text-sm font-semibold tabular-nums'>
        {row.rankLabel}
      </span>
      <Button
        className='h-auto flex-1 justify-start px-0 text-left'
        onClick={handleSelectStudent}
        type='button'
        variant='link'
      >
        {row.studentName}
      </Button>
      <span
        className={cn(
          'inline-flex items-center gap-1 text-sm font-semibold tabular-nums',
          row.gainTone === 'positive' &&
            'text-emerald-700 dark:text-emerald-300',
          row.gainTone === 'negative' && 'text-rose-700 dark:text-rose-300',
          row.gainTone === 'neutral' && 'text-slate-600 dark:text-slate-300',
        )}
      >
        <TrendingUpIcon aria-hidden='true' className='size-4' />
        {row.gainLabel}
      </span>
    </li>
  )
}

function LeaderboardSkeleton() {
  return (
    <div
      aria-label='Loading weekly progress leaders'
      className='space-y-3'
      role='status'
    >
      {[0, 1, 2].map((key) => (
        <Skeleton className='h-10 w-full' key={key} />
      ))}
    </div>
  )
}
