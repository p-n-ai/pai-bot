import { useEffect, useMemo, useState } from 'react'
import {
  BoltIcon,
  HeartHandshakeIcon,
  MessageCircleIcon,
  TrophyIcon,
} from 'lucide-react'
import type { CSSProperties } from 'react'

import type { ParentSummary } from '@/lib/parent-summary-types'
import { AdminHighlightPanel } from '@/components/admin-highlight-panel'
import { AdminInsetPanel } from '@/components/admin-inset-panel'
import { AdminSurface, AdminSurfaceHeader } from '@/components/admin-surface'
import { LoadState } from '@/components/load-state'
import { PageHero } from '@/components/page-hero'
import { StatePanel } from '@/components/state-panel'
import { StatCard } from '@/components/stat-card'
import { StreakMetricsPanel } from '@/components/streak-metrics-panel'
import { getParentSummary } from '@/lib/admin-api'
import { formatAdminDateTime } from '@/lib/date-format'
import {
  formatParentTopicLabel,
  getParentMasteryTone,
  getParentSummaryView,
} from '@/lib/parent-summary-view'

type ParentLoadState = 'loading' | 'ready' | 'error'

export function ParentSummaryPanel({ parentID }: { parentID: string }) {
  const [summary, setSummary] = useState<ParentSummary | null>(null)
  const [loadState, setLoadState] = useState<ParentLoadState>('loading')

  useEffect(() => {
    let mounted = true

    getParentSummary(parentID)
      .then((payload) => {
        if (mounted) {
          markParentReady(payload, setSummary, setLoadState)
        }
      })
      .catch(() => {
        if (mounted) {
          markParentError(setLoadState)
        }
      })

    return () => {
      mounted = false
    }
  }, [parentID])

  if (loadState !== 'ready') {
    return (
      <LoadState
        error='Try again later.'
        errorTitle='Parent summary unavailable'
        loadingTitle='Loading parent summary'
        status={loadState}
      />
    )
  }

  if (!summary) {
    return (
      <StatePanel title='No parent summary yet'>
        Weekly activity appears after the learner studies.
      </StatePanel>
    )
  }

  return <ParentSummaryReady summary={summary} />
}

function markParentReady(
  payload: ParentSummary,
  setSummary: (summary: ParentSummary) => void,
  setLoadState: (state: ParentLoadState) => void,
) {
  setSummary(payload)
  setLoadState('ready')
}

function markParentError(setLoadState: (state: ParentLoadState) => void) {
  setLoadState('error')
}

function ParentSummaryReady({ summary }: { summary: ParentSummary }) {
  const view = getParentSummaryView(summary)
  const heroAside = useMemo(
    () => <ParentHeroAside summary={summary} />,
    [summary],
  )

  return (
    <div className='space-y-6'>
      <PageHero
        eyebrow='Parent support summary'
        title={`${summary.child.name} this week`}
        description={view.contextLine}
        aside={heroAside}
        className='bg-white/85 dark:bg-slate-950/60'
      />

      <section
        className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'
        aria-label='Weekly summary stats'
      >
        <StatCard
          icon={BoltIcon}
          title='Active days'
          value={String(summary.weekly_stats.days_active)}
          note='Days with study activity in the last 7 days'
        />
        <StatCard
          icon={MessageCircleIcon}
          title='Messages'
          value={String(summary.weekly_stats.messages_exchanged)}
          note='Student and AI exchanges this week'
        />
        <StatCard
          icon={TrophyIcon}
          title='Quizzes'
          value={String(summary.weekly_stats.quizzes_completed)}
          note='Quiz completions logged this week'
        />
        <StatCard
          icon={HeartHandshakeIcon}
          title='Needs review'
          value={String(summary.weekly_stats.needs_review_count)}
          note='Topics asking for parent encouragement'
        />
      </section>

      <section className='grid gap-4 xl:grid-cols-[1.05fr_0.95fr]'>
        <ParentMasteryPanel summary={summary} />
        <ParentEncouragementPanel summary={summary} />
      </section>
    </div>
  )
}

function ParentHeroAside({ summary }: { summary: ParentSummary }) {
  return (
    <StreakMetricsPanel
      current={summary.streak.current}
      longest={summary.streak.longest}
      totalXP={summary.streak.total_xp}
    />
  )
}

function ParentMasteryPanel({ summary }: { summary: ParentSummary }) {
  const view = getParentSummaryView(summary)

  return (
    <AdminSurface>
      <AdminSurfaceHeader title='Mastery progress' />
      <div className='mt-6 space-y-4'>
        {view.hasMastery ? (
          view.masteryRows.map((item) => (
            <ParentMasteryRow item={item} key={item.topic_id} />
          ))
        ) : (
          <StatePanel title='No mastery data yet'>
            Mastery data appears after practice.
          </StatePanel>
        )}
      </div>
    </AdminSurface>
  )
}

function ParentMasteryRow({
  item,
}: {
  item: ParentSummary['mastery'][number]
}) {
  const score = Math.round(item.mastery_score * 100)
  const progressStyle = useMemo<CSSProperties>(
    () => ({ width: `${score}%` }),
    [score],
  )

  return (
    <div className='space-y-2'>
      <div className='flex items-center justify-between gap-3'>
        <p className='text-sm font-medium text-slate-900 dark:text-slate-100'>
          {formatParentTopicLabel(item.topic_id)}
        </p>
        <span className='text-xs tracking-[0.18em] text-slate-500 uppercase dark:text-slate-400'>
          {score}%
        </span>
      </div>
      <div className='h-3 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800'>
        <div
          className={`h-full rounded-full ${getParentMasteryTone(item.mastery_score)}`}
          style={progressStyle}
        />
      </div>
      <p className='text-xs text-slate-500 dark:text-slate-400'>
        {item.next_review_at
          ? `Next review ${formatAdminDateTime(item.next_review_at)}`
          : 'No review scheduled yet'}
      </p>
    </div>
  )
}

function ParentEncouragementPanel({ summary }: { summary: ParentSummary }) {
  const view = getParentSummaryView(summary)

  return (
    <AdminSurface>
      <AdminSurfaceHeader title='Encouragement suggestion' />
      <div className='mt-6 space-y-4'>
        <AdminHighlightPanel className='p-5'>
          <p className='text-xs font-semibold tracking-[0.22em] text-sky-300 uppercase'>
            Suggested message
          </p>
          <p className='mt-3 text-2xl font-semibold tracking-tight'>
            {view.encouragementHeadline}
          </p>
          <p className='mt-3 text-sm leading-7 text-slate-200'>
            {view.encouragementText}
          </p>
        </AdminHighlightPanel>

        <AdminInsetPanel>
          <p className='text-sm font-medium text-slate-900 dark:text-slate-100'>
            What this means for home support
          </p>
          <p className='mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300'>
            Praise one clear win. Ask for one short practice round.
          </p>
        </AdminInsetPanel>
      </div>
    </AdminSurface>
  )
}
