import { Schema } from 'effect'

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

export type AuthUser = typeof AuthUserSchema.Type

export const SchoolChoiceSchema = Schema.Struct({
  tenant_id: Schema.String,
  tenant_slug: Schema.optionalKey(Schema.String),
  tenant_name: Schema.String,
})

export type SchoolChoice = typeof SchoolChoiceSchema.Type

const SchoolChoices = Schema.mutable(Schema.Array(SchoolChoiceSchema))

export const AuthSessionSchema = Schema.Struct({
  expires_at: Schema.String,
  user: AuthUserSchema,
  tenant_choices: Schema.optionalKey(SchoolChoices),
})

export type AuthSession = typeof AuthSessionSchema.Type

const matchesAuthSession = Schema.is(AuthSessionSchema)
const matchesSchoolChoices = Schema.is(SchoolChoices)

/** Returns whether an unknown response satisfies the complete auth session contract. */
export function isAuthSession(value: unknown): value is AuthSession {
  return matchesAuthSession(value)
}

/** Returns whether an unknown response is a valid mutable school-choice list. */
export function isSchoolChoices(value: unknown): value is Array<SchoolChoice> {
  return matchesSchoolChoices(value)
}
