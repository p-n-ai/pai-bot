import { useEffect, useMemo, useState } from 'react'
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
import { AdminInsetPanel } from '@/components/admin-inset-panel'
import { AdminSurface, AdminSurfaceHeader } from '@/components/admin-surface'
import { LoadState } from '@/components/load-state'
import { PageHero } from '@/components/page-hero'
import { StatePanel } from '@/components/state-panel'
import { StreakMetricsPanel } from '@/components/streak-metrics-panel'
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

  useEffect(() => {
    let mounted = true

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
  }, [studentID])

  if (loadState !== 'ready') {
    return (
      <LoadState
        error='Student information is not available right now.'
        errorTitle='Student detail unavailable'
        loadingTitle='Loading student detail'
        status={loadState}
      />
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
  const heroAside = useMemo(
    () => <StudentHeroAside detail={detail} />,
    [detail],
  )

  return (
    <div className='space-y-6'>
      {variant === 'page' ? (
        <PageHero
          eyebrow='Student detail'
          title={detail.student.name}
          description={`${detail.student.form} | ${detail.student.channel} | ${detail.student.external_id}`}
          aside={heroAside}
          className='bg-white/85 lg:grid-cols-[1.15fr_0.85fr] dark:bg-slate-950/60'
        >
          <a
            href='/dashboard'
            className='text-sm font-medium text-sky-700 hover:text-sky-900 dark:text-sky-300 dark:hover:text-sky-200'
          >
            Back to dashboard
          </a>
        </PageHero>
      ) : null}

      <section className='grid gap-4 xl:grid-cols-[0.75fr_1fr_0.9fr]'>
        <StudentProfileCard detail={detail} />
        <StudentMasteryRadar view={view} />
        <StudentStruggleAreas detail={detail} view={view} />
      </section>

      <StudentActivityGrid view={view} />
      <StudentConversationList conversations={conversations} view={view} />
    </div>
  )
}

function StudentHeroAside({ detail }: { detail: StudentDetail }) {
  return (
    <StreakMetricsPanel
      current={detail.streak.current}
      longest={detail.streak.longest}
      totalXP={detail.streak.total_xp}
    />
  )
}

function StudentProfileCard({ detail }: { detail: StudentDetail }) {
  return (
    <AdminSurface>
      <AdminSurfaceHeader title='Profile card' />
      <div className='mt-6 space-y-4'>
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
      </div>
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
    <AdminInsetPanel>
      <p className='text-xs font-semibold tracking-[0.18em] text-slate-500 uppercase dark:text-slate-400'>
        {label}
      </p>
      <p
        className={`mt-2 text-sm font-medium text-slate-900 dark:text-slate-100 ${valueClassName ?? ''}`}
      >
        {value}
      </p>
    </AdminInsetPanel>
  )
}

function StudentMasteryRadar({
  view,
}: {
  view: ReturnType<typeof buildStudentViewModel>
}) {
  return (
    <AdminSurface>
      <AdminSurfaceHeader title='Mastery radar' />
      <div className='mt-6 h-[320px]'>
        {view.radarData.length > 0 ? (
          <ResponsiveContainer height='100%' width='100%'>
            <RadarChart data={view.radarData}>
              <PolarGrid />
              <PolarAngleAxis dataKey='topic' />
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
            <StatePanel title='No mastery radar yet'>
              Progress details will appear after the student completes some
              work.
            </StatePanel>
          </div>
        )}
      </div>
    </AdminSurface>
  )
}

function StudentStruggleAreas({
  detail,
  view,
}: {
  detail: StudentDetail
  view: ReturnType<typeof buildStudentViewModel>
}) {
  return (
    <AdminSurface>
      <AdminSurfaceHeader title='Struggle areas' />
      <div className='mt-6 space-y-4'>
        <div className='flex flex-wrap gap-2'>
          {view.struggleAreas.length > 0 ? (
            view.struggleAreas.map((item) => (
              <span
                className='inline-flex min-h-7 items-center rounded-full bg-amber-100 px-3 py-1 text-sm font-medium text-amber-900 hover:bg-amber-100 dark:bg-amber-300/15 dark:text-amber-100 dark:hover:bg-amber-300/15'
                key={item.topic_id}
              >
                {formatTopicLabel(item.topic_id)}{' '}
                {Math.round(item.mastery_score * 100)}%
              </span>
            ))
          ) : (
            <StatePanel title='No active struggle areas'>
              This learner does not currently have any topics below the
              intervention threshold.
            </StatePanel>
          )}
        </div>

        <div className='space-y-3'>
          {view.hasProgress ? (
            detail.progress.map((item) => (
              <AdminInsetPanel key={item.topic_id}>
                <div className='flex items-center justify-between gap-3'>
                  <p className='text-sm font-medium text-slate-900 dark:text-slate-100'>
                    {formatTopicLabel(item.topic_id)}
                  </p>
                  <span className='text-xs tracking-[0.16em] text-slate-500 uppercase dark:text-slate-400'>
                    {Math.round(item.mastery_score * 100)}% mastery
                  </span>
                </div>
                <p className='mt-2 text-xs text-slate-500 dark:text-slate-400'>
                  Last studied:{' '}
                  {item.last_studied_at
                    ? formatAdminDateTime(item.last_studied_at)
                    : 'Not recorded yet'}
                </p>
                <p className='text-xs text-slate-500 dark:text-slate-400'>
                  Next review:{' '}
                  {item.next_review_at
                    ? formatAdminDateTime(item.next_review_at)
                    : 'To be scheduled'}
                </p>
              </AdminInsetPanel>
            ))
          ) : (
            <StatePanel title='No topic progress yet'>
              No topic progress has been recorded for this student yet.
            </StatePanel>
          )}
        </div>
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
        title='Activity grid'
        description='Conversation activity over the last 14 days.'
      />
      <div className='mt-6 space-y-4'>
        <div className='grid grid-cols-7 gap-2 md:grid-cols-[repeat(14,minmax(0,1fr))]'>
          {view.activityGrid.map((item) => (
            <div className='space-y-2 text-center' key={item.date}>
              <div
                aria-label={`${item.shortLabel}: ${item.count} messages`}
                className={`h-12 rounded-2xl border border-white/60 shadow-inner dark:border-white/10 ${getActivityTone(item.level)}`}
                title={`${item.shortLabel}: ${item.count} messages`}
              />
              <p className='text-[11px] tracking-[0.14em] text-slate-500 uppercase dark:text-slate-400'>
                {item.shortLabel}
              </p>
            </div>
          ))}
        </div>
        <div className='flex flex-wrap items-center gap-3 text-xs text-slate-500 dark:text-slate-400'>
          <span>Less active</span>
          {[0, 1, 2, 3, 4].map((level) => (
            <span
              className={`inline-flex size-4 rounded-full border border-white/60 dark:border-white/10 ${getActivityTone(level)}`}
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
  conversations,
  view,
}: {
  conversations: Array<StudentConversation>
  view: ReturnType<typeof buildStudentViewModel>
}) {
  return (
    <AdminSurface>
      <AdminSurfaceHeader title='Recent conversations' />
      <div className='mt-6 space-y-3'>
        {!view.hasConversations ? (
          <StatePanel title='No tutoring messages yet'>
            Recent tutoring messages will appear here once the student has
            chatted.
          </StatePanel>
        ) : null}
        {conversations.map((item) => (
          <AdminInsetPanel
            className={
              item.role === 'student'
                ? 'bg-slate-50 dark:border-white/10 dark:bg-slate-900/80'
                : 'bg-sky-50 dark:border-sky-400/20 dark:bg-sky-400/10'
            }
            key={item.id}
          >
            <div className='mb-2 flex items-center justify-between text-xs font-medium tracking-[0.18em] text-slate-500 uppercase dark:text-slate-400'>
              <span>{item.role}</span>
              <span>{formatAdminDateTime(item.timestamp)}</span>
            </div>
            <p className='text-sm leading-6 text-slate-700 dark:text-slate-200'>
              {item.text}
            </p>
          </AdminInsetPanel>
        ))}
      </div>
    </AdminSurface>
  )
}
