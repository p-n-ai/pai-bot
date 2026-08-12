/* oxlint-disable react-perf/jsx-no-new-array-as-prop, react-perf/jsx-no-new-function-as-prop -- This local illustrative state machine intentionally keeps event transitions beside their controls; child views are not memoized, so callback identity does not affect rendering. */
import { useCallback, useId, useReducer, useState } from 'react'
import type { ReactNode } from 'react'
import type { BuildAIPageKey } from '@/lib/build-ai-search'
import type { CharacterConfig } from '@/components/build-ai/character-creator'
import type { PandaiIconName } from '@/components/ui/pandai-icon'

import {
  CharacterCreator,
  characterSummary,
  defaultCharacterConfig,
} from '@/components/build-ai/character-creator'
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
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { PandaiIcon } from '@/components/ui/pandai-icon'
import { cn } from '@/lib/utils'

type BuildPage = BuildAIPageKey
type TestState = 'out-of-date' | 'passed'
type PreviewState = 'not-run' | 'ready' | 'out-of-date'
type TeachingSetting = 'start' | 'detail' | 'language' | 'tone'

interface TeachingSettings {
  readonly start: string
  readonly detail: string
  readonly language: string
  readonly tone: string
}

interface PublishedVersion {
  readonly authorizedClassCount: number
  readonly character: CharacterConfig
  readonly curriculumRevision: string
  readonly draftRevision: number
  readonly id: string
  readonly note: string
  readonly publisher: string
  readonly teachingSettings: TeachingSettings
}

const initialTeachingSettings: TeachingSettings = {
  start: 'Adapt to the learner',
  detail: 'Adapt to the learner',
  language: 'follow',
  tone: 'Calm and clear',
}

const initialPublishedVersions = [
  {
    authorizedClassCount: 4,
    character: defaultCharacterConfig,
    curriculumRevision: '2026.08',
    draftRevision: 17,
    id: '3',
    note: 'Clearer mixed-language guidance',
    publisher: 'Nabila',
    teachingSettings: initialTeachingSettings,
  },
  {
    authorizedClassCount: 0,
    character: defaultCharacterConfig,
    curriculumRevision: '2026.06',
    draftRevision: 12,
    id: '2',
    note: 'Initial pilot',
    publisher: 'Nabila',
    teachingSettings: {
      start: 'Start with one guiding question',
      detail: 'Brief',
      language: 'english',
      tone: 'Calm and clear',
    },
  },
] as const satisfies ReadonlyArray<PublishedVersion>

interface BuildAIWorkflowState {
  readonly announcement: string
  readonly character: CharacterConfig
  readonly draftBase: string
  readonly draftRevision: number
  readonly draftSaved: boolean
  readonly educatorReviewComplete: boolean
  readonly previewState: PreviewState
  readonly publishedVersions: ReadonlyArray<PublishedVersion>
  readonly savedCharacter: CharacterConfig
  readonly savedDraftBase: string
  readonly savedTeachingSettings: TeachingSettings
  readonly scenario: string
  readonly teachingSettings: TeachingSettings
  readonly testState: TestState
  readonly versionNote: string
}

type BuildAIWorkflowAction =
  | {
      readonly config: CharacterConfig
      readonly message: string
      readonly type: 'character-updated'
    }
  | { readonly type: 'draft-saved' }
  | { readonly type: 'changes-discarded' }
  | { readonly type: 'tests-run' }
  | { readonly type: 'educator-review-completed' }
  | { readonly type: 'version-published' }
  | { readonly type: 'draft-started' }
  | { readonly state: PreviewState; readonly type: 'preview-updated' }
  | { readonly scenario: string; readonly type: 'scenario-updated' }
  | {
      readonly message: string
      readonly setting: TeachingSetting
      readonly type: 'teaching-setting-updated'
      readonly value: string
    }
  | { readonly type: 'version-note-updated'; readonly value: string }

const initialBuildAIWorkflowState: BuildAIWorkflowState = {
  announcement: '',
  character: defaultCharacterConfig,
  draftBase: '3',
  draftRevision: 18,
  draftSaved: true,
  educatorReviewComplete: true,
  previewState: 'not-run',
  publishedVersions: initialPublishedVersions,
  savedCharacter: defaultCharacterConfig,
  savedDraftBase: '3',
  savedTeachingSettings: initialTeachingSettings,
  scenario: 'fractions',
  teachingSettings: initialTeachingSettings,
  testState: 'out-of-date',
  versionNote: '',
}

function markWorkflowChanged(
  state: BuildAIWorkflowState,
  message: string,
): Pick<BuildAIWorkflowState, 'announcement' | 'draftSaved' | 'previewState'> {
  return {
    announcement: `${message} The Draft has unsaved changes.`,
    draftSaved: false,
    previewState:
      state.previewState === 'ready' ? 'out-of-date' : state.previewState,
  }
}

function buildAIWorkflowReducer(
  state: BuildAIWorkflowState,
  action: BuildAIWorkflowAction,
): BuildAIWorkflowState {
  switch (action.type) {
    case 'character-updated':
      return {
        ...state,
        ...markWorkflowChanged(state, action.message),
        character: action.config,
      }
    case 'teaching-setting-updated':
      return {
        ...state,
        ...markWorkflowChanged(state, action.message),
        teachingSettings: {
          ...state.teachingSettings,
          [action.setting]: action.value,
        },
      }
    case 'draft-saved':
      return {
        ...state,
        announcement:
          'Draft saved. Existing Published versions and classes are unchanged. Test results and educator review must be refreshed for this saved revision.',
        draftRevision: state.draftRevision + 1,
        draftSaved: true,
        educatorReviewComplete: false,
        savedCharacter: state.character,
        savedDraftBase: state.draftBase,
        savedTeachingSettings: state.teachingSettings,
        testState: 'out-of-date',
        versionNote: '',
      }
    case 'changes-discarded':
      return {
        ...state,
        announcement: 'Unsaved Draft changes discarded.',
        character: state.savedCharacter,
        draftBase: state.savedDraftBase,
        draftSaved: true,
        previewState:
          state.previewState === 'ready' ? 'out-of-date' : state.previewState,
        teachingSettings: state.savedTeachingSettings,
      }
    case 'tests-run':
      if (!state.draftSaved) {
        return {
          ...state,
          announcement: 'Save the Draft before running required tests.',
        }
      }
      return {
        ...state,
        announcement:
          'Synthetic Test runner completed: required results passed for the exact saved Draft.',
        testState: 'passed',
      }
    case 'educator-review-completed':
      if (!state.draftSaved || state.testState !== 'passed') {
        return {
          ...state,
          announcement:
            'Save the Draft and complete required tests before educator review.',
        }
      }
      return {
        ...state,
        announcement:
          'Synthetic educator review completed for the exact saved Draft.',
        educatorReviewComplete: true,
      }
    case 'version-published': {
      const currentDraftPublished = state.publishedVersions.some(
        (version) => version.draftRevision === state.draftRevision,
      )
      if (currentDraftPublished) return state
      const nextIdentifier = String(
        state.publishedVersions.reduce(
          (maximum, { id }) => Math.max(maximum, Number(id)),
          0,
        ) + 1,
      )
      const publishedVersion: PublishedVersion = {
        authorizedClassCount: 0,
        character: state.character,
        curriculumRevision: '2026.08',
        draftRevision: state.draftRevision,
        id: nextIdentifier,
        note: state.versionNote.trim(),
        publisher: 'Nabila',
        teachingSettings: state.teachingSettings,
      }
      return {
        ...state,
        announcement: `Published version ${nextIdentifier} is available. No classes changed.`,
        draftBase: nextIdentifier,
        publishedVersions: [publishedVersion, ...state.publishedVersions],
        savedDraftBase: nextIdentifier,
      }
    }
    case 'draft-started': {
      const sourceVersion =
        state.publishedVersions.find((version) => version.id === '2') ??
        initialPublishedVersions[1]
      const latestPublishedVersion = state.publishedVersions[0]
      return {
        ...state,
        announcement: `Private Draft started from Published version ${sourceVersion.id}. Published version ${latestPublishedVersion.id} remains available and no classes changed.`,
        character: sourceVersion.character,
        draftBase: sourceVersion.id,
        draftSaved: false,
        educatorReviewComplete: false,
        teachingSettings: sourceVersion.teachingSettings,
        testState: 'out-of-date',
        versionNote: '',
      }
    }
    case 'preview-updated':
      return { ...state, previewState: action.state }
    case 'scenario-updated':
      return {
        ...state,
        previewState:
          state.previewState === 'ready' ? 'out-of-date' : state.previewState,
        scenario: action.scenario,
      }
    case 'version-note-updated':
      return { ...state, versionNote: action.value }
  }
}

const destinations: ReadonlyArray<{
  id: BuildPage
  label: string
  icon: PandaiIconName
}> = [
  { id: 'overview', label: 'Overview', icon: 'layout' },
  { id: 'character', label: 'P-Bot character', icon: 'star' },
  { id: 'curriculum', label: 'Curriculum', icon: 'book-open' },
  { id: 'teaching', label: 'Teaching', icon: 'settings' },
  { id: 'test', label: 'Test tutor', icon: 'zap' },
  { id: 'publish', label: 'Publish', icon: 'check-circle' },
  { id: 'activity', label: 'Activity', icon: 'activity' },
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
  const [workflow, dispatch] = useReducer(
    buildAIWorkflowReducer,
    initialBuildAIWorkflowState,
  )
  const {
    announcement,
    character,
    draftBase,
    draftRevision,
    draftSaved,
    educatorReviewComplete,
    previewState,
    publishedVersions,
    scenario,
    teachingSettings,
    testState,
    versionNote,
  } = workflow

  const currentDraftPublished = publishedVersions.some(
    (version) => version.draftRevision === draftRevision,
  )
  const effectiveEducatorReviewComplete = draftSaved && educatorReviewComplete
  const effectiveTestState = draftSaved ? testState : 'out-of-date'
  const latestPublishedVersion = publishedVersions[0]

  const navigate = useCallback(
    (nextPage: BuildPage) => {
      onPageChange(nextPage)
    },
    [onPageChange],
  )

  const updateTeachingSetting = useCallback(
    (setting: TeachingSetting, value: string, message: string) => {
      dispatch({ message, setting, type: 'teaching-setting-updated', value })
    },
    [],
  )

  const updateCharacter = useCallback(
    (config: CharacterConfig, message: string) => {
      dispatch({ config, message, type: 'character-updated' })
    },
    [],
  )
  const saveDraft = useCallback(() => dispatch({ type: 'draft-saved' }), [])
  const discardChanges = useCallback(
    () => dispatch({ type: 'changes-discarded' }),
    [],
  )
  const runTests = useCallback(() => dispatch({ type: 'tests-run' }), [])
  const completeEducatorReview = useCallback(
    () => dispatch({ type: 'educator-review-completed' }),
    [],
  )
  const publishVersion = useCallback(
    () => dispatch({ type: 'version-published' }),
    [],
  )
  const startDraft = useCallback(() => dispatch({ type: 'draft-started' }), [])
  const setPreviewState = useCallback(
    (state: PreviewState) => dispatch({ state, type: 'preview-updated' }),
    [],
  )
  const setScenario = useCallback(
    (nextScenario: string) =>
      dispatch({ scenario: nextScenario, type: 'scenario-updated' }),
    [],
  )
  const setVersionNote = useCallback(
    (value: string) => dispatch({ type: 'version-note-updated', value }),
    [],
  )

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
          <PandaiIcon aria-hidden='true' className='size-3.5' name='shield' />
          Platform operator
        </Badge>
      </header>

      <MobileNavigation page={page} navigate={navigate} />
      <div className='grid min-w-0 gap-8 lg:grid-cols-[13rem_minmax(0,1fr)]'>
        <Navigation page={page} navigate={navigate} />
        <div className='min-w-0' id='build-ai-content'>
          {page === 'overview' ? (
            <Overview
              character={character}
              draftBase={draftBase}
              draftRevision={draftRevision}
              draftSaved={draftSaved}
              educatorReviewComplete={effectiveEducatorReviewComplete}
              latestPublishedVersion={latestPublishedVersion}
              navigate={navigate}
              testState={effectiveTestState}
            />
          ) : null}
          {page === 'character' ? (
            <CharacterPage
              character={character}
              draftSaved={draftSaved}
              navigate={navigate}
              saveDraft={saveDraft}
              updateCharacter={updateCharacter}
            />
          ) : null}
          {page === 'curriculum' ? (
            <Curriculum
              navigate={navigate}
              previewState={previewState}
              scenario={scenario}
              setPreviewState={setPreviewState}
              setScenario={setScenario}
            />
          ) : null}
          {page === 'teaching' ? (
            <Teaching
              discardChanges={discardChanges}
              draftSaved={draftSaved}
              navigate={navigate}
              previewState={previewState}
              saveDraft={saveDraft}
              setPreviewState={setPreviewState}
              teachingSettings={teachingSettings}
              updateTeachingSetting={updateTeachingSetting}
            />
          ) : null}
          {page === 'test' ? (
            <TestTutor
              draftRevision={draftRevision}
              draftSaved={draftSaved}
              navigate={navigate}
              runTests={runTests}
              testState={testState}
            />
          ) : null}
          {page === 'publish' ? (
            <Publish
              completeEducatorReview={completeEducatorReview}
              currentDraftPublished={currentDraftPublished}
              draftRevision={draftRevision}
              draftSaved={draftSaved}
              educatorReviewComplete={effectiveEducatorReviewComplete}
              navigate={navigate}
              publishedVersions={publishedVersions}
              publishVersion={publishVersion}
              startDraft={startDraft}
              testState={effectiveTestState}
              versionNote={versionNote}
              setVersionNote={setVersionNote}
            />
          ) : null}
          {page === 'activity' ? (
            <Activity
              latestPublishedVersion={latestPublishedVersion}
              navigate={navigate}
            />
          ) : null}
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
      {destinations.map(({ icon, id, label }) => (
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
            <PandaiIcon
              aria-hidden='true'
              className='size-4 shrink-0'
              name={icon}
            />
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
  character,
  draftBase,
  draftRevision,
  draftSaved,
  educatorReviewComplete,
  latestPublishedVersion,
  navigate,
  testState,
}: {
  character: CharacterConfig
  draftBase: string
  draftRevision: number
  draftSaved: boolean
  educatorReviewComplete: boolean
  latestPublishedVersion: PublishedVersion
  navigate: (page: BuildPage) => void
  testState: TestState
}) {
  return (
    <div>
      <PageHeader page='overview' title='P&AI Tutor'>
        {draftSaved
          ? `Draft saved. Published version ${latestPublishedVersion.id} remains available to teachers, and ${latestPublishedVersion.authorizedClassCount} authorized classes use it. Test results are ${testState === 'passed' ? 'passed and current' : 'out of date'}.`
          : `The Draft has unsaved changes. Published version ${latestPublishedVersion.id} remains available to teachers, and ${latestPublishedVersion.authorizedClassCount} authorized classes still use it. Learners are not affected.`}
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
                    : pageSummary(id, draftSaved, testState)}
              </TextLink>
            </li>
          ))}
        </ol>
      </section>
      <AdminSurface className='shadow-none'>
        <h2 className='text-lg font-semibold'>Tutor state</h2>
        <dl className='mt-4 grid gap-5 border-t border-border pt-5 sm:grid-cols-2'>
          <StateItem term='Character'>
            {characterSummary(character)}.{' '}
            {draftSaved
              ? 'Saved with the private Draft.'
              : 'Unsaved character changes. Learners are not affected.'}
          </StateItem>
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
            {testLabel(testState)} for Draft revision D-{draftRevision}.
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
          Published version {latestPublishedVersion.id} is available
        </h2>
        <p className='mt-2 text-muted-foreground'>
          {latestPublishedVersion.authorizedClassCount} authorized classes use
          this immutable version. The private Draft is based on Published
          version {draftBase}.
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

function CharacterPage({
  character,
  draftSaved,
  navigate,
  saveDraft,
  updateCharacter,
}: {
  character: CharacterConfig
  draftSaved: boolean
  navigate: (page: BuildPage) => void
  saveDraft: () => void
  updateCharacter: (config: CharacterConfig, message: string) => void
}) {
  return (
    <div>
      <PageHeader page='character' title='P-Bot character'>
        Create one Tutor from P-Bot’s recognizable visor, movement, and Pandai
        color language. Choices stay private until publication.
      </PageHeader>
      <CharacterCreator config={character} onChange={updateCharacter} />
      <div className='mt-8 flex flex-wrap gap-3 border-t border-border pt-7'>
        <Button className='min-h-11' disabled={draftSaved} onClick={saveDraft}>
          <PandaiIcon aria-hidden='true' name='check' />
          Save Draft
        </Button>
        <Button
          variant='outline'
          disabled={!draftSaved}
          onClick={() => navigate('curriculum')}
        >
          Continue to Curriculum
        </Button>
      </div>
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
        <Dialog open={narrowPreviewOpen} onOpenChange={setNarrowPreviewOpen}>
          <DialogTrigger asChild>
            <Button className='min-h-11 xl:hidden' variant='outline'>
              Open full-screen preview
            </Button>
          </DialogTrigger>
          <DialogContent
            className='inset-0 top-0 left-0 h-dvh max-w-none translate-x-0 translate-y-0 content-start overflow-y-auto rounded-none p-4 xl:hidden'
            data-testid='narrow-screen-preview'
            showCloseButton={false}
          >
            <DialogHeader className='sr-only'>
              <DialogTitle>Curriculum preview</DialogTitle>
              <DialogDescription>
                Preview the current local Curriculum values.
              </DialogDescription>
            </DialogHeader>
            <DialogClose asChild>
              <Button className='mb-4 min-h-11' variant='ghost'>
                <PandaiIcon aria-hidden='true' name='arrow-left' />
                Back
              </Button>
            </DialogClose>
            <PreviewPanel
              owner='Curriculum'
              previewState={previewState}
              scenario={scenario}
              scenarioID={`${scenarioID}-narrow`}
              setPreviewState={setPreviewState}
              setScenario={setScenario}
            />
          </DialogContent>
        </Dialog>
      </div>
      <div className='mt-7 flex flex-wrap gap-3'>
        <div>
          <Button disabled variant='outline'>
            Change curriculum unavailable
          </Button>
          <p className='mt-2 max-w-sm text-sm text-muted-foreground'>
            This illustrative workspace does not connect to a curriculum-change
            contract.
          </p>
        </div>
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

function Teaching({
  discardChanges,
  draftSaved,
  navigate,
  previewState,
  saveDraft,
  setPreviewState,
  teachingSettings,
  updateTeachingSetting,
}: {
  discardChanges: () => void
  draftSaved: boolean
  navigate: (page: BuildPage) => void
  previewState: PreviewState
  saveDraft: () => void
  setPreviewState: (state: PreviewState) => void
  teachingSettings: TeachingSettings
  updateTeachingSetting: (
    setting: TeachingSetting,
    value: string,
    message: string,
  ) => void
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
            selectedValue={teachingSettings.start}
            onChange={(value) =>
              updateTeachingSetting(
                'start',
                value,
                'Tutor start preference changed.',
              )
            }
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
            selectedValue={teachingSettings.detail}
            onChange={(value) =>
              updateTeachingSetting('detail', value, 'Response detail changed.')
            }
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
                  checked={teachingSettings.language === value}
                  onChange={() =>
                    updateTeachingSetting(
                      'language',
                      value,
                      'Tutor reply language changed.',
                    )
                  }
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
            selectedValue={teachingSettings.tone}
            onChange={(value) =>
              updateTeachingSetting('tone', value, 'Tone changed.')
            }
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
        <Dialog open={narrowPreviewOpen} onOpenChange={setNarrowPreviewOpen}>
          <DialogTrigger asChild>
            <Button className='min-h-11 xl:hidden' variant='outline'>
              Open full-screen preview
            </Button>
          </DialogTrigger>
          <DialogContent
            className='inset-0 top-0 left-0 h-dvh max-w-none translate-x-0 translate-y-0 content-start overflow-y-auto rounded-none p-4 xl:hidden'
            data-testid='narrow-screen-preview'
            showCloseButton={false}
          >
            <DialogHeader className='sr-only'>
              <DialogTitle>Teaching preview</DialogTitle>
              <DialogDescription>
                Preview the current local Teaching values.
              </DialogDescription>
            </DialogHeader>
            <DialogClose asChild>
              <Button className='mb-4 min-h-11' variant='ghost'>
                <PandaiIcon aria-hidden='true' name='arrow-left' />
                Back
              </Button>
            </DialogClose>
            <PreviewPanel
              owner='Teaching'
              previewState={previewState}
              scenario='fractions'
              scenarioID='teaching-preview-narrow'
              setPreviewState={setPreviewState}
            />
          </DialogContent>
        </Dialog>
      </div>
      <div className='mt-7 flex flex-wrap gap-3'>
        <Button className='min-h-11' disabled={draftSaved} onClick={saveDraft}>
          <PandaiIcon aria-hidden='true' name='check' />
          Save Draft
        </Button>
        <Button
          variant='outline'
          disabled={draftSaved}
          onClick={discardChanges}
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
  selectedValue,
}: {
  legend: string
  name: string
  onChange: (value: string) => void
  options: ReadonlyArray<string>
  selectedValue: string
}) {
  return (
    <fieldset className='rounded-xl border border-border p-4'>
      <legend className='px-1 font-semibold'>{legend}</legend>
      <p className='mb-3 text-sm text-muted-foreground'>
        The Tutor may adapt when learner evidence calls for it.
      </p>
      {options.map((option) => (
        <label
          key={option}
          className='flex min-h-11 cursor-pointer items-start gap-3 py-2 text-sm'
        >
          <input
            className='mt-0.5 size-4'
            checked={selectedValue === option}
            name={name}
            onChange={() => onChange(option)}
            type='radio'
          />
          <span>{option}</span>
        </label>
      ))}
    </fieldset>
  )
}

function TestTutor({
  draftRevision,
  draftSaved,
  navigate,
  runTests,
  testState,
}: {
  draftRevision: number
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
        Exact saved Draft D-{draftRevision}, saved by Nabila at 10:42, using
        curriculum revision 2026.08. These synthetic results are reproducible
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
                  · Draft D-{draftRevision}
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
                Draft D-{draftRevision}.
              </p>
            </details>
            <Button
              className='mt-3'
              variant='outline'
              onClick={() => {
                if (!draftSaved && action === 'Rerun') {
                  navigate('teaching')
                  return
                }
                if (action === 'Edit Curriculum') navigate('curriculum')
                else if (action === 'Edit Teaching') navigate('teaching')
                else runTests()
              }}
            >
              {!draftSaved && action === 'Rerun'
                ? 'Open Teaching to save'
                : action}
            </Button>
          </li>
        ))}
      </ol>
    </div>
  )
}

function Publish({
  completeEducatorReview,
  currentDraftPublished,
  draftRevision,
  draftSaved,
  educatorReviewComplete,
  navigate,
  publishedVersions,
  publishVersion,
  setVersionNote,
  startDraft,
  testState,
  versionNote,
}: {
  completeEducatorReview: () => void
  currentDraftPublished: boolean
  draftRevision: number
  draftSaved: boolean
  educatorReviewComplete: boolean
  navigate: (page: BuildPage) => void
  publishedVersions: ReadonlyArray<PublishedVersion>
  publishVersion: () => void
  setVersionNote: (value: string) => void
  startDraft: () => void
  testState: TestState
  versionNote: string
}) {
  const noteID = useId()
  const currentPublishedVersion = publishedVersions.find(
    (version) => version.draftRevision === draftRevision,
  )
  const latestPublishedVersion =
    publishedVersions[0] ?? initialPublishedVersions[0]
  const ready = draftSaved && testState === 'passed'
  const rows = [
    [
      'Draft saved and current',
      draftSaved ? 'Ready' : 'Needs action',
      'Teaching',
    ],
    [
      'Character appearance saved with the Draft',
      draftSaved ? 'Ready' : 'Needs action',
      'Character',
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
    ready &&
    educatorReviewComplete &&
    versionNote.trim().length > 0 &&
    !currentDraftPublished
  return (
    <div>
      <PageHeader page='publish' title='Publish'>
        One authoritative gate for the exact saved Draft. Published versions are
        immutable, and educator review cannot waive required safeguards.
      </PageHeader>
      {currentPublishedVersion ? (
        <div
          className='mb-7 rounded-xl border border-border bg-muted/40 p-4'
          role='status'
        >
          <Status tone='positive'>Published</Status>
          <p className='mt-2 font-medium'>
            Published version {currentPublishedVersion.id} is available. No
            classes changed.
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
                  Synthetic evidence checked at 10:48 · exact Draft D-
                  {draftRevision}.
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
                {repair === 'Character' ||
                repair === 'Curriculum' ||
                repair === 'Teaching' ||
                repair === 'Test tutor' ? (
                  <button
                    className='min-h-10 text-sm font-medium underline underline-offset-4'
                    onClick={() =>
                      navigate(
                        repair === 'Character'
                          ? 'character'
                          : repair === 'Curriculum'
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
                  <button
                    className='min-h-10 text-sm font-medium underline underline-offset-4 disabled:cursor-not-allowed disabled:opacity-50'
                    disabled={!draftSaved || testState !== 'passed'}
                    onClick={completeEducatorReview}
                    type='button'
                  >
                    Complete educator review
                  </button>
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
          disabled={currentDraftPublished}
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
          {publishedVersions.map((version, index) => (
            <li key={version.id}>
              <p className='font-medium'>
                Published version {version.id} ·{' '}
                {index === 0 ? 'Latest available' : 'Earlier'}
              </p>
              <p className='mt-1 text-sm text-muted-foreground'>
                {version.note} · published by {version.publisher} · curriculum{' '}
                {version.curriculumRevision} ·{' '}
                {version.authorizedClassCount === 0
                  ? 'no authorized classes'
                  : `${version.authorizedClassCount} authorized classes`}
              </p>
              {version.id === '2' ? (
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
                        This replaces the existing private Draft. Published
                        versions remain immutable and available, and no classes
                        change.
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
              ) : null}
            </li>
          ))}
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
          Four authorized classes use immutable Published version 3. Published
          version {latestPublishedVersion.id} is latest and has{' '}
          {latestPublishedVersion.authorizedClassCount === 0
            ? 'not been applied to a class'
            : `been applied to ${latestPublishedVersion.authorizedClassCount} authorized classes`}
          .
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

function Activity({
  latestPublishedVersion,
  navigate,
}: {
  latestPublishedVersion: PublishedVersion
  navigate: (page: BuildPage) => void
}) {
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
          Published version {latestPublishedVersion.id} remains available to
          teachers and is used by {latestPublishedVersion.authorizedClassCount}{' '}
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
          <li>
            Published version {latestPublishedVersion.id} became available ·
            synthetic · recently
          </li>
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
  const icon: PandaiIconName =
    tone === 'positive'
      ? 'check-circle'
      : tone === 'warning'
        ? 'alert-triangle'
        : 'info'
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
      <PandaiIcon aria-hidden='true' className='size-3.5' name={icon} />
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
function pageSummary(
  page: BuildPage,
  draftSaved: boolean,
  testState: TestState,
) {
  if (page === 'character') return 'Configured'
  if (page === 'curriculum') return '2026.08'
  if (page === 'teaching') return draftSaved ? 'Saved' : 'Unsaved changes'
  if (page === 'test') return testLabel(testState)
  if (page === 'publish') return 'Review'
  return ''
}
function testLabel(state: TestState) {
  if (state === 'passed') return 'Passed and current'
  return 'Out of date'
}
