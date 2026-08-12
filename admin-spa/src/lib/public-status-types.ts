import { Option, Schema, flow } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const PublicServiceIDSchema = Schema.Literals([
  'application',
  'ai_provider',
])
export const PublicServiceStateSchema = Schema.Literals([
  'operational',
  'unavailable',
])

export const PublicStatusComponentSchema = Schema.Struct({
  id: PublicServiceIDSchema,
  status: PublicServiceStateSchema,
})

export interface PublicStatusComponent extends EffectSchema.Type<
  typeof PublicStatusComponentSchema
> {}

export const PublicStatusSnapshotSchema = Schema.Struct({
  status: Schema.Literals(['ok', 'degraded']),
  components: Schema.Array(PublicStatusComponentSchema),
})

export interface PublicStatusSnapshot extends EffectSchema.Type<
  typeof PublicStatusSnapshotSchema
> {}

function isCoherentPublicStatusSnapshot(value: PublicStatusSnapshot): boolean {
  const componentIDs = value.components.map((component) => component.id)
  if (
    componentIDs.filter((id) => id === 'application').length !== 1 ||
    componentIDs.filter((id) => id === 'ai_provider').length !== 1 ||
    new Set(componentIDs).size !== componentIDs.length
  ) {
    return false
  }

  const hasUnavailableComponent = value.components.some(
    (component) => component.status === 'unavailable',
  )
  if (
    (value.status === 'ok' && hasUnavailableComponent) ||
    (value.status === 'degraded' && !hasUnavailableComponent)
  ) {
    return false
  }

  return true
}

const decodePublicStatusSnapshot = Schema.decodeUnknownOption(
  PublicStatusSnapshotSchema,
)

/** Parses a coherent public status response from the backend boundary. */
export const readPublicStatusSnapshot = flow(
  decodePublicStatusSnapshot,
  Option.filter(isCoherentPublicStatusSnapshot),
  Option.getOrNull,
)
