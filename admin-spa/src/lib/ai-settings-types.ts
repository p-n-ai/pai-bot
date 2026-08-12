import { Option, Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const APIKeyProviderNameSchema = Schema.String

export type APIKeyProviderName = EffectSchema.Type<
  typeof APIKeyProviderNameSchema
>

const SettingsSourceSchema = Schema.Literals(['db', 'env', 'default', 'none'])

export const APIKeyProviderSelectorSchema = Schema.Struct({
  type: Schema.Literal('api_key'),
  name: APIKeyProviderNameSchema,
})

export const OllamaProviderSelectorSchema = Schema.Struct({
  type: Schema.Literal('ollama'),
})

export const ManagedCodexProviderSelectorSchema = Schema.Struct({
  type: Schema.Literal('managed_codex'),
})

export const ProviderSelectorSchema = Schema.Union([
  APIKeyProviderSelectorSchema,
  OllamaProviderSelectorSchema,
  ManagedCodexProviderSelectorSchema,
])

export type ProviderSelector = EffectSchema.Type<typeof ProviderSelectorSchema>

export const AISettingsKeyStatusSchema = Schema.Struct({
  set: Schema.Boolean,
  last4: Schema.String,
})

export type AISettingsKeyStatus = EffectSchema.Type<
  typeof AISettingsKeyStatusSchema
>

const StringProjectionSchema = Schema.Struct({
  baseline: Schema.NullOr(Schema.String),
  override: Schema.NullOr(Schema.String),
  effective: Schema.NullOr(Schema.String),
  source: SettingsSourceSchema,
})

const BooleanProjectionSchema = Schema.Struct({
  baseline: Schema.Boolean,
  override: Schema.NullOr(Schema.Boolean),
  effective: Schema.Boolean,
  source: SettingsSourceSchema,
})

const ProviderReadinessSchema = Schema.Struct({
  supported: Schema.Boolean,
  configured: Schema.Boolean,
  registrable: Schema.Boolean,
  effective: Schema.Boolean,
  managedBy: Schema.String,
})

const CredentialHealthSchema = Schema.Struct({
  stored: Schema.Boolean,
  readable: Schema.Boolean,
  version: Schema.String,
  algorithm: Schema.String,
  keyId: Schema.String,
  migrationNeeded: Schema.Boolean,
})

const CredentialProjectionSchema = Schema.Struct({
  baseline: AISettingsKeyStatusSchema,
  override: AISettingsKeyStatusSchema,
  effective: AISettingsKeyStatusSchema,
  source: SettingsSourceSchema,
  health: CredentialHealthSchema,
})

export const APIKeyProviderProjectionSchema = Schema.Struct({
  type: Schema.Literal('api_key'),
  name: APIKeyProviderNameSchema,
  displayName: Schema.String,
  model: StringProjectionSchema,
  credential: CredentialProjectionSchema,
  readiness: ProviderReadinessSchema,
})

export const OllamaProviderProjectionSchema = Schema.Struct({
  type: Schema.Literal('ollama'),
  enabled: BooleanProjectionSchema,
  model: StringProjectionSchema,
  readiness: ProviderReadinessSchema,
})

export const ManagedCodexProviderProjectionSchema = Schema.Struct({
  type: Schema.Literal('managed_codex'),
  enabled: BooleanProjectionSchema,
  model: StringProjectionSchema,
  readiness: ProviderReadinessSchema,
})

export const ProviderProjectionSchema = Schema.Union([
  APIKeyProviderProjectionSchema,
  OllamaProviderProjectionSchema,
  ManagedCodexProviderProjectionSchema,
])

export type ProviderProjection = EffectSchema.Type<
  typeof ProviderProjectionSchema
>

const DefaultProviderProjectionSchema = Schema.Struct({
  baseline: Schema.NullOr(ProviderSelectorSchema),
  override: Schema.NullOr(ProviderSelectorSchema),
  effective: Schema.NullOr(ProviderSelectorSchema),
  source: SettingsSourceSchema,
})

const FlagsProjectionSchema = Schema.Struct({
  baseline: Schema.Record(Schema.String, Schema.Boolean),
  override: Schema.Record(Schema.String, Schema.Boolean),
  effective: Schema.Record(Schema.String, Schema.Boolean),
  sources: Schema.Record(Schema.String, SettingsSourceSchema),
})

export const AISettingsSchema = Schema.Struct({
  defaultProvider: DefaultProviderProjectionSchema,
  providers: Schema.mutable(Schema.Array(ProviderProjectionSchema)),
  flags: FlagsProjectionSchema,
  revision: Schema.Number,
  appliedRevision: Schema.Number,
  drift: Schema.Boolean,
})

export type AISettings = EffectSchema.Type<typeof AISettingsSchema>

export type APIKeyProviderPatch = {
  type: 'api_key'
  name: APIKeyProviderName
  model?: string | null
  apiKey?: string | null
}

export type OllamaProviderPatch = {
  type: 'ollama'
  enabled?: boolean | null
  model?: string | null
}

export type ManagedCodexProviderPatch = {
  type: 'managed_codex'
  model?: string | null
}

export type ProviderPatch =
  | APIKeyProviderPatch
  | OllamaProviderPatch
  | ManagedCodexProviderPatch

export interface UpdateAISettingsInput {
  defaultProvider?: ProviderSelector | null
  provider?: ProviderPatch
  expectedRevision: number
  // null deletes the DB override so the value falls back to env/default.
  flags?: Record<string, boolean | null>
}

const decodeAISettings = Schema.decodeUnknownOption(AISettingsSchema, {
  onExcessProperty: 'error',
})

/** Decodes an unknown response into the exact redacted AI settings contract. */
export function readAISettings(value: unknown): AISettings | null {
  const settings = Option.getOrNull(decodeAISettings(value))
  if (!settings) return null
  const apiKeyNames = new Set<string>()
  for (const provider of settings.providers) {
    if (provider.type === 'api_key') {
      if (
        !provider.name.trim() ||
        !provider.displayName.trim() ||
        apiKeyNames.has(provider.name)
      ) {
        return null
      }
      apiKeyNames.add(provider.name)
    }
  }
  for (const selector of [
    settings.defaultProvider.baseline,
    settings.defaultProvider.override,
    settings.defaultProvider.effective,
  ]) {
    if (selector?.type === 'api_key' && !apiKeyNames.has(selector.name)) {
      return null
    }
  }
  return settings
}
