import { useEffect, useMemo, useState } from 'react'
import type { CSSProperties } from 'react'

import type { ParentSummary } from '@/lib/parent-summary-types'
import {
  BoltIcon,
  HeartHandshakeIcon,
  MessageCircleIcon,
  TrophyIcon,
} from '@/components/ui/pandai-icons'
import { AdminHighlightPanel } from '@/components/shared/admin-highlight-panel'
import { AdminInsetPanel } from '@/components/shared/admin-inset-panel'
import {
  AdminSurface,
  AdminSurfaceHeader,
} from '@/components/shared/admin-surface'
import { LoadState } from '@/components/shared/load-state'
import { PageHero } from '@/components/shared/page-hero'
import { StatePanel } from '@/components/shared/state-panel'
import { StatCard } from '@/components/shared/stat-card'
import { StreakMetricsPanel } from '@/components/shared/streak-metrics-panel'
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
        error='Check your connection and try again.'
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
        eyebrow='Weekly overview'
        title={`${summary.child.name} this week`}
        description={view.contextLine}
        aside={heroAside}
        className='bg-[var(--admin-surface)]'
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
          note='Messages between the student and tutor this week'
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
        <p className='text-sm font-medium text-[var(--admin-ink)]'>
          {formatParentTopicLabel(item.topic_id)}
        </p>
        <span className='text-xs tracking-[0.18em] text-[var(--admin-muted)] uppercase'>
          {score}%
        </span>
      </div>
      <div className='h-3 overflow-hidden rounded-full bg-[var(--border-general-default)]'>
        <div
          className={`h-full rounded-full ${getParentMasteryTone(item.mastery_score)}`}
          style={progressStyle}
        />
      </div>
      <p className='text-xs text-[var(--admin-muted)]'>
        {item.next_review_at
          ? `Next review: ${formatAdminDateTime(item.next_review_at)}`
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
          <p className='text-xs font-semibold tracking-[0.22em] text-[var(--admin-nav-muted)] uppercase'>
            Suggested message
          </p>
          <p className='mt-3 text-2xl font-semibold tracking-tight'>
            {view.encouragementHeadline}
          </p>
          <p className='mt-3 text-sm leading-7 text-[var(--admin-nav-muted)]'>
            {view.encouragementText}
          </p>
        </AdminHighlightPanel>

        <AdminInsetPanel>
          <p className='text-sm font-medium text-[var(--admin-ink)]'>
            What this means for home support
          </p>
          <p className='mt-2 text-sm leading-6 text-[var(--admin-ink-soft)]'>
            Praise one clear win. Ask for one short practice round.
          </p>
        </AdminInsetPanel>
      </div>
    </AdminSurface>
  )
}
