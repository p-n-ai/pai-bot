import { Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

const ManagedUserRoleSchema = Schema.Literals([
  'teacher',
  'parent',
  'admin',
  'platform_admin',
])
const InviteDeliveryStatusSchema = Schema.Literals([
  'pending',
  'sent',
  'failed',
])
const OptionalNullableString = Schema.optionalKey(Schema.NullOr(Schema.String))

export type ManagedUserRole = typeof ManagedUserRoleSchema.Type
export type InviteIssueRole = 'teacher' | 'parent' | 'admin'
export type InviteDeliveryStatus = typeof InviteDeliveryStatusSchema.Type

export const InviteRecordSchema = Schema.Struct({
  activation_url: OptionalNullableString,
  delivery_error: OptionalNullableString,
  delivery_status: Schema.optionalKey(InviteDeliveryStatusSchema),
  email: Schema.String,
  expires_at: OptionalNullableString,
  id: OptionalNullableString,
  invite_token: Schema.String,
  invited_by_user_id: OptionalNullableString,
  role: ManagedUserRoleSchema,
})

export interface InviteRecord extends EffectSchema.Type<
  typeof InviteRecordSchema
> {}

export const UserManagementSummarySchema = Schema.Struct({
  parents: Schema.Number,
  pending_invites: Schema.Number,
  students: Schema.Number,
  teachers: Schema.Number,
  total_users: Schema.Number,
})

export interface UserManagementSummary extends EffectSchema.Type<
  typeof UserManagementSummarySchema
> {}

export const ManagedUserSchema = Schema.Struct({
  created_at: Schema.String,
  email: Schema.String,
  id: Schema.String,
  name: Schema.String,
  role: ManagedUserRoleSchema,
  status: Schema.Literal('active'),
  tenant_name: OptionalNullableString,
})

export interface ManagedUser extends EffectSchema.Type<
  typeof ManagedUserSchema
> {}

export const PendingInviteSchema = Schema.Struct({
  created_at: Schema.String,
  delivery_error: OptionalNullableString,
  delivery_sent_at: Schema.optionalKey(Schema.NullOr(Schema.String)),
  delivery_status: Schema.optionalKey(InviteDeliveryStatusSchema),
  email: Schema.String,
  expires_at: Schema.String,
  id: Schema.String,
  invited_by: Schema.String,
  role: ManagedUserRoleSchema,
  status: Schema.Literal('pending'),
  tenant_name: OptionalNullableString,
})

export interface PendingInvite extends EffectSchema.Type<
  typeof PendingInviteSchema
> {}

export const ManagedStudentSchema = Schema.Struct({
  channel: Schema.String,
  created_at: Schema.String,
  external_id: Schema.String,
  form: Schema.String,
  id: Schema.String,
  name: Schema.String,
})

export interface ManagedStudent extends EffectSchema.Type<
  typeof ManagedStudentSchema
> {}

export const UserManagementViewSchema = Schema.Struct({
  active_users: Schema.mutable(Schema.Array(ManagedUserSchema)),
  pending_invites: Schema.mutable(Schema.Array(PendingInviteSchema)),
  students: Schema.mutable(Schema.Array(ManagedStudentSchema)),
  summary: UserManagementSummarySchema,
})

export interface UserManagementView extends EffectSchema.Type<
  typeof UserManagementViewSchema
> {}

/** Returns whether an unknown response satisfies the invite contract. */
export const isInviteRecord = Schema.is(InviteRecordSchema)

/** Returns whether an unknown response satisfies the user-management view contract. */
export const isUserManagementView = Schema.is(UserManagementViewSchema)
