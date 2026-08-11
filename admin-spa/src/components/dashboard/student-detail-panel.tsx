/* oxlint-disable jsx-a11y/no-noninteractive-tabindex -- Scrollable regions must be keyboard-focusable. */
import { useCallback, useEffect, useState } from 'react'
import {
  PolarAngleAxis,
  PolarGrid,
  Radar,
  RadarChart,
  ResponsiveContainer,
  Tooltip,
} from 'recharts'

import type {
  StudentConversation,
  StudentDetail,
} from '@/lib/student-detail-types'
import { AdminInsetPanel } from '@/components/shared/admin-inset-panel'
import {
  AdminSurface,
  AdminSurfaceHeader,
} from '@/components/shared/admin-surface'
import { LoadState } from '@/components/shared/load-state'
import { PageHero } from '@/components/shared/page-hero'
import { StatePanel } from '@/components/shared/state-panel'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { getStudentConversations, getStudentDetail } from '@/lib/admin-api'
import { formatAdminDateTime } from '@/lib/date-format'
import {
  buildStudentViewModel,
  formatTopicLabel,
  getActivityTone,
} from '@/lib/student-detail-view'

type StudentLoadState = 'loading' | 'ready' | 'error'

export function StudentDetailPanel({
  studentID,
  variant = 'page',
}: {
  studentID: string
  variant?: 'page' | 'sheet'
}) {
  const [detail, setDetail] = useState<StudentDetail | null>(null)
  const [conversations, setConversations] = useState<
    Array<StudentConversation>
  >([])
  const [loadState, setLoadState] = useState<StudentLoadState>('loading')
  const [requestVersion, setRequestVersion] = useState(0)
  const retryLoad = useCallback(() => {
    setRequestVersion((version) => version + 1)
  }, [])

  useEffect(() => {
    let mounted = true
    setLoadState('loading')

    Promise.all([
      getStudentDetail(studentID),
      getStudentConversations(studentID),
    ])
      .then(([nextDetail, nextConversations]) => {
        if (mounted) {
          markStudentReady(
            nextDetail,
            nextConversations,
            setDetail,
            setConversations,
            setLoadState,
          )
        }
      })
      .catch(() => {
        if (mounted) {
          markStudentError(setLoadState)
        }
      })

    return () => {
      mounted = false
    }
  }, [requestVersion, studentID])

  if (loadState === 'loading') {
    return (
      <LoadState
        error={null}
        errorTitle='Learner profile unavailable'
        loadingTitle='Loading learner profile…'
        status={loadState}
      />
    )
  }

  if (loadState === 'error') {
    return (
      <StatePanel role='alert' title='Learner profile unavailable'>
        <p>
          Unable to load the latest learner record. Check your connection and
          try again.
        </p>
        <Button
          className='mt-4 transition-transform duration-150 active:scale-[0.96]'
          onClick={retryLoad}
          type='button'
        >
          Try again
        </Button>
      </StatePanel>
    )
  }

  if (!detail) {
    return (
      <StatePanel title='Learner profile unavailable'>
        Student details will appear once the latest record is ready.
      </StatePanel>
    )
  }

  return (
    <StudentDetailReady
      conversations={conversations}
      detail={detail}
      variant={variant}
    />
  )
}

function markStudentReady(
  nextDetail: StudentDetail,
  nextConversations: Array<StudentConversation>,
  setDetail: (detail: StudentDetail) => void,
  setConversations: (conversations: Array<StudentConversation>) => void,
  setLoadState: (state: StudentLoadState) => void,
) {
  setDetail(nextDetail)
  setConversations(nextConversations)
  setLoadState('ready')
}

function markStudentError(setLoadState: (state: StudentLoadState) => void) {
  setLoadState('error')
}

function StudentDetailReady({
  conversations,
  detail,
  variant,
}: {
  conversations: Array<StudentConversation>
  detail: StudentDetail
  variant: 'page' | 'sheet'
}) {
  const view = buildStudentViewModel(detail, conversations)

  return (
    <div className='min-w-0 space-y-6'>
      {variant === 'page' ? (
        <PageHero
          eyebrow='Student detail'
          title={detail.student.name}
          description={`${detail.student.form} | ${detail.student.channel} | ${detail.student.external_id}`}
          className='bg-[var(--admin-surface)]'
        >
          <a
            href='/dashboard'
            className='inline-flex rounded-sm text-sm font-medium text-[var(--text-tertiary-default)] transition-colors duration-150 hover:text-[var(--text-primary-default)] focus-visible:ring-2 focus-visible:ring-[var(--border-primary-focus)] focus-visible:ring-offset-2 focus-visible:outline-none'
          >
            Back to dashboard
          </a>
        </PageHero>
      ) : null}

      <StudentActionSummary detail={detail} view={view} />

      <section
        className={
          variant === 'sheet'
            ? 'grid min-w-0 gap-4'
            : 'grid min-w-0 gap-4 xl:grid-cols-[0.8fr_1.2fr]'
        }
      >
        <StudentMasteryRadar view={view} />
        <StudentTopicProgress detail={detail} view={view} />
      </section>

      <StudentActivityGrid view={view} />
      <StudentConversationList view={view} />
      <StudentProfileCard detail={detail} />
    </div>
  )
}

function StudentActionSummary({
  detail,
  view,
}: {
  detail: StudentDetail
  view: ReturnType<typeof buildStudentViewModel>
}) {
  const statusClassName = {
    attention:
      'border-[var(--status-warning-border)] bg-[var(--status-warning-surface)] text-[var(--status-warning-text)]',
    neutral:
      'border-[var(--status-neutral-border)] bg-[var(--status-neutral-surface)] text-[var(--status-neutral-text)]',
    positive:
      'border-[var(--status-success-border)] bg-[var(--status-success-surface)] text-[var(--status-success-text)]',
  }[view.status.tone]

  return (
    <AdminSurface
      className='overflow-hidden border-[var(--admin-line)] bg-[var(--surface-primary-default-subtle)]'
      contentClassName='p-4 sm:p-6'
    >
      <AdminSurfaceHeader
        title='Teacher snapshot'
        description='What to know and what to do next.'
        action={renderStatusBadge(view.status.label, statusClassName)}
      />

      <div className='mt-5 grid gap-3 lg:grid-cols-2'>
        <AdminInsetPanel className='rounded-lg bg-[var(--admin-surface)] p-4 lg:col-span-2'>
          <p className='text-xs font-semibold tracking-[0.12em] text-[var(--text-tertiary-default)] uppercase'>
            Recommended next step
          </p>
          <h3 className='mt-2 text-lg font-semibold text-pretty text-[var(--admin-ink)]'>
            {view.recommendation.title}
          </h3>
          <p className='mt-1 max-w-3xl text-sm leading-6 text-[var(--admin-ink-soft)]'>
            {view.recommendation.description}
          </p>
        </AdminInsetPanel>

        <StudentTopicSummary view={view} />
        <StudentRecentActivity view={view} />
      </div>

      <dl className='mt-3 grid grid-cols-2 gap-px overflow-hidden rounded-lg bg-[var(--border-general-default)] sm:grid-cols-4'>
        <StudentSummaryMetric
          label='Current streak'
          value={`${detail.streak.current} ${detail.streak.current === 1 ? 'day' : 'days'}`}
        />
        <StudentSummaryMetric
          label='Longest streak'
          value={`${detail.streak.longest} ${detail.streak.longest === 1 ? 'day' : 'days'}`}
        />
        <StudentSummaryMetric
          label='Total XP'
          value={xpNumberFormatter.format(detail.streak.total_xp)}
        />
        <StudentSummaryMetric
          label='Topics tracked'
          value={String(detail.progress.length)}
        />
      </dl>
    </AdminSurface>
  )
}

function StudentTopicSummary({
  view,
}: {
  view: ReturnType<typeof buildStudentViewModel>
}) {
  return (
    <AdminInsetPanel className='rounded-lg bg-[var(--admin-surface)] p-4'>
      <h3 className='text-sm font-semibold text-[var(--admin-ink)]'>
        Strengths & struggles
      </h3>
      <p className='mt-1 text-sm leading-5 text-[var(--admin-muted)]'>
        {view.status.description}
      </p>
      <div className='mt-4 space-y-3'>
        <TopicBadgeList
          emptyLabel='No established strengths yet'
          items={view.strengthAreas}
          label='Strengths'
          tone='strength'
        />
        <TopicBadgeList
          emptyLabel='No active struggles'
          items={view.struggleAreas}
          label='Needs support'
          tone='struggle'
        />
      </div>
    </AdminInsetPanel>
  )
}

function TopicBadgeList({
  emptyLabel,
  items,
  label,
  tone,
}: {
  emptyLabel: string
  items: StudentDetail['progress']
  label: string
  tone: 'strength' | 'struggle'
}) {
  return (
    <div>
      <p className='text-xs font-medium text-[var(--admin-muted)]'>{label}</p>
      <div className='mt-1.5 flex flex-wrap gap-1.5'>
        {items.length > 0 ? (
          items.map((item) => (
            <Badge
              className={
                tone === 'strength'
                  ? 'h-auto border-[var(--status-success-border)] bg-[var(--status-success-surface)] py-1 text-[var(--status-success-text)] tabular-nums'
                  : 'h-auto border-[var(--status-warning-border)] bg-[var(--status-warning-surface)] py-1 text-[var(--status-warning-text)] tabular-nums'
              }
              key={item.topic_id}
              variant='outline'
            >
              {formatTopicLabel(item.topic_id)}{' '}
              {Math.round(item.mastery_score * 100)}%
            </Badge>
          ))
        ) : (
          <span className='text-sm text-[var(--admin-muted)]'>
            {emptyLabel}
          </span>
        )}
      </div>
    </div>
  )
}

function StudentRecentActivity({
  view,
}: {
  view: ReturnType<typeof buildStudentViewModel>
}) {
  return (
    <AdminInsetPanel className='rounded-lg bg-[var(--admin-surface)] p-4'>
      <h3 className='text-sm font-semibold text-[var(--admin-ink)]'>
        Recent activity
      </h3>
      {view.hasConversations ? (
        <dl className='mt-4 grid grid-cols-2 gap-4'>
          <div>
            <dt className='text-xs text-[var(--admin-muted)]'>Last 14 days</dt>
            <dd className='mt-1 text-2xl font-semibold tracking-tight text-[var(--admin-ink)] tabular-nums'>
              {view.recentMessageCount}
            </dd>
            <p className='text-xs text-[var(--admin-muted)]'>messages</p>
          </div>
          <div>
            <dt className='text-xs text-[var(--admin-muted)]'>Active days</dt>
            <dd className='mt-1 text-2xl font-semibold tracking-tight text-[var(--admin-ink)] tabular-nums'>
              {view.activeDays}
            </dd>
            <p className='text-xs text-[var(--admin-muted)]'>of 14 days</p>
          </div>
          <div className='col-span-2 border-t border-[var(--border-general-default)] pt-3'>
            <dt className='text-xs text-[var(--admin-muted)]'>
              Latest message
            </dt>
            <dd className='mt-1 text-sm font-medium text-[var(--admin-ink)] tabular-nums'>
              {formatAdminDateTime(view.latestActivityAt)}
            </dd>
          </div>
        </dl>
      ) : (
        <div className='mt-4 rounded-md bg-[var(--surface-secondary-default-hover)] p-3 text-sm leading-5 text-[var(--admin-ink-soft)]'>
          No tutoring activity yet. The recommended check-in will create a
          baseline.
        </div>
      )}
    </AdminInsetPanel>
  )
}

function StudentSummaryMetric({
  label,
  value,
}: {
  label: string
  value: string
}) {
  return (
    <div className='min-w-0 bg-[var(--admin-surface)] px-3 py-3'>
      <dt className='truncate text-[11px] font-medium text-[var(--admin-muted)]'>
        {label}
      </dt>
      <dd className='mt-1 truncate text-sm font-semibold text-[var(--admin-ink)] tabular-nums'>
        {value}
      </dd>
    </div>
  )
}

function StudentMasteryRadar({
  view,
}: {
  view: ReturnType<typeof buildStudentViewModel>
}) {
  return (
    <AdminSurface>
      <AdminSurfaceHeader
        title='Mastery overview'
        description='Relative mastery across tracked topics.'
      />
      <div className='mt-4 h-[280px] sm:h-[320px]'>
        {view.radarData.length > 0 ? (
          <ResponsiveContainer height='100%' width='100%'>
            <RadarChart data={view.radarData}>
              <PolarGrid />
              <PolarAngleAxis dataKey='topic' tick={radarAxisTick} />
              <Tooltip />
              <Radar
                dataKey='mastery'
                stroke='var(--icon-tertiary-default)'
                fill='var(--surface-primary-default)'
                fillOpacity={0.35}
              />
            </RadarChart>
          </ResponsiveContainer>
        ) : (
          <div className='flex h-full items-center justify-center'>
            <EmptyDetail
              title='No mastery overview yet'
              description='Complete a guided check-in to start tracking topic mastery.'
            />
          </div>
        )}
      </div>
    </AdminSurface>
  )
}

function StudentTopicProgress({
  detail,
  view,
}: {
  detail: StudentDetail
  view: ReturnType<typeof buildStudentViewModel>
}) {
  return (
    <AdminSurface>
      <AdminSurfaceHeader
        title='Topic progress'
        description='Mastery and review timing for each tracked topic.'
      />
      <div className='mt-4 space-y-2'>
        {view.hasProgress ? (
          detail.progress.map((item) => {
            const mastery = Math.round(item.mastery_score * 100)

            return (
              <AdminInsetPanel
                className='rounded-lg p-3 sm:p-4'
                key={item.topic_id}
              >
                <div className='flex min-w-0 items-start justify-between gap-3'>
                  <p className='min-w-0 truncate text-sm font-semibold text-[var(--admin-ink)]'>
                    {formatTopicLabel(item.topic_id)}
                  </p>
                  <span className='shrink-0 text-sm font-semibold text-[var(--admin-ink-soft)] tabular-nums'>
                    {mastery}%
                  </span>
                </div>
                <Progress
                  aria-label={`${formatTopicLabel(item.topic_id)} mastery`}
                  className='mt-2 h-1.5'
                  value={mastery}
                />
                <dl className='mt-3 grid gap-2 text-xs sm:grid-cols-2'>
                  <div>
                    <dt className='text-[var(--admin-muted)]'>Last studied</dt>
                    <dd className='mt-0.5 text-[var(--admin-ink-soft)] tabular-nums'>
                      {item.last_studied_at
                        ? formatAdminDateTime(item.last_studied_at)
                        : 'Not recorded yet'}
                    </dd>
                  </div>
                  <div>
                    <dt className='text-[var(--admin-muted)]'>Next review</dt>
                    <dd className='mt-0.5 text-[var(--admin-ink-soft)] tabular-nums'>
                      {item.next_review_at
                        ? formatAdminDateTime(item.next_review_at)
                        : 'To be scheduled'}
                    </dd>
                  </div>
                </dl>
              </AdminInsetPanel>
            )
          })
        ) : (
          <EmptyDetail
            title='No topic progress yet'
            description='Topic progress will appear after the learner completes a guided check-in.'
          />
        )}
      </div>
    </AdminSurface>
  )
}

function StudentActivityGrid({
  view,
}: {
  view: ReturnType<typeof buildStudentViewModel>
}) {
  return (
    <AdminSurface>
      <AdminSurfaceHeader
        title='14-day activity'
        description='Tutoring message activity by day.'
      />
      <div className='mt-4 space-y-3'>
        <section
          aria-label='14-day tutoring activity'
          className='overflow-x-auto rounded-sm pb-2 focus-visible:ring-2 focus-visible:ring-[var(--border-primary-focus)] focus-visible:ring-offset-2 focus-visible:outline-none'
          tabIndex={0}
        >
          <div className='grid min-w-[42rem] grid-cols-[repeat(14,minmax(0,1fr))] gap-2'>
            {view.activityGrid.map((item) => (
              <div className='space-y-1.5 text-center' key={item.date}>
                <div
                  aria-label={`${item.shortLabel}: ${item.count} messages`}
                  className={`flex h-10 items-center justify-center rounded-md border border-[var(--border-general-default)] text-xs font-semibold tabular-nums ${getActivityTone(item.level)}`}
                  title={`${item.shortLabel}: ${item.count} messages`}
                >
                  {item.count}
                </div>
                <p className='text-[10px] text-[var(--admin-muted)] tabular-nums'>
                  {item.shortLabel}
                </p>
              </div>
            ))}
          </div>
        </section>
        <div className='flex flex-wrap items-center gap-2 text-xs text-[var(--admin-muted)]'>
          <span>Less active</span>
          {[0, 1, 2, 3, 4].map((level) => (
            <span
              className={`inline-flex size-3 rounded-sm border border-[var(--border-general-default)] ${getActivityTone(level)}`}
              key={level}
            />
          ))}
          <span>More active</span>
        </div>
      </div>
    </AdminSurface>
  )
}

function StudentConversationList({
  view,
}: {
  view: ReturnType<typeof buildStudentViewModel>
}) {
  return (
    <AdminSurface>
      <AdminSurfaceHeader
        title='Recent conversations'
        description='Review recent tutoring messages for context.'
        action={renderConversationCount(view.conversations.length)}
      />
      <section
        aria-label='Conversation history'
        className='mt-4 max-h-[32rem] space-y-2 overflow-y-auto overscroll-contain rounded-sm pr-1 focus-visible:ring-2 focus-visible:ring-[var(--border-primary-focus)] focus-visible:ring-offset-2 focus-visible:outline-none'
        tabIndex={0}
      >
        {!view.hasConversations ? (
          <EmptyDetail
            title='No tutoring messages yet'
            description='Messages will appear here after the learner starts a guided check-in.'
          />
        ) : null}
        {view.conversations.map((item) => (
          <AdminInsetPanel
            className={`rounded-lg p-3 [content-visibility:auto] ${
              item.role === 'student'
                ? 'bg-[var(--status-neutral-surface)]'
                : 'border-[var(--border-primary-default)] bg-[var(--surface-primary-default-subtle)]'
            }`}
            key={item.id}
          >
            <div className='mb-2 flex flex-wrap items-center justify-between gap-1 text-xs font-medium text-[var(--admin-muted)]'>
              <span>{item.role}</span>
              <span className='tabular-nums'>
                {formatAdminDateTime(item.timestamp)}
              </span>
            </div>
            <p className='text-sm leading-6 break-words text-[var(--admin-ink-soft)]'>
              {item.text}
            </p>
          </AdminInsetPanel>
        ))}
      </section>
    </AdminSurface>
  )
}

function StudentProfileCard({ detail }: { detail: StudentDetail }) {
  return (
    <AdminSurface>
      <AdminSurfaceHeader
        title='Learner record'
        description='Profile and channel identifiers.'
      />
      <dl className='mt-4 grid gap-px overflow-hidden rounded-lg bg-[var(--border-general-default)] sm:grid-cols-2 lg:grid-cols-4'>
        <StudentProfileField label='Form' value={detail.student.form} />
        <StudentProfileField
          label='Channel'
          value={detail.student.channel}
          valueClassName='capitalize'
        />
        <StudentProfileField
          label='Channel account ID'
          value={detail.student.external_id}
          valueClassName='break-all'
        />
        <StudentProfileField
          label='Joined'
          value={formatAdminDateTime(detail.student.created_at)}
        />
      </dl>
    </AdminSurface>
  )
}

function StudentProfileField({
  label,
  value,
  valueClassName,
}: {
  label: string
  value: string
  valueClassName?: string
}) {
  return (
    <div className='min-w-0 bg-[var(--admin-surface)] p-3'>
      <dt className='text-xs font-medium text-[var(--admin-muted)]'>{label}</dt>
      <dd
        className={`mt-1 text-sm font-medium text-[var(--admin-ink)] tabular-nums ${valueClassName ?? ''}`}
      >
        {value || 'Not recorded'}
      </dd>
    </div>
  )
}

function EmptyDetail({
  description,
  title,
}: {
  description: string
  title: string
}) {
  return (
    <div className='w-full rounded-lg border border-dashed border-[var(--status-neutral-border)] bg-[var(--status-neutral-surface)] p-4'>
      <h3 className='text-sm font-semibold text-[var(--admin-ink)]'>{title}</h3>
      <p className='mt-1 text-sm leading-5 text-[var(--admin-muted)]'>
        {description}
      </p>
    </div>
  )
}

function renderStatusBadge(label: string, className: string) {
  return (
    <Badge
      className={`h-auto min-h-7 px-2.5 py-1 tabular-nums transition-none! ${className}`}
      variant='outline'
    >
      {label}
    </Badge>
  )
}

function renderConversationCount(count: number) {
  if (count === 0) {
    return null
  }

  return (
    <Badge className='tabular-nums transition-none!' variant='outline'>
      {count} {count === 1 ? 'message' : 'messages'}
    </Badge>
  )
}

const radarAxisTick = { fontSize: 11 }
const xpNumberFormatter = new Intl.NumberFormat('en-US')
