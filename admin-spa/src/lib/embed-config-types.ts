import { Option, Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

const StringArray = Schema.mutable(Schema.Array(Schema.String))
const ThemeConfig = Schema.Record(Schema.String, Schema.Unknown)

const EmbedConfigWire = Schema.Struct({
  id: Schema.optionalKey(Schema.Unknown),
  ID: Schema.optionalKey(Schema.Unknown),
  tenant_id: Schema.optionalKey(Schema.Unknown),
  TenantID: Schema.optionalKey(Schema.Unknown),
  enabled: Schema.optionalKey(Schema.Unknown),
  Enabled: Schema.optionalKey(Schema.Unknown),
  public_embed_base_url: Schema.optionalKey(Schema.Unknown),
  PublicEmbedBaseURL: Schema.optionalKey(Schema.Unknown),
  allowed_origins: Schema.optionalKey(Schema.Unknown),
  AllowedOrigins: Schema.optionalKey(Schema.Unknown),
  theme_config: Schema.optionalKey(Schema.Unknown),
  ThemeConfig: Schema.optionalKey(Schema.Unknown),
  created_at: Schema.optionalKey(Schema.Unknown),
  CreatedAt: Schema.optionalKey(Schema.Unknown),
  updated_at: Schema.optionalKey(Schema.Unknown),
  UpdatedAt: Schema.optionalKey(Schema.Unknown),
})

export const EmbedConfigSchema = Schema.Struct({
  id: Schema.String,
  tenant_id: Schema.String,
  enabled: Schema.Boolean,
  public_embed_base_url: Schema.String,
  allowed_origins: StringArray,
  theme_config: ThemeConfig,
  created_at: Schema.optionalKey(Schema.String),
  updated_at: Schema.optionalKey(Schema.String),
})

export interface EmbedConfig extends EffectSchema.Type<
  typeof EmbedConfigSchema
> {}

export interface UpdateEmbedConfigInput {
  enabled?: boolean
  theme_config?: Record<string, unknown>
}

const decodeEmbedConfigWire = Schema.decodeUnknownOption(EmbedConfigWire)
const decodeBoolean = Schema.decodeUnknownOption(Schema.Boolean)
const decodeString = Schema.decodeUnknownOption(Schema.String)
const decodeStringArray = Schema.decodeUnknownOption(StringArray)
const decodeThemeConfig = Schema.decodeUnknownOption(ThemeConfig)

/** Decodes and normalizes snake-case or Go-style embed configuration fields. */
export function readEmbedConfig(value: unknown): EmbedConfig | null {
  const wire = Option.getOrNull(decodeEmbedConfigWire(value))
  if (wire === null) {
    return null
  }

  const enabled = Option.getOrNull(decodeBoolean(wire.enabled ?? wire.Enabled))
  const tenantID = Option.getOrNull(
    decodeString(wire.tenant_id ?? wire.TenantID),
  )
  const publicEmbedBaseURL = Option.getOrNull(
    decodeString(wire.public_embed_base_url ?? wire.PublicEmbedBaseURL),
  )

  if (enabled === null || tenantID === null || publicEmbedBaseURL === null) {
    return null
  }

  return {
    id: Option.getOrElse(decodeString(wire.id ?? wire.ID), () => ''),
    tenant_id: tenantID,
    enabled,
    public_embed_base_url: publicEmbedBaseURL,
    allowed_origins: Option.getOrElse(
      decodeStringArray(wire.allowed_origins ?? wire.AllowedOrigins),
      () => [],
    ),
    theme_config: Option.getOrElse(
      decodeThemeConfig(wire.theme_config ?? wire.ThemeConfig),
      () => ({}),
    ),
    created_at: Option.getOrUndefined(
      decodeString(wire.created_at ?? wire.CreatedAt),
    ),
    updated_at: Option.getOrUndefined(
      decodeString(wire.updated_at ?? wire.UpdatedAt),
    ),
  }
}
