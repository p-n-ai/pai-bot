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

const colorLabels: Record<CharacterColor, string> = {
  forest: 'Pandai deep green',
  leaf: 'Pandai leaf',
  mint: 'Pandai mint',
  pandai: 'Pandai green',
}

const expressionLabels: Record<CharacterExpression, string> = {
  attentive: 'Attentive',
  joyful: 'Joyful',
  neutral: 'Neutral',
  thoughtful: 'Thoughtful',
}

const shapeLabels: Record<CharacterShape, string> = {
  bean: 'Bean',
  blob: 'Blob',
  capsule: 'Capsule',
  cloud: 'Cloud',
  egg: 'Egg',
  pebble: 'Pebble',
}

export function characterSummary(config: CharacterConfig) {
  return `${shapeLabels[config.shape]} · ${colorLabels[config.color]} · ${expressionLabels[config.expression]}`
}
