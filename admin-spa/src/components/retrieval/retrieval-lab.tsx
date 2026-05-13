import { useNavigate } from '@tanstack/react-router'
import { ChevronDownIcon, SearchIcon } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'

import type {
  RetrievalSearchForm,
  RetrievalSearchHit,
  RetrievalSearchPayload,
} from '@/lib/retrieval-lab-types'
import { useAuth } from '@/auth-provider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  buildRetrievalSearchPayload,
  defaultRetrievalSearchForm,
  exampleRetrievalSearchForm,
  getRetrievalRunCount,
  isExpiredRetrievalSession,
  normalizeRetrievalError,
  searchRetrieval,
} from '@/lib/retrieval-lab'

type SearchStatus =
  | { kind: 'idle'; message: string }
  | { kind: 'running'; message: string }
  | { kind: 'ok'; message: string }
  | { kind: 'error'; message: string }

interface SearchResult {
  durations: Array<number>
  hits: Array<RetrievalSearchHit>
  payload: RetrievalSearchPayload
}

interface SearchMetrics {
  average: string
  hits: string
  last: string
  spread: string
}

const resultColumns = [
  ['Last', 'last'],
  ['Average', 'average'],
  ['Spread', 'spread'],
  ['Hits', 'hits'],
] as const
const emptyHits: Array<RetrievalSearchHit> = []
const searchSummaryHelp =
  'Number of benchmark runs completed for this query. Use repeats to compare latency and ranking stability.'
const statusLabels = {
  error: 'Error',
  idle: 'Idle',
  ok: 'Ready',
  running: 'Running',
} satisfies Record<SearchStatus['kind'], string>

// fallow-ignore-next-line complexity
export function RetrievalLab() {
  const [form, setForm] = useState<RetrievalSearchForm>(
    defaultRetrievalSearchForm,
  )
  const [status, setStatus] = useState<SearchStatus>({
    kind: 'idle',
    message: 'Idle.',
  })
  const [result, setResult] = useState<SearchResult | null>(null)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [hasRunSearch, setHasRunSearch] = useState(false)
  const navigate = useNavigate()
  const { setAnonymousSession } = useAuth()

  const metrics = useMemo(() => getMetrics(result), [result])

  const setField = useCallback(
    (field: keyof RetrievalSearchForm) =>
      (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        setForm((current) => ({
          ...current,
          [field]: event.target.value,
        }))
      },
    [],
  )
  const setIncludeInactive = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      setForm((current) => ({
        ...current,
        include_inactive: event.target.checked,
      }))
    },
    [],
  )
  const loadExample = useCallback(() => {
    setForm(exampleRetrievalSearchForm)
    setStatus({ kind: 'idle', message: 'Example loaded.' })
  }, [])
  const toggleAdvancedSettings = useCallback(() => {
    setShowAdvanced((current) => !current)
  }, [])

  const submit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      setHasRunSearch(true)
      runSearch({
        form,
        navigateToLogin: async () => {
          setAnonymousSession()
          await navigate({
            to: '/',
            search: {
              next: '/dashboard/retrieval-lab',
            },
          })
        },
        setResult,
        setStatus,
      }).catch((caught: unknown) => {
        setHasRunSearch(true)
        setStatus({
          kind: 'error',
          message: caught instanceof Error ? caught.message : 'Search failed',
        })
      })
    },
    [form, navigate, setAnonymousSession],
  )

  return (
    <div className='grid gap-6'>
      <form className='grid gap-4' onSubmit={submit}>
        <header className='mx-auto w-full max-w-3xl pt-6 text-center'>
          <h1 className='m-0 text-4xl leading-none font-semibold text-balance text-foreground md:text-5xl'>
            PaiBot Search
          </h1>
          <p className='mx-auto mt-2.5 mb-0 max-w-2xl text-sm leading-relaxed text-muted-foreground'>
            Search retrieval content across collections and sources.
          </p>
        </header>
        <div className='flex min-h-16 items-center gap-3 rounded-lg border border-border bg-card px-4'>
          <SearchIcon data-icon='inline-start' />
          <Input
            aria-label='Search retrieval content'
            className='border-0 shadow-none'
            onChange={setField('query')}
            placeholder='Search retrieval content…'
            value={form.query}
          />
        </div>
        <div className='flex flex-wrap items-center justify-center gap-3'>
          <Button disabled={status.kind === 'running' || !form.query.trim()}>
            {status.kind === 'running' ? 'Running…' : 'Search'}
          </Button>
          <Button
            aria-controls='retrieval-advanced-settings'
            aria-expanded={showAdvanced}
            onClick={toggleAdvancedSettings}
            size='sm'
            type='button'
            variant='ghost'
          >
            Additional settings
            <ChevronDownIcon
              aria-hidden='true'
              className='transition-transform data-[open=true]:rotate-180'
              data-open={showAdvanced}
            />
          </Button>
        </div>
        {showAdvanced ? (
          <div
            className='grid gap-4 rounded-lg border border-border bg-muted p-4'
            id='retrieval-advanced-settings'
          >
            <div className='grid grid-cols-[repeat(auto-fit,minmax(160px,1fr))] gap-3'>
              <FieldInput
                id='retrieval-limit'
                inputMode='numeric'
                label='Limit'
                onChange={setField('limit')}
                value={form.limit}
              />
              <FieldInput
                id='retrieval-repeats'
                inputMode='numeric'
                label='Repeat runs'
                onChange={setField('repeats')}
                value={form.repeats}
              />
              <FieldInput
                id='retrieval-collection-ids'
                label='Collection IDs'
                onChange={setField('collection_ids')}
                value={form.collection_ids}
              />
              <FieldInput
                id='retrieval-kinds'
                label='Kinds'
                onChange={setField('kinds')}
                value={form.kinds}
              />
              <FieldInput
                id='retrieval-source-types'
                label='Source types'
                onChange={setField('source_types')}
                value={form.source_types}
              />
              <label className='flex min-h-10 items-center gap-2.5'>
                <input
                  checked={form.include_inactive}
                  onChange={setIncludeInactive}
                  type='checkbox'
                />
                Include inactive records
              </label>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='retrieval-metadata'>Metadata JSON</Label>
              <textarea
                className='min-h-24 w-full resize-y rounded-lg border border-border bg-card px-3 py-2.5 text-foreground'
                id='retrieval-metadata'
                onChange={setField('metadata')}
                placeholder='{"form":"1","topic_id":"F1-02"}'
                value={form.metadata}
              />
            </div>
            <div className='flex flex-wrap items-center justify-between gap-3 text-xs text-muted-foreground'>
              <span>Status: {getStatusLabel(status.kind)}</span>
              <Button onClick={loadExample} type='button' variant='outline'>
                Load example
              </Button>
            </div>
          </div>
        ) : null}
      </form>
      {hasRunSearch ? (
        <>
          <SearchSummary metrics={metrics} status={status} />
          <SearchHits hits={result?.hits ?? emptyHits} />
          <RawOutput result={result} />
        </>
      ) : null}
    </div>
  )
}

function FieldInput({
  id,
  inputMode,
  label,
  onChange,
  value,
}: {
  id: string
  inputMode?: 'numeric'
  label: string
  onChange: (event: ChangeEvent<HTMLInputElement>) => void
  value: string
}) {
  return (
    <div className='grid gap-2'>
      <Label htmlFor={id}>{label}</Label>
      <Input id={id} inputMode={inputMode} onChange={onChange} value={value} />
    </div>
  )
}

function SearchSummary({
  metrics,
  status,
}: {
  metrics: SearchMetrics
  status: SearchStatus
}) {
  if (status.kind === 'ok') {
    return <SearchSuccessSummary metrics={metrics} status={status} />
  }

  return <SearchStatePanel metrics={metrics} status={status} />
}

function SearchSuccessSummary({
  metrics,
  status,
}: {
  metrics: SearchMetrics
  status: Extract<SearchStatus, { kind: 'ok' }>
}) {
  return (
    <div className='retrieval-summary-row'>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger
            aria-label='Search run summary'
            className='font-inherit cursor-help bg-transparent p-0 text-left text-muted-foreground underline decoration-dotted underline-offset-4'
          >
            {status.message}
          </TooltipTrigger>
          <TooltipContent>{searchSummaryHelp}</TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <p>{formatHitSummary(metrics)}</p>
    </div>
  )
}

function SearchStatePanel({
  metrics,
  status,
}: {
  metrics: SearchMetrics
  status: Exclude<SearchStatus, { kind: 'ok' }>
}) {
  return (
    <div className='state-panel' data-tone={status.kind}>
      <p className='eyebrow'>Status</p>
      <h2>{status.kind === 'error' ? 'Search failed' : status.message}</h2>
      {status.kind === 'error' ? <p>{status.message}</p> : null}
      <dl className='stat-grid'>
        {resultColumns.map(([label, metric]) => (
          <div className='stat-item' key={label}>
            <dt>{label}</dt>
            <dd>{metrics[metric]}</dd>
          </div>
        ))}
      </dl>
    </div>
  )
}

function formatHitSummary(metrics: SearchMetrics): string {
  const hits = metrics.hits
  const hitCount = Number(hits)
  const hitLabel = hitCount === 1 ? 'hit' : 'hits'

  return `${hits} ${hitLabel} in about ${metrics.average}`
}

function SearchHits({ hits }: { hits: Array<RetrievalSearchHit> }) {
  if (hits.length === 0) {
    return (
      <p className='m-0 rounded-lg border border-border bg-muted px-5 py-4 text-sm text-muted-foreground'>
        No hits yet.
      </p>
    )
  }

  return (
    <ol className='grid list-none gap-6 p-0'>
      {hits.map((hit, index) => (
        <li className='grid gap-1.5' key={hit.document.id}>
          <p className='muted-note'>
            {hit.document.collection_id || 'unscoped'} · {hit.document.kind} ·
            score {hit.score.toFixed(2)}
          </p>
          <h2>{hit.document.title}</h2>
          <p>{hit.excerpt}</p>
          <div className='flex flex-wrap gap-2 text-xs text-muted-foreground'>
            <span className='rounded-full border border-border px-3 py-1'>
              Rank {index + 1}
            </span>
            <span className='rounded-full border border-border px-3 py-1'>
              Terms {hit.matched_terms}/{hit.high_signal_terms}
            </span>
            <span className='rounded-full border border-border px-3 py-1'>
              {hit.document.id}
            </span>
          </div>
        </li>
      ))}
    </ol>
  )
}

function RawOutput({ result }: { result: SearchResult | null }) {
  const [showRawOutput, setShowRawOutput] = useState(false)
  const toggleRawOutput = useCallback(() => {
    setShowRawOutput((current) => !current)
  }, [])
  const output = result
    ? JSON.stringify(
        {
          durations: result.durations,
          hits: result.hits,
          payload: result.payload,
        },
        null,
        2,
      )
    : 'Run a query to inspect the raw payload and hits.'

  return (
    <section className='grid justify-items-start gap-3 border-t border-border pt-1'>
      <Button
        aria-controls='retrieval-raw-output'
        aria-expanded={showRawOutput}
        onClick={toggleRawOutput}
        size='sm'
        type='button'
        variant='ghost'
      >
        Raw output
      </Button>
      {showRawOutput ? (
        <pre
          className='min-h-30 w-full overflow-x-auto rounded-lg border border-border bg-card p-3.5 text-xs leading-relaxed text-foreground'
          id='retrieval-raw-output'
        >
          {output}
        </pre>
      ) : null}
    </section>
  )
}

function getStatusLabel(kind: SearchStatus['kind']): string {
  return statusLabels[kind]
}

// fallow-ignore-next-line complexity
async function runSearch({
  form,
  navigateToLogin,
  setResult,
  setStatus,
}: {
  form: RetrievalSearchForm
  navigateToLogin: () => Promise<void>
  setResult: (result: SearchResult | null) => void
  setStatus: (status: SearchStatus) => void
}) {
  let payload: RetrievalSearchPayload

  try {
    payload = buildRetrievalSearchPayload(form)
  } catch (caught: unknown) {
    setStatus({
      kind: 'error',
      message: caught instanceof Error ? caught.message : 'Invalid search',
    })
    return
  }

  if (!payload.query) {
    setStatus({ kind: 'error', message: 'Query required.' })
    return
  }

  setStatus({ kind: 'running', message: 'Running…' })

  const durations: Array<number> = []

  try {
    const hits = await runRepeatedSearch({
      durations,
      payload,
      repeats: getRetrievalRunCount(form.repeats),
    })

    setResult({ durations, hits, payload })
    setStatus({ kind: 'ok', message: `Done. ${durations.length} run(s).` })
  } catch (caught: unknown) {
    const message = caught instanceof Error ? caught.message : 'Search failed'

    if (isExpiredRetrievalSession(message)) {
      await navigateToLogin()
      return
    }

    setResult({ durations, hits: [], payload })
    setStatus({ kind: 'error', message: normalizeRetrievalError(message) })
  }
}

async function runRepeatedSearch({
  durations,
  payload,
  repeats,
}: {
  durations: Array<number>
  payload: RetrievalSearchPayload
  repeats: number
}): Promise<Array<RetrievalSearchHit>> {
  let hits: Array<RetrievalSearchHit> = emptyHits

  await Array.from({ length: repeats }).reduce<Promise<void>>(
    async (previousRun) => {
      await previousRun
      const startedAt = performance.now()
      hits = await searchRetrieval(payload)
      durations.push(performance.now() - startedAt)
    },
    Promise.resolve(),
  )

  return hits
}

// fallow-ignore-next-line complexity
function getMetrics(result: SearchResult | null): SearchMetrics {
  if (!result) {
    return {
      average: '0 ms',
      hits: '0',
      last: '0 ms',
      spread: '0 / 0',
    }
  }

  const total = result.durations.reduce((sum, value) => sum + value, 0)
  const average = result.durations.length ? total / result.durations.length : 0
  const min = result.durations.length ? Math.min(...result.durations) : 0
  const max = result.durations.length ? Math.max(...result.durations) : 0
  const last = result.durations.at(-1) ?? 0

  return {
    average: `${Math.round(average)} ms`,
    hits: String(result.hits.length),
    last: `${Math.round(last)} ms`,
    spread: `${Math.round(min)} / ${Math.round(max)}`,
  }
}
