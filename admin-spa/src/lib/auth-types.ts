import { Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

const AdminRoleSchema = Schema.Literals([
  'student',
  'teacher',
  'parent',
  'admin',
  'platform_admin',
])

export type AdminRole = typeof AdminRoleSchema.Type

export const AuthUserSchema = Schema.Struct({
  user_id: Schema.String,
  tenant_id: Schema.optionalKey(Schema.String),
  tenant_slug: Schema.optionalKey(Schema.String),
  tenant_name: Schema.optionalKey(Schema.String),
  role: AdminRoleSchema,
  name: Schema.optionalKey(Schema.String),
  email: Schema.optionalKey(Schema.String),
  can_manage_ai_settings: Schema.optionalKey(Schema.Boolean),
})

export interface AuthUser extends EffectSchema.Type<typeof AuthUserSchema> {}

export const SchoolChoiceSchema = Schema.Struct({
  tenant_id: Schema.String,
  tenant_slug: Schema.optionalKey(Schema.String),
  tenant_name: Schema.String,
})

export interface SchoolChoice extends EffectSchema.Type<
  typeof SchoolChoiceSchema
> {}

const SchoolChoices = Schema.mutable(Schema.Array(SchoolChoiceSchema))

export const AuthSessionSchema = Schema.Struct({
  expires_at: Schema.String,
  user: AuthUserSchema,
  tenant_choices: Schema.optionalKey(SchoolChoices),
})

export interface AuthSession extends EffectSchema.Type<
  typeof AuthSessionSchema
> {}

/** Returns whether an unknown response satisfies the complete auth session contract. */
export const isAuthSession = Schema.is(AuthSessionSchema)

/** Returns whether an unknown response is a valid mutable school-choice list. */
export const isSchoolChoices = Schema.is(SchoolChoices)
