export type CharacterSilhouette =
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
  readonly silhouette: CharacterSilhouette
  readonly turn: number
}

export const defaultCharacterConfig: CharacterConfig = {
  color: 'pandai',
  expression: 'attentive',
  eyeScale: 1,
  gazeX: 0,
  gazeY: 0,
  silhouette: 'blob',
  turn: 0,
}

const colorLabels = {
  forest: 'Pandai deep green',
  leaf: 'Pandai leaf',
  mint: 'Pandai mint',
  pandai: 'Pandai green',
} satisfies Record<CharacterColor, string>

const expressionLabels = {
  attentive: 'Attentive',
  joyful: 'Joyful',
  neutral: 'Neutral',
  thoughtful: 'Thoughtful',
} satisfies Record<CharacterExpression, string>

const silhouetteLabels = {
  bean: 'Bean',
  blob: 'Blob',
  capsule: 'Capsule',
  cloud: 'Cloud',
  egg: 'Egg',
  pebble: 'Pebble',
} satisfies Record<CharacterSilhouette, string>

export function characterSummary(config: CharacterConfig) {
  return `${silhouetteLabels[config.silhouette]} · ${colorLabels[config.color]} · ${expressionLabels[config.expression]}`
}
