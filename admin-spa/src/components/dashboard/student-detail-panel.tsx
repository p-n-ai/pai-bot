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
        errorTitle='Student detail unavailable'
        loadingTitle='Loading student detail…'
        status={loadState}
      />
    )
  }

  if (loadState === 'error') {
    return (
      <StatePanel role='alert' title='Student detail unavailable'>
        <p>
          The latest learner record could not be loaded. Check your connection
          and try again.
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
      <StatePanel title='Student detail unavailable'>
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
          className='bg-white/85 dark:bg-slate-950/60'
        >
          <a
            href='/dashboard'
            className='inline-flex rounded-sm text-sm font-medium text-sky-700 transition-colors duration-150 hover:text-sky-900 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:outline-none dark:text-sky-300 dark:hover:text-sky-200 dark:focus-visible:ring-offset-slate-950'
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
      'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-300/30 dark:bg-amber-300/10 dark:text-amber-100',
    neutral:
      'border-slate-300 bg-slate-50 text-slate-700 dark:border-white/15 dark:bg-white/5 dark:text-slate-200',
    positive:
      'border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-300/30 dark:bg-emerald-300/10 dark:text-emerald-100',
  }[view.status.tone]

  return (
    <AdminSurface
      className='overflow-hidden border-sky-200/80 bg-sky-50/50 shadow-sm dark:border-sky-400/15 dark:bg-sky-400/5'
      contentClassName='p-4 sm:p-6'
    >
      <AdminSurfaceHeader
        title='Teacher snapshot'
        description='What to know and what to do next.'
        action={renderStatusBadge(view.status.label, statusClassName)}
      />

      <div className='mt-5 grid gap-3 lg:grid-cols-2'>
        <AdminInsetPanel className='rounded-lg border-sky-200 bg-white p-4 shadow-xs lg:col-span-2 dark:border-sky-300/15 dark:bg-slate-950/55'>
          <p className='text-xs font-semibold tracking-[0.12em] text-sky-700 uppercase dark:text-sky-300'>
            Recommended next step
          </p>
          <h3 className='mt-2 text-lg font-semibold text-pretty text-slate-950 dark:text-white'>
            {view.recommendation.title}
          </h3>
          <p className='mt-1 max-w-3xl text-sm leading-6 text-slate-600 dark:text-slate-300'>
            {view.recommendation.description}
          </p>
        </AdminInsetPanel>

        <StudentTopicSummary view={view} />
        <StudentRecentActivity view={view} />
      </div>

      <dl className='mt-3 grid grid-cols-2 gap-px overflow-hidden rounded-lg bg-slate-200/80 sm:grid-cols-4 dark:bg-white/10'>
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
    <AdminInsetPanel className='rounded-lg bg-white p-4 dark:bg-slate-950/55'>
      <h3 className='text-sm font-semibold text-slate-950 dark:text-white'>
        Strengths & struggles
      </h3>
      <p className='mt-1 text-sm leading-5 text-slate-500 dark:text-slate-400'>
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
      <p className='text-xs font-medium text-slate-500 dark:text-slate-400'>
        {label}
      </p>
      <div className='mt-1.5 flex flex-wrap gap-1.5'>
        {items.length > 0 ? (
          items.map((item) => (
            <Badge
              className={
                tone === 'strength'
                  ? 'h-auto border-emerald-200 bg-emerald-50 py-1 text-emerald-800 tabular-nums dark:border-emerald-300/20 dark:bg-emerald-300/10 dark:text-emerald-100'
                  : 'h-auto border-amber-200 bg-amber-50 py-1 text-amber-900 tabular-nums dark:border-amber-300/20 dark:bg-amber-300/10 dark:text-amber-100'
              }
              key={item.topic_id}
              variant='outline'
            >
              {formatTopicLabel(item.topic_id)}{' '}
              {Math.round(item.mastery_score * 100)}%
            </Badge>
          ))
        ) : (
          <span className='text-sm text-slate-500 dark:text-slate-400'>
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
    <AdminInsetPanel className='rounded-lg bg-white p-4 dark:bg-slate-950/55'>
      <h3 className='text-sm font-semibold text-slate-950 dark:text-white'>
        Recent activity
      </h3>
      {view.hasConversations ? (
        <dl className='mt-4 grid grid-cols-2 gap-4'>
          <div>
            <dt className='text-xs text-slate-500 dark:text-slate-400'>
              Last 14 days
            </dt>
            <dd className='mt-1 text-2xl font-semibold tracking-tight text-slate-950 tabular-nums dark:text-white'>
              {view.recentMessageCount}
            </dd>
            <p className='text-xs text-slate-500 dark:text-slate-400'>
              messages
            </p>
          </div>
          <div>
            <dt className='text-xs text-slate-500 dark:text-slate-400'>
              Active days
            </dt>
            <dd className='mt-1 text-2xl font-semibold tracking-tight text-slate-950 tabular-nums dark:text-white'>
              {view.activeDays}
            </dd>
            <p className='text-xs text-slate-500 dark:text-slate-400'>
              of 14 days
            </p>
          </div>
          <div className='col-span-2 border-t border-slate-200 pt-3 dark:border-white/10'>
            <dt className='text-xs text-slate-500 dark:text-slate-400'>
              Latest message
            </dt>
            <dd className='mt-1 text-sm font-medium text-slate-800 tabular-nums dark:text-slate-200'>
              {formatAdminDateTime(view.latestActivityAt)}
            </dd>
          </div>
        </dl>
      ) : (
        <div className='mt-4 rounded-md bg-slate-50 p-3 text-sm leading-5 text-slate-600 dark:bg-white/5 dark:text-slate-300'>
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
    <div className='min-w-0 bg-white px-3 py-3 dark:bg-slate-950/55'>
      <dt className='truncate text-[11px] font-medium text-slate-500 dark:text-slate-400'>
        {label}
      </dt>
      <dd className='mt-1 truncate text-sm font-semibold text-slate-950 tabular-nums dark:text-white'>
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
                stroke='#0284c7'
                fill='#38bdf8'
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
                  <p className='min-w-0 truncate text-sm font-semibold text-slate-900 dark:text-slate-100'>
                    {formatTopicLabel(item.topic_id)}
                  </p>
                  <span className='shrink-0 text-sm font-semibold text-slate-700 tabular-nums dark:text-slate-200'>
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
                    <dt className='text-slate-500 dark:text-slate-400'>
                      Last studied
                    </dt>
                    <dd className='mt-0.5 text-slate-700 tabular-nums dark:text-slate-200'>
                      {item.last_studied_at
                        ? formatAdminDateTime(item.last_studied_at)
                        : 'Not recorded yet'}
                    </dd>
                  </div>
                  <div>
                    <dt className='text-slate-500 dark:text-slate-400'>
                      Next review
                    </dt>
                    <dd className='mt-0.5 text-slate-700 tabular-nums dark:text-slate-200'>
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
          className='overflow-x-auto rounded-sm pb-2 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:outline-none dark:focus-visible:ring-offset-slate-950'
          tabIndex={0}
        >
          <div className='grid min-w-[42rem] grid-cols-[repeat(14,minmax(0,1fr))] gap-2'>
            {view.activityGrid.map((item) => (
              <div className='space-y-1.5 text-center' key={item.date}>
                <div
                  aria-label={`${item.shortLabel}: ${item.count} messages`}
                  className={`flex h-10 items-center justify-center rounded-md border border-white/60 text-xs font-semibold tabular-nums shadow-inner dark:border-white/10 ${getActivityTone(item.level)}`}
                  title={`${item.shortLabel}: ${item.count} messages`}
                >
                  {item.count}
                </div>
                <p className='text-[10px] text-slate-500 tabular-nums dark:text-slate-400'>
                  {item.shortLabel}
                </p>
              </div>
            ))}
          </div>
        </section>
        <div className='flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400'>
          <span>Less active</span>
          {[0, 1, 2, 3, 4].map((level) => (
            <span
              className={`inline-flex size-3 rounded-sm border border-white/60 dark:border-white/10 ${getActivityTone(level)}`}
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
        description='Raw tutoring messages for context.'
        action={renderConversationCount(view.conversations.length)}
      />
      <section
        aria-label='Conversation history'
        className='mt-4 max-h-[32rem] space-y-2 overflow-y-auto overscroll-contain rounded-sm pr-1 focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:outline-none dark:focus-visible:ring-offset-slate-950'
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
                ? 'bg-slate-50 dark:border-white/10 dark:bg-slate-900/80'
                : 'bg-sky-50 dark:border-sky-400/20 dark:bg-sky-400/10'
            }`}
            key={item.id}
          >
            <div className='mb-2 flex flex-wrap items-center justify-between gap-1 text-xs font-medium text-slate-500 dark:text-slate-400'>
              <span>{item.role}</span>
              <span className='tabular-nums'>
                {formatAdminDateTime(item.timestamp)}
              </span>
            </div>
            <p className='text-sm leading-6 break-words text-slate-700 dark:text-slate-200'>
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
      <dl className='mt-4 grid gap-px overflow-hidden rounded-lg bg-slate-200 sm:grid-cols-2 lg:grid-cols-4 dark:bg-white/10'>
        <StudentProfileField label='Form' value={detail.student.form} />
        <StudentProfileField
          label='Channel'
          value={detail.student.channel}
          valueClassName='capitalize'
        />
        <StudentProfileField
          label='External ID'
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
    <div className='min-w-0 bg-white p-3 dark:bg-slate-950/45'>
      <dt className='text-xs font-medium text-slate-500 dark:text-slate-400'>
        {label}
      </dt>
      <dd
        className={`mt-1 text-sm font-medium text-slate-900 tabular-nums dark:text-slate-100 ${valueClassName ?? ''}`}
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
    <div className='w-full rounded-lg border border-dashed border-slate-300 bg-slate-50/70 p-4 dark:border-white/15 dark:bg-white/5'>
      <h3 className='text-sm font-semibold text-slate-900 dark:text-slate-100'>
        {title}
      </h3>
      <p className='mt-1 text-sm leading-5 text-slate-500 dark:text-slate-400'>
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
