import { Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const WhatsAppStatusSchema = Schema.Struct({
  connected: Schema.Boolean,
  enabled: Schema.optionalKey(Schema.Boolean),
  qr: Schema.optionalKey(Schema.String),
  qr_image: Schema.optionalKey(Schema.String),
})

export interface WhatsAppStatus extends EffectSchema.Type<
  typeof WhatsAppStatusSchema
> {}

const matchesWhatsAppStatus = Schema.is(WhatsAppStatusSchema)

/** Returns whether an unknown response satisfies the WhatsApp status contract. */
export function isWhatsAppStatus(value: unknown): value is WhatsAppStatus {
  return matchesWhatsAppStatus(value)
}
