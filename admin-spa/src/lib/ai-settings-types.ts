import { Option, Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const AISettingsKeyStatusSchema = Schema.Struct({
  set: Schema.Boolean,
  last4: Schema.String,
})

export interface AISettingsKeyStatus extends EffectSchema.Type<
  typeof AISettingsKeyStatusSchema
> {}

export const AISettingsSourcesSchema = Schema.Struct({
  defaultProvider: Schema.String,
  openrouterModel: Schema.String,
  openrouterKey: Schema.String,
  flags: Schema.Record(Schema.String, Schema.String),
})

export interface AISettingsSources extends EffectSchema.Type<
  typeof AISettingsSourcesSchema
> {}

export const AISettingsSchema = Schema.Struct({
  defaultProvider: Schema.String,
  openrouterModel: Schema.String,
  openrouterKey: AISettingsKeyStatusSchema,
  flags: Schema.Record(Schema.String, Schema.Boolean),
  availableProviders: Schema.mutable(Schema.Array(Schema.String)),
  sources: AISettingsSourcesSchema,
})

export interface AISettings extends EffectSchema.Type<
  typeof AISettingsSchema
> {}

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
