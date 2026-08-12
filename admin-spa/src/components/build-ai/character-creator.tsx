/* oxlint-disable react-perf/jsx-no-new-array-as-prop, react-perf/jsx-no-new-function-as-prop, react-perf/jsx-no-new-object-as-prop -- This bounded local editor keeps each visible control transition beside its state update; extracted children are not memoized. */
import { useEffect, useId, useRef, useState } from 'react'

import { AdminSurface } from '@/components/shared/admin-surface'
import { Button } from '@/components/ui/button'
import { PandaiIcon } from '@/components/ui/pandai-icon'
import { Slider } from '@/components/ui/slider'
import { cn } from '@/lib/utils'

export type CharacterShape =
  | 'blob'
  | 'pebble'
  | 'bean'
  | 'egg'
  | 'capsule'
  | 'cloud'
export type CharacterColor = 'pandai' | 'mint' | 'leaf' | 'forest'
export type CharacterExpression =
  | 'neutral'
  | 'joyful'
  | 'thoughtful'
  | 'attentive'

export interface CharacterConfig {
  readonly color: CharacterColor
  readonly expression: CharacterExpression
  readonly eyeScale: number
  readonly gazeX: number
  readonly gazeY: number
  readonly shape: CharacterShape
  readonly turn: number
}

export const defaultCharacterConfig: CharacterConfig = {
  color: 'pandai',
  expression: 'attentive',
  eyeScale: 1,
  gazeX: 0,
  gazeY: 0,
  shape: 'blob',
  turn: 0,
}

const bodyPath =
  'M228.541 114.228C228.541 130.133 225.184 145.994 218.738 160.534C212.674 174.217 203.904 186.669 193.065 196.988C155.933 232.34 99.497 238.596 55.5255 212.24C45.097 205.99 35.6851 198.072 27.7451 188.866C19.1926 178.953 12.3686 167.569 7.65781 155.351C2.60712 142.264 0 128.257 0 114.228C0 98.3219 3.35751 82.4611 9.80315 67.9215C15.8672 54.2382 24.6377 41.7862 35.4767 31.4668C72.6081 -3.88483 129.044 -10.1413 173.016 16.2153C183.444 22.4653 192.856 30.3829 200.796 39.5896C209.349 49.5018 216.173 60.8859 220.883 73.1037C225.934 86.1906 228.541 100.198 228.541 114.228Z'

const shapes: ReadonlyArray<{
  id: CharacterShape
  label: string
  offsetX: number
  offsetY: number
  scaleX: number
  scaleY: number
}> = [
  { id: 'blob', label: 'Blob', offsetX: 0, offsetY: 0, scaleX: 1, scaleY: 1 },
  {
    id: 'pebble',
    label: 'Pebble',
    offsetX: 2,
    offsetY: 0,
    scaleX: 0.97,
    scaleY: 0.86,
  },
  {
    id: 'bean',
    label: 'Bean',
    offsetX: 0,
    offsetY: 0,
    scaleX: 0.72,
    scaleY: 0.9,
  },
  {
    id: 'egg',
    label: 'Egg',
    offsetX: 0,
    offsetY: 2,
    scaleX: 0.77,
    scaleY: 0.97,
  },
  {
    id: 'capsule',
    label: 'Capsule',
    offsetX: 0,
    offsetY: 0,
    scaleX: 0.64,
    scaleY: 0.92,
  },
  {
    id: 'cloud',
    label: 'Cloud',
    offsetX: 0,
    offsetY: 4,
    scaleX: 0.79,
    scaleY: 0.7,
  },
]

const colors: ReadonlyArray<{
  id: CharacterColor
  label: string
  value: string
}> = [
  {
    id: 'pandai',
    label: 'Pandai green',
    value: 'var(--surface-primary-default)',
  },
  {
    id: 'mint',
    label: 'Pandai mint',
    value: 'var(--surface-primary-default-subtle-hover)',
  },
  {
    id: 'leaf',
    label: 'Pandai leaf',
    value: 'var(--surface-secondary-default)',
  },
  {
    id: 'forest',
    label: 'Pandai deep green',
    value: 'var(--surface-primary-focus)',
  },
]

const expressions: ReadonlyArray<{
  id: CharacterExpression
  label: string
  sourceIndex: number
  left: string
  right: string
}> = [
  {
    id: 'neutral',
    label: 'Neutral',
    sourceIndex: 0,
    left: 'M130.36 45.98L132.71 46.19L134.98 46.81L137.11 47.83L138.97 49.28L140.47 51.09L141.68 53.12L142.73 55.23L143.76 57.36L144.78 59.49L145.79 61.62L146.79 63.76L147.76 65.91L148.71 68.07L149.63 70.25L150.52 72.43L151.37 74.63L151.99 76.91L152.1 79.26L151.64 81.57L150.59 83.68L149.04 85.45L147.1 86.78L144.9 87.62L142.56 87.93L140.22 87.71L137.98 86.99L135.93 85.82L134.17 84.24L132.78 82.34L131.69 80.25L130.77 78.08L129.87 75.89L128.94 73.72L128 71.56L127.03 69.4L126.05 67.26L125.05 65.12L124.03 62.99L122.93 60.9L121.87 58.79L121.03 56.59L120.72 54.26L121.1 51.93L122.15 49.83L123.75 48.1L125.76 46.89L128.01 46.19Z',
    right:
      'M176.61 37.08L178.72 37.59L180.7 38.48L182.52 39.65L184.2 41.03L185.71 42.59L187.03 44.31L188.2 46.14L189.26 48.03L190.27 49.96L191.26 51.89L192.23 53.84L193.16 55.8L194.05 57.78L194.92 59.77L195.74 61.78L196.53 63.8L197.27 65.84L197.97 67.9L198.47 70.01L198.63 72.18L198.4 74.33L197.58 76.33L195.95 77.72L193.83 78.08L191.71 77.65L189.76 76.69L188.03 75.38L186.53 73.82L185.28 72.05L184.25 70.13L183.4 68.14L182.63 66.11L181.87 64.07L181.07 62.05L180.25 60.04L179.39 58.05L178.49 56.07L177.57 54.1L176.61 52.15L175.62 50.22L174.59 48.31L173.53 46.41L172.54 44.48L171.86 42.42L171.76 40.26L172.62 38.3L174.45 37.19Z',
  },
  {
    id: 'joyful',
    label: 'Joyful',
    sourceIndex: 2,
    left: 'M104.84 104.08L109.39 104.89L113.63 106.73L117.32 109.5L120.25 113.08L122.22 117.25L123.15 121.77L122.92 126.39L121.86 130.89L120.61 135.35L119.37 139.81L118.15 144.27L116.94 148.74L115.74 153.21L114.56 157.69L113.39 162.17L112.24 166.65L111.13 171.15L110.04 175.65L108.89 180.13L106.98 184.33L103.8 187.67L99.75 189.87L95.25 190.93L90.63 190.93L86.11 189.97L81.86 188.16L78.05 185.54L74.89 182.18L72.62 178.16L71.54 173.67L71.94 169.08L73.01 164.57L74.13 160.08L75.26 155.59L76.42 151.11L77.59 146.63L78.77 142.15L79.97 137.68L81.18 133.22L82.41 128.75L83.65 124.29L84.87 119.83L86.33 115.43L88.67 111.46L91.9 108.15L95.83 105.72L100.23 104.35Z',
    right:
      'M174.26 115.55L178.59 116.54L182.37 118.86L185.31 122.2L187.23 126.21L188.12 130.57L188.06 135.02L187.52 139.44L186.86 143.85L186.11 148.25L185.25 152.62L184.3 156.98L183.24 161.31L182.07 165.61L180.79 169.88L179.39 174.12L177.89 178.31L176.27 182.47L174.47 186.55L172.17 190.35L169.21 193.68L165.73 196.46L161.84 198.64L157.66 200.16L153.27 200.91L148.83 200.69L144.65 199.19L141.47 196.14L140.18 191.91L140.82 187.53L142.46 183.38L144.16 179.26L145.73 175.09L147.18 170.87L148.51 166.62L149.73 162.33L150.84 158.01L151.84 153.67L152.74 149.3L153.54 144.91L154.24 140.51L154.85 136.1L155.58 131.7L156.93 127.46L159.15 123.6L162.12 120.29L165.73 117.68L169.84 116Z',
  },
  {
    id: 'thoughtful',
    label: 'Thoughtful',
    sourceIndex: 8,
    left: 'M64.12 83.43L66.69 84.08L68.9 85.53L70.57 87.6L71.56 90.05L71.86 92.69L71.53 95.32L70.55 97.78L69.06 99.99L67.36 102.03L65.64 104.06L63.95 106.11L62.28 108.18L60.63 110.27L59 112.38L57.41 114.5L55.84 116.65L54.29 118.82L52.78 121L51.31 123.22L49.86 125.45L48.42 127.69L46.67 129.68L44.36 130.96L41.73 131.22L39.22 130.38L37.16 128.72L35.64 126.55L34.65 124.08L34.2 121.46L34.29 118.81L34.94 116.24L36.12 113.86L37.54 111.61L39.04 109.41L40.54 107.21L42.06 105.03L43.61 102.87L45.18 100.73L46.79 98.6L48.41 96.5L50.07 94.42L51.74 92.35L53.45 90.31L55.17 88.28L56.94 86.3L59.04 84.67L61.48 83.65Z',
    right:
      'M104.23 93.09L106.97 93.77L109.4 95.24L111.32 97.31L112.59 99.85L113.4 102.57L114.13 105.32L114.87 108.06L115.62 110.8L116.37 113.54L117.14 116.28L117.9 119.02L118.67 121.75L119.45 124.49L120.23 127.22L121.03 129.95L121.83 132.68L122.6 135.41L122.98 138.22L122.57 141.02L121.34 143.57L119.44 145.67L117.05 147.19L114.34 148.04L111.51 148.14L108.77 147.45L106.33 146.01L104.38 143.96L103.07 141.44L102.21 138.73L101.43 136L100.65 133.27L99.87 130.53L99.09 127.8L98.32 125.06L97.55 122.33L96.79 119.59L96.04 116.85L95.28 114.11L94.53 111.37L93.79 108.62L93.05 105.88L92.73 103.07L93.18 100.27L94.42 97.72L96.32 95.62L98.7 94.08L101.4 93.21Z',
  },
  {
    id: 'attentive',
    label: 'Attentive',
    sourceIndex: 10,
    left: 'M103.39 78.25L106.56 78.38L109.59 79.28L112.29 80.94L114.43 83.28L115.86 86.1L116.52 89.2L116.32 92.37L115.62 95.46L114.86 98.55L114.1 101.63L113.36 104.72L112.63 107.81L111.91 110.91L111.2 114.01L110.51 117.1L109.82 120.21L109.16 123.31L108.52 126.42L107.84 129.53L106.8 132.52L105.04 135.16L102.66 137.25L99.81 138.63L96.7 139.2L93.54 138.96L90.53 137.95L87.87 136.24L85.71 133.92L84.21 131.13L83.48 128.05L83.5 124.88L84 121.74L84.7 118.64L85.4 115.54L86.08 112.44L86.77 109.34L87.48 106.24L88.19 103.15L88.92 100.05L89.66 96.96L90.41 93.88L91.18 90.8L91.95 87.71L93.1 84.76L94.95 82.19L97.4 80.19L100.28 78.86Z',
    right:
      'M161.99 91.57L165.07 92L167.9 93.28L170.32 95.23L172.22 97.69L173.49 100.53L174.05 103.59L173.93 106.7L173.45 109.78L172.9 112.85L172.33 115.91L171.71 118.97L171.06 122.02L170.38 125.06L169.66 128.09L168.91 131.12L168.13 134.14L167.33 137.15L166.5 140.16L165.58 143.14L164.33 145.98L162.5 148.5L160.15 150.53L157.38 151.95L154.34 152.61L151.24 152.43L148.32 151.38L145.83 149.52L143.99 147.01L142.95 144.08L142.73 140.98L143.25 137.91L144.09 134.91L144.94 131.91L145.76 128.9L146.54 125.89L147.29 122.86L148.01 119.83L148.69 116.79L149.34 113.74L149.95 110.68L150.54 107.62L151.09 104.55L151.6 101.47L152.44 98.48L154 95.79L156.21 93.6L158.94 92.12Z',
  },
]

const previewStates: ReadonlyArray<{
  id: string
  label: string
  expression: CharacterExpression | 'resting'
}> = [
  { id: 'idle', label: 'Idle', expression: 'resting' },
  { id: 'listening', label: 'Listening', expression: 'attentive' },
  { id: 'thinking', label: 'Thinking', expression: 'thoughtful' },
  { id: 'celebrating', label: 'Celebrating', expression: 'joyful' },
]

export function CharacterCreator({
  config,
  onChange,
}: {
  config: CharacterConfig
  onChange: (config: CharacterConfig, message: string) => void
}) {
  const [previewState, setPreviewState] = useState('idle')
  const [blinking, setBlinking] = useState(false)
  const blinkTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(
    () => () => {
      if (blinkTimer.current !== null) clearTimeout(blinkTimer.current)
    },
    [],
  )

  const update = (next: Partial<CharacterConfig>, message: string) => {
    onChange({ ...config, ...next }, message)
  }
  const blink = () => {
    if (blinkTimer.current !== null) clearTimeout(blinkTimer.current)
    setBlinking(true)
    blinkTimer.current = setTimeout(() => setBlinking(false), 320)
  }

  return (
    <div className='grid gap-8 xl:grid-cols-[minmax(20rem,0.9fr)_minmax(0,1.1fr)]'>
      <div className='xl:sticky xl:top-6 xl:self-start'>
        <AdminSurface
          className='overflow-hidden shadow-none'
          contentClassName='p-0'
        >
          <div className='flex items-center justify-between gap-4 border-b border-[var(--admin-line)] px-5 py-4'>
            <div className='flex min-w-0 items-center gap-3'>
              <img
                alt=''
                className='size-12 shrink-0'
                height='48'
                src='/illustrations/pbot.svg'
                width='48'
              />
              <div className='min-w-0'>
                <p className='font-semibold'>P-Bot character preview</p>
                <p className='mt-1 truncate text-xs text-muted-foreground'>
                  {shapeLabel(config.shape)} ·{' '}
                  {expressionLabel(config.expression)}
                </p>
              </div>
            </div>
            <span className='shrink-0 rounded-full border border-border px-2.5 py-1 text-xs font-medium'>
              P-Bot · {previewState}
            </span>
          </div>
          <div className='grid min-h-[22rem] place-items-center bg-[var(--admin-surface-muted)] p-8 sm:min-h-[28rem]'>
            <CharacterPreview
              blinking={blinking}
              config={config}
              previewState={previewState}
            />
          </div>
          <div className='border-t border-[var(--admin-line)] px-5 py-4'>
            <p className='text-xs font-medium tracking-wide text-muted-foreground uppercase'>
              Preview states
            </p>
            <div
              className='mt-3 flex flex-wrap gap-2'
              role='group'
              aria-label='Character preview state'
            >
              {previewStates.map((state) => (
                <Button
                  aria-pressed={previewState === state.id}
                  key={state.id}
                  onClick={() => setPreviewState(state.id)}
                  size='sm'
                  variant={previewState === state.id ? 'default' : 'outline'}
                >
                  {state.label}
                </Button>
              ))}
              <Button onClick={blink} size='sm' variant='outline'>
                <PandaiIcon aria-hidden='true' name='activity' />
                Blink
              </Button>
            </div>
          </div>
        </AdminSurface>
      </div>

      <div className='space-y-8'>
        <fieldset>
          <legend className='text-lg font-semibold'>P-Bot silhouette</legend>
          <p className='mt-1 text-sm leading-6 text-muted-foreground'>
            Shape one P-Bot family character for every learner-facing chat.
          </p>
          <div className='mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3'>
            {shapes.map((shape) => (
              <button
                aria-pressed={config.shape === shape.id}
                className={cn(
                  'min-h-28 rounded-xl border border-border p-3 text-left outline-offset-2 transition-[border-color,background-color,box-shadow] hover:border-foreground/40 hover:bg-muted/50 focus-visible:outline-2 motion-reduce:transition-none',
                  config.shape === shape.id &&
                    'border-foreground bg-muted ring-1 ring-foreground/15',
                )}
                key={shape.id}
                onClick={() =>
                  update({ shape: shape.id }, 'Character silhouette changed.')
                }
                type='button'
              >
                <MiniShape color={colorValue(config.color)} shape={shape} />
                <span className='mt-2 block text-sm font-medium'>
                  {shape.label}
                </span>
              </button>
            ))}
          </div>
        </fieldset>

        <fieldset className='border-t border-border pt-7'>
          <legend className='text-lg font-semibold'>Green tone</legend>
          <p className='mt-1 text-sm leading-6 text-muted-foreground'>
            Every option stays inside Pandai’s green family with P-Bot’s fixed
            dark visor.
          </p>
          <div className='mt-4 grid gap-2 sm:grid-cols-2'>
            {colors.map((color) => (
              <button
                aria-pressed={config.color === color.id}
                className={cn(
                  'flex min-h-11 items-center gap-3 rounded-lg border border-border px-3 py-2 text-left text-sm font-medium outline-offset-2 hover:bg-muted focus-visible:outline-2',
                  config.color === color.id && 'border-foreground bg-muted',
                )}
                key={color.id}
                onClick={() =>
                  update({ color: color.id }, 'Character palette changed.')
                }
                type='button'
              >
                <span
                  aria-hidden='true'
                  className='size-5 rounded-full border border-black/15'
                  style={{ backgroundColor: color.value }}
                />
                {color.label}
              </button>
            ))}
          </div>
        </fieldset>

        <fieldset className='border-t border-border pt-7'>
          <legend className='text-lg font-semibold'>Resting expression</legend>
          <p className='mt-1 text-sm leading-6 text-muted-foreground'>
            Set the face learners see before the Tutor reacts to a conversation.
          </p>
          <div className='mt-4 grid grid-cols-2 gap-2 sm:grid-cols-4'>
            {expressions.map((expression) => (
              <button
                aria-pressed={config.expression === expression.id}
                className={cn(
                  'min-h-11 rounded-lg border border-border px-3 py-2 text-sm font-medium outline-offset-2 hover:bg-muted focus-visible:outline-2',
                  config.expression === expression.id &&
                    'border-foreground bg-muted',
                )}
                key={expression.id}
                onClick={() => {
                  setPreviewState('idle')
                  update(
                    { expression: expression.id },
                    'Resting expression changed.',
                  )
                }}
                type='button'
              >
                {expression.label}
              </button>
            ))}
          </div>
        </fieldset>

        <section
          className='border-t border-border pt-7'
          aria-labelledby='motion-heading'
        >
          <div className='flex items-start justify-between gap-4'>
            <div>
              <h2 id='motion-heading' className='text-lg font-semibold'>
                Motion tuning
              </h2>
              <p className='mt-1 text-sm leading-6 text-muted-foreground'>
                Tune small movements. Conversation state still controls live
                reactions.
              </p>
            </div>
            <Button
              aria-label='Reset motion tuning'
              onClick={() =>
                update(
                  { eyeScale: 1, gazeX: 0, gazeY: 0, turn: 0 },
                  'Character motion tuning reset.',
                )
              }
              size='icon-sm'
              title='Reset motion tuning'
              variant='ghost'
            >
              <PandaiIcon aria-hidden='true' name='rotate-ccw' />
            </Button>
          </div>
          <div className='mt-5 space-y-6'>
            <CharacterSlider
              label='Horizontal gaze'
              max={0.6}
              min={-0.6}
              onChange={(value) =>
                update({ gazeX: value }, 'Horizontal gaze changed.')
              }
              output={config.gazeX.toFixed(2)}
              step={0.05}
              value={config.gazeX}
            />
            <CharacterSlider
              label='Vertical gaze'
              max={0.6}
              min={-0.6}
              onChange={(value) =>
                update({ gazeY: value }, 'Vertical gaze changed.')
              }
              output={config.gazeY.toFixed(2)}
              step={0.05}
              value={config.gazeY}
            />
            <CharacterSlider
              label='Head turn'
              max={18}
              min={-18}
              onChange={(value) =>
                update({ turn: value }, 'Head turn changed.')
              }
              output={`${config.turn}°`}
              step={1}
              value={config.turn}
            />
            <CharacterSlider
              label='Eye scale'
              max={1.35}
              min={0.75}
              onChange={(value) =>
                update({ eyeScale: value }, 'Eye scale changed.')
              }
              output={`${config.eyeScale.toFixed(2)}×`}
              step={0.05}
              value={config.eyeScale}
            />
          </div>
        </section>

        <div className='flex gap-3 border-t border-border pt-7 text-sm text-muted-foreground'>
          <PandaiIcon
            aria-hidden='true'
            className='mt-0.5 size-4 shrink-0'
            name='star'
          />
          <p>
            Every option keeps P-Bot’s visor, orbit nodes, and Pandai color
            language. Settings stay private until publication.
          </p>
        </div>
      </div>
    </div>
  )
}

function CharacterPreview({
  blinking,
  config,
  previewState,
}: {
  blinking: boolean
  config: CharacterConfig
  previewState: string
}) {
  const clipID = `character-head-${useId().replaceAll(':', '')}`
  const shape =
    shapes.find((candidate) => candidate.id === config.shape) ?? shapes[0]
  const state = previewStates.find((candidate) => candidate.id === previewState)
  const expressionID =
    state?.expression === 'resting' ? config.expression : state?.expression
  const expression =
    expressions.find((candidate) => candidate.id === expressionID) ??
    expressions[0]
  const shapeTransform = `translate(${shape.offsetX} ${shape.offsetY}) translate(114.27 114.27) scale(${shape.scaleX} ${shape.scaleY}) translate(-114.27 -114.27)`
  const eyeTransform = `translate(${(config.gazeX * 13.2).toFixed(2)} ${(config.gazeY * 8.4).toFixed(2)}) translate(114.27 114.27) scale(${config.eyeScale} ${blinking ? 0.04 : config.eyeScale}) translate(-114.27 -114.27)`

  return (
    <svg
      aria-labelledby={`${clipID}-title ${clipID}-description`}
      className='size-full max-h-[24rem] max-w-[24rem] overflow-visible'
      data-color={config.color}
      data-expression={expression.id}
      data-shape={config.shape}
      role='img'
      viewBox='-22 -22 273 273'
    >
      <title id={`${clipID}-title`}>
        P&amp;AI Tutor character preview · P-Bot
      </title>
      <desc id={`${clipID}-description`}>
        {shape.label} P-Bot character with the {expression.label.toLowerCase()}{' '}
        expression.
      </desc>
      <defs>
        <clipPath id={clipID}>
          <path d={bodyPath} />
        </clipPath>
        <clipPath id={`${clipID}-visor`}>
          <rect height='174' rx='52' width='174' x='27' y='27' />
        </clipPath>
      </defs>
      <g
        className='transition-transform duration-300 ease-out motion-reduce:transition-none'
        style={{ transformBox: 'fill-box', transformOrigin: 'center' }}
        transform={`rotate(${config.turn} 114.27 114.27) ${shapeTransform}`}
      >
        <circle
          cx='13'
          cy='67'
          fill={colorValue(config.color)}
          r='11'
          stroke='var(--text-default-heading)'
          strokeWidth='1.5'
        />
        <circle
          cx='216'
          cy='171'
          fill={colorValue(config.color)}
          r='11'
          stroke='var(--text-default-heading)'
          strokeWidth='1.5'
        />
        <path
          d={bodyPath}
          fill={colorValue(config.color)}
          stroke='var(--text-default-heading)'
          strokeWidth='1.5'
        />
        <g clipPath={`url(#${clipID})`}>
          <rect
            fill='var(--text-default-heading)'
            height='174'
            rx='52'
            width='174'
            x='27'
            y='27'
          />
          <g clipPath={`url(#${clipID}-visor)`}>
            <g
              className='transition-transform duration-150 ease-out motion-reduce:transition-none'
              style={{ transformBox: 'fill-box', transformOrigin: 'center' }}
              transform={eyeTransform}
            >
              <path
                d={expression.left}
                fill='var(--surface-primary-default-subtle-hover)'
              />
              <path
                d={expression.right}
                fill='var(--surface-primary-default-subtle-hover)'
              />
            </g>
          </g>
        </g>
      </g>
    </svg>
  )
}

function MiniShape({
  color,
  shape,
}: {
  color: string
  shape: (typeof shapes)[number]
}) {
  const clipID = `mini-pbot-${useId().replaceAll(':', '')}`
  const transform = `translate(${shape.offsetX} ${shape.offsetY}) translate(114.27 114.27) scale(${shape.scaleX} ${shape.scaleY}) translate(-114.27 -114.27)`
  return (
    <svg aria-hidden='true' className='h-14 w-full' viewBox='-15 -15 259 259'>
      <defs>
        <clipPath id={clipID}>
          <path d={bodyPath} />
        </clipPath>
      </defs>
      <g transform={transform}>
        <circle
          cx='13'
          cy='67'
          fill={color}
          r='11'
          stroke='var(--text-default-heading)'
          strokeWidth='2'
        />
        <circle
          cx='216'
          cy='171'
          fill={color}
          r='11'
          stroke='var(--text-default-heading)'
          strokeWidth='2'
        />
        <path
          d={bodyPath}
          fill={color}
          stroke='var(--text-default-heading)'
          strokeWidth='2'
        />
        <g clipPath={`url(#${clipID})`}>
          <rect
            fill='var(--text-default-heading)'
            height='174'
            rx='52'
            width='174'
            x='27'
            y='27'
          />
          <path
            d='M61 108C68 90 82 90 89 108M139 108C146 90 160 90 167 108'
            fill='none'
            stroke='var(--surface-primary-default-subtle-hover)'
            strokeLinecap='round'
            strokeWidth='9'
          />
        </g>
      </g>
    </svg>
  )
}

function CharacterSlider({
  label,
  max,
  min,
  onChange,
  output,
  step,
  value,
}: {
  label: string
  max: number
  min: number
  onChange: (value: number) => void
  output: string
  step: number
  value: number
}) {
  return (
    <div>
      <div className='mb-3 flex items-center justify-between gap-4 text-sm'>
        <span className='font-medium'>{label}</span>
        <output className='font-mono text-xs text-muted-foreground tabular-nums'>
          {output}
        </output>
      </div>
      <Slider
        aria-label={label}
        max={max}
        min={min}
        onValueChange={(values) => {
          onChange(values[0])
        }}
        step={step}
        value={[value]}
      />
    </div>
  )
}

export function characterSummary(config: CharacterConfig) {
  return `${shapeLabel(config.shape)} · ${colorLabel(config.color)} · ${expressionLabel(config.expression)}`
}

function colorLabel(color: CharacterColor) {
  return (
    colors.find((candidate) => candidate.id === color)?.label ?? 'Pandai green'
  )
}

function colorValue(color: CharacterColor) {
  return (
    colors.find((candidate) => candidate.id === color)?.value ??
    'var(--surface-primary-default)'
  )
}

function expressionLabel(expression: CharacterExpression) {
  return (
    expressions.find((candidate) => candidate.id === expression)?.label ??
    'Attentive'
  )
}

function shapeLabel(shape: CharacterShape) {
  return shapes.find((candidate) => candidate.id === shape)?.label ?? 'Blob'
}
