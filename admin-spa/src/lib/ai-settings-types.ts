import { Option, Schema } from 'effect'

export const AISettingsKeyStatusSchema = Schema.Struct({
  set: Schema.Boolean,
  last4: Schema.String,
})

export type AISettingsKeyStatus = typeof AISettingsKeyStatusSchema.Type

export const AISettingsSourcesSchema = Schema.Struct({
  defaultProvider: Schema.String,
  openrouterModel: Schema.String,
  openrouterKey: Schema.String,
  flags: Schema.Record(Schema.String, Schema.String),
})

export type AISettingsSources = typeof AISettingsSourcesSchema.Type

export const AISettingsSchema = Schema.Struct({
  defaultProvider: Schema.String,
  openrouterModel: Schema.String,
  openrouterKey: AISettingsKeyStatusSchema,
  flags: Schema.Record(Schema.String, Schema.Boolean),
  availableProviders: Schema.mutable(Schema.Array(Schema.String)),
  sources: AISettingsSourcesSchema,
})

export type AISettings = typeof AISettingsSchema.Type

export interface UpdateAISettingsInput {
  defaultProvider?: string
  openrouterModel?: string
  openrouterApiKey?: string
  // null deletes the DB override so the value falls back to env/default.
  flags?: Record<string, boolean | null>
}

const decodeAISettings = Schema.decodeUnknownOption(AISettingsSchema)

/** Decodes an unknown response into AI settings, discarding mismatch details. */
export function readAISettings(value: unknown): AISettings | null {
  return Option.getOrNull(decodeAISettings(value))
}
