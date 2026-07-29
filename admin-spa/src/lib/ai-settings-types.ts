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

export const AISettingsViewSchema = Schema.Struct({
  defaultProvider: Schema.String,
  openrouterModel: Schema.String,
  openrouterKey: AISettingsKeyStatusSchema,
  flags: Schema.Record(Schema.String, Schema.Boolean),
})

export const AISettingsOverrideSchema = Schema.Struct({
  defaultProvider: Schema.NullOr(Schema.String),
  openrouterModel: Schema.NullOr(Schema.String),
  openrouterKey: AISettingsKeyStatusSchema,
  flags: Schema.Record(Schema.String, Schema.Boolean),
})

export const AIProviderReadinessSchema = Schema.Struct({
  name: Schema.String,
  supported: Schema.Boolean,
  configured: Schema.Boolean,
  registrable: Schema.Boolean,
  effective: Schema.Boolean,
  managedBy: Schema.String,
})

export const AISettingsHealthSchema = Schema.Struct({
  revision: Schema.Number,
  appliedRevision: Schema.Number,
  drift: Schema.Boolean,
  openrouterKey: Schema.Struct({
    stored: Schema.Boolean,
    readable: Schema.Boolean,
    version: Schema.String,
    algorithm: Schema.String,
    keyId: Schema.String,
    migrationNeeded: Schema.Boolean,
  }),
})

export const AISettingsSchema = Schema.Struct({
  defaultProvider: Schema.String,
  openrouterModel: Schema.String,
  openrouterKey: AISettingsKeyStatusSchema,
  flags: Schema.Record(Schema.String, Schema.Boolean),
  availableProviders: Schema.mutable(Schema.Array(Schema.String)),
  providers: Schema.mutable(Schema.Array(AIProviderReadinessSchema)),
  health: AISettingsHealthSchema,
  sources: AISettingsSourcesSchema,
  baseline: AISettingsViewSchema,
  override: AISettingsOverrideSchema,
  effective: AISettingsViewSchema,
  revision: Schema.Number,
  appliedRevision: Schema.Number,
  drift: Schema.Boolean,
})

export interface AISettings extends EffectSchema.Type<
  typeof AISettingsSchema
> {}

export interface UpdateAISettingsInput {
  defaultProvider?: string | null
  openrouterModel?: string | null
  openrouterApiKey?: string | null
  expectedRevision?: number
  // null deletes the DB override so the value falls back to env/default.
  flags?: Record<string, boolean | null>
}

const decodeAISettings = Schema.decodeUnknownOption(AISettingsSchema)

/** Decodes an unknown response into AI settings, discarding mismatch details. */
export function readAISettings(value: unknown): AISettings | null {
  return Option.getOrNull(decodeAISettings(value))
}
