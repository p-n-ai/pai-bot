import { Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const PublicServiceIDSchema = Schema.Literals(['application', 'ai'])
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

const matchesPublicStatusSnapshot = Schema.is(PublicStatusSnapshotSchema)

/** Parses a coherent public status response from the backend boundary. */
export function readPublicStatusSnapshot(
  value: unknown,
): PublicStatusSnapshot | null {
  if (!matchesPublicStatusSnapshot(value)) {
    return null
  }

  const componentIDs = value.components.map((component) => component.id)
  if (
    componentIDs.filter((id) => id === 'application').length !== 1 ||
    new Set(componentIDs).size !== componentIDs.length
  ) {
    return null
  }

  const hasUnavailableComponent = value.components.some(
    (component) => component.status === 'unavailable',
  )
  if (
    (value.status === 'ok' && hasUnavailableComponent) ||
    (value.status === 'degraded' && !hasUnavailableComponent)
  ) {
    return null
  }

  return value
}
