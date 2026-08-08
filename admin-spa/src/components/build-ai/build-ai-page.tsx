/* oxlint-disable react-perf/jsx-no-new-array-as-prop, react-perf/jsx-no-new-function-as-prop -- This local illustrative state machine intentionally keeps event transitions beside their controls; child views are not memoized, so callback identity does not affect rendering. */
import { useCallback, useId, useState } from 'react'
import {
  ActivityIcon,
  AlertTriangleIcon,
  ArrowLeftIcon,
  BookOpenIcon,
  CheckCircle2Icon,
  CircleDotIcon,
  FlaskConicalIcon,
  GaugeIcon,
  RadioIcon,
  SaveIcon,
  SendIcon,
  Settings2Icon,
} from 'lucide-react'
import type { ReactNode } from 'react'
import type { BuildAIPageKey } from '@/lib/build-ai-search'

import { AdminSurface } from '@/components/shared/admin-surface'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { cn } from '@/lib/utils'

type BuildPage = BuildAIPageKey
type TestState = 'out-of-date' | 'passed'
type PreviewState = 'not-run' | 'ready' | 'out-of-date'

const destinations: ReadonlyArray<{
  id: BuildPage
  label: string
  icon: typeof GaugeIcon
}> = [
  { id: 'overview', label: 'Overview', icon: GaugeIcon },
  { id: 'curriculum', label: 'Curriculum', icon: BookOpenIcon },
  { id: 'teaching', label: 'Teaching', icon: Settings2Icon },
  { id: 'test', label: 'Test tutor', icon: FlaskConicalIcon },
  { id: 'publish', label: 'Publish', icon: SendIcon },
  { id: 'activity', label: 'Activity', icon: ActivityIcon },
]

/**
 * Renders the synthetic Build AI product model used to evaluate the approved
 * information architecture. State is intentionally local and never claims to
 * persist or represent production Tutor data.
 */
export function BuildAIPage({
  onPageChange,
  page,
}: {
  onPageChange: (page: BuildAIPageKey) => void
  page: BuildAIPageKey
}) {
  const [draftSaved, setDraftSaved] = useState(true)
  const [testState, setTestState] = useState<TestState>('out-of-date')
  const [previewState, setPreviewState] = useState<PreviewState>('not-run')
  const [scenario, setScenario] = useState('fractions')
  const [replyLanguage, setReplyLanguage] = useState('follow')
  const [versionNote, setVersionNote] = useState('')
  const [published, setPublished] = useState(false)
  const [educatorReviewComplete, setEducatorReviewComplete] = useState(true)
  const [draftBase, setDraftBase] = useState('3')
  const [announcement, setAnnouncement] = useState('')

  const navigate = useCallback(
    (nextPage: BuildPage) => {
      onPageChange(nextPage)
    },
    [onPageChange],
  )

  const markDraftChanged = useCallback((message: string) => {
    setDraftSaved(false)
    setTestState('out-of-date')
    setEducatorReviewComplete(false)
    setPreviewState((current) =>
      current === 'ready' ? 'out-of-date' : current,
    )
    setAnnouncement(`${message} The Draft has unsaved changes.`)
  }, [])

  const saveDraft = useCallback(() => {
    setDraftSaved(true)
    setAnnouncement(
      'Draft saved. Existing Published versions and classes are unchanged.',
    )
  }, [])

  const runTests = useCallback(() => {
    setTestState('passed')
    setAnnouncement(
      'Synthetic Test runner completed: required results passed for the exact saved Draft.',
    )
  }, [])

  const publishVersion = useCallback(() => {
    setPublished(true)
    setDraftBase('new')
    setAnnouncement('Published version 4 is available. No classes changed.')
  }, [])

  const startDraft = useCallback(() => {
    setDraftBase('2')
    setDraftSaved(false)
    setTestState('out-of-date')
    setEducatorReviewComplete(false)
    setAnnouncement(
      'Private Draft started from Published version 2. Published version 3 remains available and no classes changed.',
    )
  }, [])

  return (
    <section className='mx-auto w-full max-w-7xl px-4 py-5 sm:px-6 lg:px-8'>
      <div
        className='sr-only'
        aria-live='polite'
        aria-atomic='true'
        data-testid='build-ai-live-region'
      >
        {announcement}
      </div>
      <header className='mb-5 flex flex-wrap items-start justify-between gap-4 border-b border-border pb-5'>
        <div className='min-w-0'>
          <p className='text-sm font-medium text-muted-foreground'>
            P&amp;AI Tutor · Platform scope · All authorized schools
          </p>
          <p className='mt-1 text-sm text-muted-foreground'>
            Synthetic illustrative data only. This preview is not connected to
            production contracts or learner records.
          </p>
        </div>
        <Badge variant='outline' className='min-h-8 gap-1.5'>
          <CircleDotIcon aria-hidden='true' className='size-3.5' />
          Platform operator
        </Badge>
      </header>

      <MobileNavigation page={page} navigate={navigate} />
      <div className='grid min-w-0 gap-8 lg:grid-cols-[13rem_minmax(0,1fr)]'>
        <Navigation page={page} navigate={navigate} />
        <div className='min-w-0' id='build-ai-content'>
          {page === 'overview' ? (
            <Overview
              draftBase={draftBase}
              draftSaved={draftSaved}
              educatorReviewComplete={educatorReviewComplete}
              navigate={navigate}
              testState={testState}
            />
          ) : null}
          {page === 'curriculum' ? (
            <Curriculum
              navigate={navigate}
              previewState={previewState}
              scenario={scenario}
              setPreviewState={setPreviewState}
              setScenario={(value) => {
                setScenario(value)
                setPreviewState((current) =>
                  current === 'ready' ? 'out-of-date' : current,
                )
              }}
            />
          ) : null}
          {page === 'teaching' ? (
            <Teaching
              draftSaved={draftSaved}
              markDraftChanged={markDraftChanged}
              navigate={navigate}
              previewState={previewState}
              replyLanguage={replyLanguage}
              saveDraft={saveDraft}
              setPreviewState={setPreviewState}
              setReplyLanguage={setReplyLanguage}
            />
          ) : null}
          {page === 'test' ? (
            <TestTutor
              draftSaved={draftSaved}
              navigate={navigate}
              runTests={runTests}
              testState={testState}
            />
          ) : null}
          {page === 'publish' ? (
            <Publish
              draftSaved={draftSaved}
              educatorReviewComplete={educatorReviewComplete}
              navigate={navigate}
              published={published}
              publishVersion={publishVersion}
              startDraft={startDraft}
              testState={testState}
              versionNote={versionNote}
              setVersionNote={setVersionNote}
            />
          ) : null}
          {page === 'activity' ? <Activity navigate={navigate} /> : null}
        </div>
      </div>
    </section>
  )
}

function Navigation({
  navigate,
  page,
}: {
  navigate: (page: BuildPage) => void
  page: BuildPage
}) {
  return (
    <nav className='hidden lg:block' aria-label='Build AI destinations'>
      <DestinationList navigate={navigate} page={page} />
    </nav>
  )
}

function MobileNavigation({
  navigate,
  page,
}: {
  navigate: (page: BuildPage) => void
  page: BuildPage
}) {
  return (
    <details className='mb-6 rounded-xl border border-border bg-card lg:hidden'>
      <summary className='flex min-h-11 cursor-pointer items-center px-4 py-2 font-medium focus-visible:outline-2 focus-visible:outline-offset-2'>
        Build AI destinations · {pageLabel(page)}
      </summary>
      <nav
        className='border-t border-border p-2'
        aria-label='Build AI destinations (mobile)'
      >
        <DestinationList navigate={navigate} page={page} />
      </nav>
    </details>
  )
}

function DestinationList({
  navigate,
  page,
}: {
  navigate: (page: BuildPage) => void
  page: BuildPage
}) {
  return (
    <ul className='grid gap-1'>
      {destinations.map(({ icon: Icon, id, label }) => (
        <li key={id}>
          <a
            aria-current={page === id ? 'page' : undefined}
            className={cn(
              'flex min-h-11 items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground outline-offset-2 hover:bg-muted hover:text-foreground focus-visible:outline-2',
              page === id && 'bg-muted text-foreground',
            )}
            href={`?page=${id}`}
            onClick={(event) => {
              if (
                event.button !== 0 ||
                event.metaKey ||
                event.ctrlKey ||
                event.shiftKey ||
                event.altKey
              ) {
                return
              }
              event.preventDefault()
              navigate(id)
            }}
          >
            <Icon aria-hidden='true' className='size-4 shrink-0' />
            <span>{label}</span>
            {id === 'activity' ? (
              <span aria-hidden='true' className='ml-auto text-xs font-normal'>
                Monitor
              </span>
            ) : null}
          </a>
        </li>
      ))}
    </ul>
  )
}

function PageHeader({
  children,
  page,
  title,
}: {
  children: ReactNode
  page: BuildPage
  title: string
}) {
  return (
    <header className='mb-7'>
      <nav
        aria-label='Breadcrumb'
        className='mb-2 text-sm text-muted-foreground'
      >
        <ol className='flex items-center gap-2'>
          <li>Build AI</li>
          <li aria-hidden='true'>/</li>
          <li aria-current='page'>{pageLabel(page)}</li>
        </ol>
      </nav>
      <h1
        className='text-3xl font-semibold tracking-tight text-foreground'
        tabIndex={-1}
      >
        {title}
      </h1>
      <div className='mt-3 max-w-3xl text-base leading-7 text-muted-foreground'>
        {children}
      </div>
    </header>
  )
}

function Overview({
  draftBase,
  draftSaved,
  educatorReviewComplete,
  navigate,
  testState,
}: {
  draftBase: string
  draftSaved: boolean
  educatorReviewComplete: boolean
  navigate: (page: BuildPage) => void
  testState: TestState
}) {
  return (
    <div>
      <PageHeader page='overview' title='P&AI Tutor'>
        {draftSaved
          ? 'Draft saved. Published version 3 remains available to teachers, and four authorized classes use it. Test results are out of date after Tutor reply language changed.'
          : 'The Draft has unsaved changes. Published version 3 remains available to teachers, and four authorized classes still use it. Learners are not affected.'}
      </PageHeader>
      <section aria-labelledby='destination-map-heading' className='mb-8'>
        <h2 id='destination-map-heading' className='mb-3 text-lg font-semibold'>
          Build AI map
        </h2>
        <ol className='grid gap-2 sm:grid-cols-2 xl:grid-cols-3'>
          {destinations.map(({ id, label }) => (
            <li key={id}>
              <TextLink navigate={navigate} page={id}>
                {label} ·{' '}
                {id === 'overview'
                  ? 'Current'
                  : id === 'activity'
                    ? 'Monitor'
                    : pageSummary(id)}
              </TextLink>
            </li>
          ))}
        </ol>
      </section>
      <AdminSurface className='shadow-none'>
        <h2 className='text-lg font-semibold'>Tutor state</h2>
        <dl className='mt-4 grid gap-5 border-t border-border pt-5 sm:grid-cols-2'>
          <StateItem term='Curriculum'>
            Pandai KSSM Mathematics Form 1 · revision 2026.08 · approved and
            validated.
          </StateItem>
          <StateItem term='Teaching'>
            {draftSaved
              ? 'Saved bounded preferences.'
              : 'Unsaved bounded preference changes. Learners are not affected.'}
          </StateItem>
          <StateItem term='Test results'>
            {testLabel(testState)} for Draft revision D-18.
          </StateItem>
          <StateItem term='Educator review'>
            {educatorReviewComplete
              ? 'Complete for the exact saved Draft.'
              : 'In progress. Educator review is recorded independently from Test results.'}
          </StateItem>
        </dl>
      </AdminSurface>
      {draftSaved && testState === 'out-of-date' ? (
        <Button className='mt-5' onClick={() => navigate('test')}>
          Run tests
        </Button>
      ) : null}
      <section
        className='mt-8 border-t border-border pt-7'
        aria-labelledby='published-heading'
      >
        <h2 id='published-heading' className='text-lg font-semibold'>
          Published version 3 is available
        </h2>
        <p className='mt-2 text-muted-foreground'>
          Four authorized classes use this immutable version. The private Draft
          is based on Published version {draftBase === 'new' ? '4' : draftBase}.
        </p>
        <div className='mt-3 flex flex-wrap gap-x-5 gap-y-2'>
          <TextLink navigate={navigate} page='publish'>
            Version history
          </TextLink>
          <TextLink navigate={navigate} page='publish'>
            Used by classes
          </TextLink>
        </div>
      </section>
      <section
        className='mt-8 border-t border-border pt-7'
        aria-labelledby='health-heading'
      >
        <h2 id='health-heading' className='text-lg font-semibold'>
          Operational context
        </h2>
        <p className='mt-2 text-muted-foreground'>
          Monitoring is not connected, so Tutor health cannot be confirmed. No
          production health claim is shown.
        </p>
        <div className='mt-3'>
          <TextLink navigate={navigate} page='activity'>
            Open Activity
          </TextLink>
        </div>
      </section>
    </div>
  )
}

function Curriculum({
  navigate,
  previewState,
  scenario,
  setPreviewState,
  setScenario,
}: {
  navigate: (page: BuildPage) => void
  previewState: PreviewState
  scenario: string
  setPreviewState: (state: PreviewState) => void
  setScenario: (value: string) => void
}) {
  const scenarioID = useId()
  const [narrowPreviewOpen, setNarrowPreviewOpen] = useState(false)
  return (
    <div>
      <PageHeader page='curriculum' title='Curriculum'>
        Pandai KSSM Mathematics Form 1, aligned to KSSM, uses approved immutable
        revision 2026.08. It is validated and differs from Published version 3
        only through the private Draft.
      </PageHeader>
      <div className='grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(19rem,0.8fr)]'>
        <div className='space-y-7'>
          <AdminSurface className='shadow-none'>
            <h2 className='text-lg font-semibold'>Source and revision</h2>
            <dl className='mt-4 grid gap-4 sm:grid-cols-2'>
              <StateItem term='Approved bundle'>
                Mathematics · Form 1 · all approved objectives
              </StateItem>
              <StateItem term='Revision'>
                2026.08 · approved 18 Aug 2026
              </StateItem>
              <StateItem term='Language coverage'>
                English and Bahasa Melayu
              </StateItem>
              <StateItem term='Validation'>
                24 objectives supported · 1 wording warning
              </StateItem>
            </dl>
            <details className='mt-5 border-t border-border pt-4 text-sm'>
              <summary className='min-h-10 cursor-pointer py-2 font-medium'>
                Exact snapshot identifier
              </summary>
              <p className='mt-2 break-all text-muted-foreground'>
                curriculum-snapshot-2026-08-approved
              </p>
            </details>
          </AdminSurface>
          <section aria-labelledby='grounding-heading'>
            <h2 id='grounding-heading' className='text-lg font-semibold'>
              Grounding example
            </h2>
            <p className='mt-2 text-muted-foreground'>
              Synthetic scenario: explain equivalent fractions using approved
              Form 1 examples.
            </p>
            <blockquote className='mt-3 border-l-2 border-primary pl-4 text-sm leading-6'>
              “Two fractions are equivalent when they describe the same part of
              a whole.”
            </blockquote>
            <p className='mt-2 text-sm text-muted-foreground'>
              Citation: Pandai Mathematics Form 1, Fractions, equivalent values.
            </p>
          </section>
        </div>
        <div className='hidden xl:block'>
          <PreviewPanel
            owner='Curriculum'
            previewState={previewState}
            scenario={scenario}
            scenarioID={`${scenarioID}-desktop`}
            setPreviewState={setPreviewState}
            setScenario={setScenario}
          />
        </div>
        <Button
          className='min-h-11 xl:hidden'
          variant='outline'
          onClick={() => setNarrowPreviewOpen(true)}
        >
          Open full-screen preview
        </Button>
        {narrowPreviewOpen ? (
          <NarrowScreenPreview onBack={() => setNarrowPreviewOpen(false)}>
            <PreviewPanel
              owner='Curriculum'
              previewState={previewState}
              scenario={scenario}
              scenarioID={`${scenarioID}-narrow`}
              setPreviewState={setPreviewState}
              setScenario={setScenario}
            />
          </NarrowScreenPreview>
        ) : null}
      </div>
      <div className='mt-7 flex flex-wrap gap-3'>
        <Button variant='outline'>Change curriculum</Button>
        <Button variant='outline' onClick={() => navigate('teaching')}>
          Continue to Teaching
        </Button>
      </div>
    </div>
  )
}

function PreviewPanel({
  owner,
  previewState,
  scenario,
  scenarioID,
  setPreviewState,
  setScenario,
}: {
  owner: 'Curriculum' | 'Teaching'
  previewState: PreviewState
  scenario: string
  scenarioID: string
  setPreviewState: (state: PreviewState) => void
  setScenario?: (value: string) => void
}) {
  return (
    <AdminSurface className='self-start shadow-none xl:sticky xl:top-4'>
      <h2 className='text-lg font-semibold'>Synthetic preview</h2>
      <p className='mt-2 text-sm leading-6 text-muted-foreground'>
        Uses current local {owner} values, including unsaved changes. Learners
        are not affected and this never becomes Test results.
      </p>
      {setScenario ? (
        <div className='mt-5'>
          <Label htmlFor={scenarioID}>Approved scenario</Label>
          <NativeSelect
            className='mt-2 w-full'
            id={scenarioID}
            value={scenario}
            onChange={(event) => setScenario(event.target.value)}
          >
            <NativeSelectOption value='fractions'>
              Equivalent fractions · English
            </NativeSelectOption>
            <NativeSelectOption value='ratio'>
              Ratios · Bahasa Melayu
            </NativeSelectOption>
          </NativeSelect>
        </div>
      ) : null}
      <Button
        className='mt-5 min-h-11'
        variant='outline'
        onClick={() => setPreviewState('ready')}
      >
        {previewState === 'not-run' ? 'Run preview' : 'Run preview again'}
      </Button>
      {previewState !== 'not-run' ? (
        <div className='mt-5 border-t border-border pt-5'>
          <Status tone={previewState === 'ready' ? 'positive' : 'warning'}>
            {previewState === 'ready' ? 'Current local preview' : 'Out of date'}
          </Status>
          <p className='mt-3 text-sm leading-6'>
            Synthetic preview using{' '}
            {owner === 'Teaching'
              ? 'unsaved Teaching changes'
              : 'curriculum revision 2026.08'}
            . Learners are not affected.
          </p>
          <p className='mt-3 text-sm text-muted-foreground'>
            A learner can compare fractions by using a shared denominator, then
            explain why the values match.
          </p>
        </div>
      ) : null}
    </AdminSurface>
  )
}

function NarrowScreenPreview({
  children,
  onBack,
}: {
  children: ReactNode
  onBack: () => void
}) {
  return (
    <div
      className='fixed inset-0 z-50 overflow-y-auto bg-background p-4 xl:hidden'
      data-testid='narrow-screen-preview'
    >
      <Button className='mb-4 min-h-11' variant='outline' onClick={onBack}>
        <ArrowLeftIcon aria-hidden='true' />
        Back
      </Button>
      {children}
    </div>
  )
}

function Teaching({
  draftSaved,
  markDraftChanged,
  navigate,
  previewState,
  replyLanguage,
  saveDraft,
  setPreviewState,
  setReplyLanguage,
}: {
  draftSaved: boolean
  markDraftChanged: (message: string) => void
  navigate: (page: BuildPage) => void
  previewState: PreviewState
  replyLanguage: string
  saveDraft: () => void
  setPreviewState: (state: PreviewState) => void
  setReplyLanguage: (value: string) => void
}) {
  const [narrowPreviewOpen, setNarrowPreviewOpen] = useState(false)
  return (
    <div>
      <PageHeader page='teaching' title='Teaching'>
        Set bounded preferences for the private Draft. Explicit learner
        requests, demonstrated progress, curriculum evidence, and locked
        safeguards take priority.
      </PageHeader>
      <div className='grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(19rem,0.8fr)]'>
        <div className='space-y-6'>
          <TeachingFieldset
            legend='How the Tutor starts'
            name='start'
            options={[
              'Adapt to the learner',
              'Start with one guiding question',
              'Start with one small worked example',
              'Start with a short explanation',
            ]}
            onChange={() => markDraftChanged('Tutor start preference changed.')}
          />
          <TeachingFieldset
            legend='Typical response detail'
            name='detail'
            options={[
              'Adapt to the learner',
              'Brief',
              'Balanced',
              'More detailed when useful',
            ]}
            onChange={() => markDraftChanged('Response detail changed.')}
          />
          <fieldset className='rounded-xl border border-border p-4'>
            <legend className='px-1 font-semibold'>Tutor reply language</legend>
            <p className='mb-3 text-sm text-muted-foreground'>
              Limited to approved curriculum language coverage.
            </p>
            {[
              [
                'follow',
                'Follow the learner’s English, Bahasa Melayu, or natural mix',
              ],
              ['english', 'Prefer English'],
              ['bm', 'Prefer Bahasa Melayu'],
            ].map(([value, label]) => (
              <label
                key={value}
                className='flex min-h-11 cursor-pointer items-start gap-3 py-2 text-sm'
              >
                <input
                  className='mt-0.5 size-4'
                  type='radio'
                  name='language'
                  value={value}
                  checked={replyLanguage === value}
                  onChange={() => {
                    setReplyLanguage(value)
                    markDraftChanged('Tutor reply language changed.')
                  }}
                />
                <span>{label}</span>
              </label>
            ))}
          </fieldset>
          <TeachingFieldset
            legend='Tone'
            name='tone'
            options={[
              'Calm and clear',
              'Warm and encouraging',
              'Direct and concise',
            ]}
            onChange={() => markDraftChanged('Tone changed.')}
          />
          <section
            className='border-t border-border pt-6'
            aria-labelledby='locked-heading'
          >
            <h2 id='locked-heading' className='text-lg font-semibold'>
              Locked behavior
            </h2>
            <ul className='mt-3 list-disc space-y-2 pl-5 text-sm text-muted-foreground'>
              <li>
                Hints, practice, and assessment are controlled by the teaching
                policy.
              </li>
              <li>
                AI identity, uncertainty, and approved curriculum boundaries
                cannot be disabled.
              </li>
              <li>
                Privacy boundaries are controlled by platform policy and tested
                before publication.
              </li>
            </ul>
          </section>
          <section
            className='border-t border-border pt-6'
            aria-labelledby='adaptation-heading'
          >
            <h2 id='adaptation-heading' className='text-lg font-semibold'>
              Class-adaptation bounds
            </h2>
            <p className='mt-2 text-sm text-muted-foreground'>
              Teachers may choose help-first preference, pacing/detail, reply
              language within coverage, and current-topic emphasis. Free-form
              class instructions are not available.
            </p>
          </section>
        </div>
        <div className='hidden xl:block'>
          <PreviewPanel
            owner='Teaching'
            previewState={previewState}
            scenario='fractions'
            scenarioID='teaching-preview-desktop'
            setPreviewState={setPreviewState}
          />
        </div>
        <Button
          className='min-h-11 xl:hidden'
          variant='outline'
          onClick={() => setNarrowPreviewOpen(true)}
        >
          Open full-screen preview
        </Button>
        {narrowPreviewOpen ? (
          <NarrowScreenPreview onBack={() => setNarrowPreviewOpen(false)}>
            <PreviewPanel
              owner='Teaching'
              previewState={previewState}
              scenario='fractions'
              scenarioID='teaching-preview-narrow'
              setPreviewState={setPreviewState}
            />
          </NarrowScreenPreview>
        ) : null}
      </div>
      <div className='mt-7 flex flex-wrap gap-3'>
        <Button className='min-h-11' disabled={draftSaved} onClick={saveDraft}>
          <SaveIcon aria-hidden='true' />
          Save Draft
        </Button>
        <Button
          variant='outline'
          disabled={draftSaved}
          onClick={() => window.location.reload()}
        >
          Discard changes
        </Button>
        <Button
          variant='outline'
          disabled={!draftSaved}
          onClick={() => navigate('test')}
        >
          Continue to Test tutor
        </Button>
      </div>
    </div>
  )
}

function TeachingFieldset({
  legend,
  name,
  onChange,
  options,
}: {
  legend: string
  name: string
  onChange: () => void
  options: ReadonlyArray<string>
}) {
  return (
    <fieldset className='rounded-xl border border-border p-4'>
      <legend className='px-1 font-semibold'>{legend}</legend>
      <p className='mb-3 text-sm text-muted-foreground'>
        The Tutor may adapt when learner evidence calls for it.
      </p>
      {options.map((option, index) => (
        <label
          key={option}
          className='flex min-h-11 cursor-pointer items-start gap-3 py-2 text-sm'
        >
          <input
            className='mt-0.5 size-4'
            defaultChecked={index === 0}
            name={name}
            onChange={onChange}
            type='radio'
          />
          <span>{option}</span>
        </label>
      ))}
    </fieldset>
  )
}

function TestTutor({
  draftSaved,
  navigate,
  runTests,
  testState,
}: {
  draftSaved: boolean
  navigate: (page: BuildPage) => void
  runTests: () => void
  testState: TestState
}) {
  const groups = [
    [
      'Curriculum grounding',
      testState === 'passed' ? 'Passed' : 'Out of date',
      'Edit Curriculum',
    ],
    [
      'Teaching behavior',
      testState === 'passed' ? 'Passed' : 'Needs review',
      'Edit Teaching',
    ],
    [
      'English, Bahasa Melayu, and mixed language',
      testState === 'passed' ? 'Passed' : 'Inconclusive',
      'Edit Teaching',
    ],
    [
      'Privacy boundaries',
      testState === 'passed' ? 'Passed' : 'Failed',
      'Rerun',
    ],
    [
      'Enabled-channel constraints',
      testState === 'passed' ? 'Passed' : 'Cancelled',
      'Rerun',
    ],
  ] as const
  return (
    <div>
      <PageHeader page='test' title='Test tutor'>
        Exact saved Draft D-18, saved by Nabila at 10:42, using curriculum
        revision 2026.08. These synthetic results are reproducible
        illustrations, not production evidence.
      </PageHeader>
      <div className='mb-6 flex flex-wrap items-center justify-between gap-4 border-b border-border pb-6'>
        <Status tone={testState === 'passed' ? 'positive' : 'warning'}>
          {testLabel(testState)}
        </Status>
        {!draftSaved ? (
          <Button onClick={() => navigate('teaching')}>
            Open Teaching to save
          </Button>
        ) : testState === 'out-of-date' ? (
          <Button onClick={runTests}>Run tests</Button>
        ) : (
          <Button onClick={() => navigate('publish')}>Open Publish</Button>
        )}
      </div>
      <ol className='space-y-4' aria-label='Required test groups'>
        {groups.map(([name, status, action]) => (
          <li key={name} className='rounded-xl border border-border p-4'>
            <div className='flex flex-wrap items-start justify-between gap-3'>
              <div>
                <h2 className='font-semibold'>{name}</h2>
                <p className='mt-1 text-sm text-muted-foreground'>
                  Required · synthetic approved scenario · rule-based evaluator
                  · Draft D-18
                </p>
              </div>
              <Status tone={status === 'Passed' ? 'positive' : 'warning'}>
                {status}
              </Status>
            </div>
            <p className='mt-3 text-sm'>
              Expected: stays within the approved behavior and curriculum
              boundary. Observed evidence is privacy-safe and contains no
              learner records.
            </p>
            <details className='mt-3 text-sm'>
              <summary className='min-h-10 cursor-pointer py-2 font-medium'>
                Reproduction details
              </summary>
              <p className='text-muted-foreground'>
                Approved synthetic case · curriculum revision 2026.08 · saved
                Draft D-18.
              </p>
            </details>
            <Button
              className='mt-3'
              variant='outline'
              onClick={() =>
                action === 'Edit Curriculum'
                  ? navigate('curriculum')
                  : action === 'Edit Teaching'
                    ? navigate('teaching')
                    : runTests()
              }
            >
              {action}
            </Button>
          </li>
        ))}
      </ol>
    </div>
  )
}

function Publish({
  draftSaved,
  educatorReviewComplete,
  navigate,
  published,
  publishVersion,
  setVersionNote,
  startDraft,
  testState,
  versionNote,
}: {
  draftSaved: boolean
  educatorReviewComplete: boolean
  navigate: (page: BuildPage) => void
  published: boolean
  publishVersion: () => void
  setVersionNote: (value: string) => void
  startDraft: () => void
  testState: TestState
  versionNote: string
}) {
  const noteID = useId()
  const ready = draftSaved && testState === 'passed'
  const rows = [
    [
      'Draft saved and current',
      draftSaved ? 'Ready' : 'Needs action',
      'Teaching',
    ],
    ['Curriculum revision approved and validated', 'Ready', 'Curriculum'],
    [
      'Teaching and grounding Test results passed and current',
      testState === 'passed' ? 'Ready' : 'Out of date',
      'Test tutor',
    ],
    [
      'Non-waivable safeguards and coverage passed and current',
      testState === 'passed' ? 'Ready' : 'Needs action',
      'Test tutor',
    ],
    [
      'Educator review complete',
      educatorReviewComplete ? 'Ready' : 'In progress',
      'Educator review',
    ],
    [
      'Version note and recovery ready',
      versionNote.trim() ? 'Ready' : 'Needs action',
      'Add note',
    ],
  ] as const
  const canPublish =
    ready && educatorReviewComplete && versionNote.trim().length > 0
  return (
    <div>
      <PageHeader page='publish' title='Publish'>
        One authoritative gate for the exact saved Draft. Published versions are
        immutable, and educator review cannot waive required safeguards.
      </PageHeader>
      {published ? (
        <div
          className='mb-7 rounded-xl border border-border bg-muted/40 p-4'
          role='status'
        >
          <Status tone='positive'>Published</Status>
          <p className='mt-2 font-medium'>
            Published version 4 is available. No classes changed.
          </p>
        </div>
      ) : null}
      <section aria-labelledby='readiness-heading'>
        <h2 id='readiness-heading' className='text-lg font-semibold'>
          Publication readiness
        </h2>
        <ol className='mt-4 divide-y divide-border border-y border-border'>
          {rows.map(([label, status, repair], index) => (
            <li
              className='grid gap-2 py-4 sm:grid-cols-[2rem_minmax(0,1fr)_auto] sm:items-center'
              key={label}
            >
              <span className='text-sm text-muted-foreground'>
                {index + 1}.
              </span>
              <div>
                <p className='font-medium'>{label}</p>
                <p className='mt-1 text-sm text-muted-foreground'>
                  Synthetic evidence checked at 10:48 · exact Draft D-18.
                </p>
              </div>
              <div className='flex items-center gap-3'>
                <Status
                  tone={
                    status === 'Ready'
                      ? 'positive'
                      : status === 'In progress'
                        ? 'info'
                        : 'warning'
                  }
                >
                  {status}
                </Status>
                {repair === 'Curriculum' ||
                repair === 'Teaching' ||
                repair === 'Test tutor' ? (
                  <button
                    className='min-h-10 text-sm font-medium underline underline-offset-4'
                    onClick={() =>
                      navigate(
                        repair === 'Curriculum'
                          ? 'curriculum'
                          : repair === 'Teaching'
                            ? 'teaching'
                            : 'test',
                      )
                    }
                    type='button'
                  >
                    {repair}
                  </button>
                ) : repair === 'Educator review' && !educatorReviewComplete ? (
                  <a
                    className='min-h-10 py-2 text-sm font-medium underline underline-offset-4'
                    href='?workspace=teach&page=reviews'
                  >
                    Request educator review in Teach
                  </a>
                ) : null}
              </div>
            </li>
          ))}
        </ol>
      </section>
      <div className='mt-7 max-w-xl'>
        <Label htmlFor={noteID}>Version note</Label>
        <Input
          className='mt-2 min-h-11 text-base sm:text-sm'
          id={noteID}
          value={versionNote}
          onChange={(event) => setVersionNote(event.target.value)}
          placeholder='What teachers should know about this version'
        />
      </div>
      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button className='mt-5 min-h-11' disabled={!canPublish}>
            Review publication
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Publish this version?</AlertDialogTitle>
            <AlertDialogDescription>
              It will become available to teachers, but no class will change
              until a teacher applies it.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep reviewing</AlertDialogCancel>
            <AlertDialogAction onClick={publishVersion}>
              Publish version
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <section
        className='mt-10 border-t border-border pt-7'
        aria-labelledby='history-heading'
      >
        <h2 id='history-heading' className='text-lg font-semibold'>
          Version history
        </h2>
        <ul className='mt-4 space-y-5'>
          {published ? (
            <li>
              <p className='font-medium'>
                Published version 4 · Latest available
              </p>
              <p className='mt-1 text-sm text-muted-foreground'>
                {versionNote.trim()} · published by Nabila · curriculum 2026.08
                · no authorized classes
              </p>
            </li>
          ) : null}
          <li>
            <p className='font-medium'>
              Published version 3 · {published ? 'Earlier' : 'Latest available'}
            </p>
            <p className='mt-1 text-sm text-muted-foreground'>
              Clearer mixed-language guidance · published by Nabila · curriculum
              2026.08 · 4 authorized classes
            </p>
          </li>
          <li>
            <p className='font-medium'>Published version 2 · Earlier</p>
            <p className='mt-1 text-sm text-muted-foreground'>
              Initial pilot · independently reviewed · curriculum 2026.06 · no
              authorized classes
            </p>
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button className='mt-3' variant='outline'>
                  Start Draft from this version
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>
                    Start Draft from version 2?
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    This replaces the existing private Draft. Published versions
                    remain immutable and available, and no classes change.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Keep existing Draft</AlertDialogCancel>
                  <AlertDialogAction onClick={startDraft}>
                    Replace Draft
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </li>
        </ul>
      </section>
      <section
        className='mt-10 border-t border-border pt-7'
        aria-labelledby='used-by-classes-heading'
      >
        <h2 id='used-by-classes-heading' className='text-lg font-semibold'>
          Used by classes
        </h2>
        <p className='mt-2 text-sm text-muted-foreground'>
          Four authorized classes use immutable Published version 3. Newly
          published version 4 is available to teachers but is not applied to any
          class.
        </p>
        <ul className='mt-4 grid gap-2 text-sm sm:grid-cols-2'>
          <li>Form 1 Amanah · version 3</li>
          <li>Form 1 Bestari · version 3</li>
          <li>Form 1 Cekal · version 3</li>
          <li>Form 1 Dinamik · version 3</li>
        </ul>
      </section>
    </div>
  )
}

function Activity({ navigate }: { navigate: (page: BuildPage) => void }) {
  return (
    <div>
      <PageHeader page='activity' title='Activity'>
        Monitoring is not connected. Tutor health cannot be confirmed, and this
        illustrative component does not create a synthetic production claim.
      </PageHeader>
      <section aria-labelledby='tutor-status-heading'>
        <h2 id='tutor-status-heading' className='text-xl font-semibold'>
          Tutor status
        </h2>
        <div className='mt-4 rounded-xl border border-border p-5'>
          <Status tone='warning'>Monitoring incomplete</Status>
          <p className='mt-3 text-sm leading-6 text-muted-foreground'>
            No trustworthy safeguarding, answer-quality, curriculum-grounding,
            or channel-delivery monitoring contract is connected. No learner
            identities or conversations are shown.
          </p>
        </div>
      </section>
      <section
        className='mt-8 border-t border-border pt-7'
        aria-labelledby='serving-heading'
      >
        <h2 id='serving-heading' className='text-lg font-semibold'>
          Published-version context
        </h2>
        <p className='mt-2 text-muted-foreground'>
          Published version 3 remains available to teachers and is used by four
          authorized classes.
        </p>
        <div className='mt-3 flex gap-5'>
          <TextLink navigate={navigate} page='publish'>
            Open Publish
          </TextLink>
          <TextLink navigate={navigate} page='test'>
            Open Test results
          </TextLink>
        </div>
      </section>
      <section
        className='mt-8 border-t border-border pt-7'
        aria-labelledby='events-heading'
      >
        <h2 id='events-heading' className='text-lg font-semibold'>
          Recent significant changes
        </h2>
        <ul className='mt-3 space-y-3 text-sm text-muted-foreground'>
          <li>
            Draft reply-language preference changed · synthetic · 12 minutes ago
          </li>
          <li>Published version 3 became available · synthetic · 3 days ago</li>
          <li>
            Four classes remain authorized to use version 3 · synthetic
            aggregate
          </li>
        </ul>
      </section>
    </div>
  )
}

function StateItem({ children, term }: { children: ReactNode; term: string }) {
  return (
    <div>
      <dt className='font-medium'>{term}</dt>
      <dd className='mt-1 text-sm leading-6 text-muted-foreground'>
        {children}
      </dd>
    </div>
  )
}

function Status({
  children,
  tone,
}: {
  children: ReactNode
  tone: 'positive' | 'warning' | 'info'
}) {
  const Icon =
    tone === 'positive'
      ? CheckCircle2Icon
      : tone === 'warning'
        ? AlertTriangleIcon
        : RadioIcon
  return (
    <span
      className={cn(
        'inline-flex min-h-7 items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium',
        tone === 'positive' &&
          'border-emerald-700/30 text-emerald-800 dark:text-emerald-300',
        tone === 'warning' &&
          'border-amber-700/30 text-amber-900 dark:text-amber-200',
        tone === 'info' && 'border-border text-foreground',
      )}
    >
      <Icon aria-hidden='true' className='size-3.5' />
      {children}
    </span>
  )
}

function TextLink({
  children,
  navigate,
  page,
}: {
  children: ReactNode
  navigate: (page: BuildPage) => void
  page: BuildPage
}) {
  return (
    <a
      className='inline-flex min-h-10 items-center text-sm font-medium text-foreground underline decoration-border underline-offset-4 hover:decoration-foreground focus-visible:outline-2 focus-visible:outline-offset-2'
      href={`?page=${page}`}
      onClick={(event) => {
        if (
          event.button !== 0 ||
          event.metaKey ||
          event.ctrlKey ||
          event.shiftKey ||
          event.altKey
        ) {
          return
        }
        event.preventDefault()
        navigate(page)
      }}
    >
      {children}
    </a>
  )
}

function pageLabel(page: BuildPage) {
  return destinations.find(({ id }) => id === page)?.label ?? 'Overview'
}
function pageSummary(page: BuildPage) {
  if (page === 'curriculum') return '2026.08'
  if (page === 'teaching') return 'Saved'
  if (page === 'test') return 'Out of date'
  if (page === 'publish') return 'Review'
  return ''
}
function testLabel(state: TestState) {
  if (state === 'passed') return 'Passed and current'
  return 'Out of date'
}
