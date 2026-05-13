import { isNumber, isRecord, isString } from './type-guards'

const managedUserRoles = new Set([
  'teacher',
  'parent',
  'admin',
  'platform_admin',
])
const inviteDeliveryStatuses = new Set(['pending', 'sent', 'failed'])

export type ManagedUserRole = 'teacher' | 'parent' | 'admin' | 'platform_admin'
export type InviteIssueRole = 'teacher' | 'parent' | 'admin'
export type InviteDeliveryStatus = 'pending' | 'sent' | 'failed'

export interface InviteRecord {
  activation_url?: string
  delivery_error?: string
  delivery_status?: InviteDeliveryStatus
  email: string
  expires_at?: string
  id?: string
  invite_token: string
  invited_by_user_id?: string
  role: ManagedUserRole
}

export interface UserManagementSummary {
  parents: number
  pending_invites: number
  students: number
  teachers: number
  total_users: number
}

export interface ManagedUser {
  created_at: string
  email: string
  id: string
  name: string
  role: ManagedUserRole
  status: 'active'
  tenant_name?: string
}

export interface PendingInvite {
  created_at: string
  delivery_error?: string
  delivery_sent_at?: string | null
  delivery_status?: InviteDeliveryStatus
  email: string
  expires_at: string
  id: string
  invited_by: string
  role: ManagedUserRole
  status: 'pending'
  tenant_name?: string
}

export interface UserManagementView {
  active_users: Array<ManagedUser>
  pending_invites: Array<PendingInvite>
  students: Array<ManagedStudent>
  summary: UserManagementSummary
}

export interface ManagedStudent {
  channel: string
  created_at: string
  external_id: string
  form: string
  id: string
  name: string
}

export function isInviteRecord(value: unknown): value is InviteRecord {
  if (!isRecord(value)) {
    return false
  }

  return [
    isInviteEmail,
    hasInviteRole,
    hasInviteToken,
    hasOptionalInviteID,
    hasOptionalActivationURL,
    hasOptionalExpiry,
    hasOptionalInviterID,
    hasOptionalDeliveryError,
    hasOptionalDeliveryStatus,
  ].every((check) => check(value))
}

export function isUserManagementView(
  value: unknown,
): value is UserManagementView {
  if (!isRecord(value)) {
    return false
  }

  return [
    hasUserManagementSummary,
    hasActiveUsers,
    hasManagedStudents,
    hasPendingInvites,
  ].every((check) => check(value))
}

function isUserManagementSummary(
  value: unknown,
): value is UserManagementSummary {
  if (!isRecord(value)) {
    return false
  }

  return [
    hasParentCount,
    hasPendingInviteCount,
    hasStudentCount,
    hasTeacherCount,
    hasTotalUserCount,
  ].every((check) => check(value))
}

function isManagedUser(value: unknown): value is ManagedUser {
  if (!isRecord(value)) {
    return false
  }

  return [
    hasManagedUserID,
    hasManagedUserName,
    hasManagedUserEmail,
    hasManagedUserRole,
    hasActiveStatus,
    hasCreatedAt,
    hasOptionalTenantName,
  ].every((check) => check(value))
}

function isPendingInvite(value: unknown): value is PendingInvite {
  if (!isRecord(value)) {
    return false
  }

  return [
    hasPendingInviteID,
    hasPendingInviteEmail,
    hasPendingInviteRole,
    hasPendingStatus,
    hasPendingExpiry,
    hasPendingCreatedAt,
    hasPendingInviter,
    hasOptionalPendingDeliveryError,
    hasOptionalDeliverySentAt,
    hasOptionalPendingTenantName,
    hasOptionalPendingDeliveryStatus,
  ].every((check) => check(value))
}

function isManagedStudent(value: unknown): value is ManagedStudent {
  if (!isRecord(value)) {
    return false
  }

  return [
    hasStudentID,
    hasStudentName,
    hasStudentExternalID,
    hasStudentChannel,
    hasStudentForm,
    hasStudentCreatedAt,
  ].every((check) => check(value))
}

function isInviteEmail(value: Record<string, unknown>): boolean {
  return isString(value.email)
}

function hasInviteRole(value: Record<string, unknown>): boolean {
  return isManagedUserRole(value.role)
}

function hasInviteToken(value: Record<string, unknown>): boolean {
  return isString(value.invite_token)
}

function hasOptionalInviteID(value: Record<string, unknown>): boolean {
  return optionalString(value.id)
}

function hasOptionalActivationURL(value: Record<string, unknown>): boolean {
  return optionalString(value.activation_url)
}

function hasOptionalExpiry(value: Record<string, unknown>): boolean {
  return optionalString(value.expires_at)
}

function hasOptionalInviterID(value: Record<string, unknown>): boolean {
  return optionalString(value.invited_by_user_id)
}

function hasOptionalDeliveryError(value: Record<string, unknown>): boolean {
  return optionalString(value.delivery_error)
}

function hasOptionalDeliveryStatus(value: Record<string, unknown>): boolean {
  return optionalDeliveryStatus(value.delivery_status)
}

function hasUserManagementSummary(value: Record<string, unknown>): boolean {
  return isUserManagementSummary(value.summary)
}

function hasActiveUsers(value: Record<string, unknown>): boolean {
  return (
    Array.isArray(value.active_users) && value.active_users.every(isManagedUser)
  )
}

function hasPendingInvites(value: Record<string, unknown>): boolean {
  return (
    Array.isArray(value.pending_invites) &&
    value.pending_invites.every(isPendingInvite)
  )
}

function hasManagedStudents(value: Record<string, unknown>): boolean {
  return Array.isArray(value.students) && value.students.every(isManagedStudent)
}

function hasParentCount(value: Record<string, unknown>): boolean {
  return isNumber(value.parents)
}

function hasPendingInviteCount(value: Record<string, unknown>): boolean {
  return isNumber(value.pending_invites)
}

function hasTeacherCount(value: Record<string, unknown>): boolean {
  return isNumber(value.teachers)
}

function hasStudentCount(value: Record<string, unknown>): boolean {
  return isNumber(value.students)
}

function hasTotalUserCount(value: Record<string, unknown>): boolean {
  return isNumber(value.total_users)
}

function hasManagedUserID(value: Record<string, unknown>): boolean {
  return isString(value.id)
}

function hasManagedUserName(value: Record<string, unknown>): boolean {
  return isString(value.name)
}

function hasManagedUserEmail(value: Record<string, unknown>): boolean {
  return isString(value.email)
}

function hasManagedUserRole(value: Record<string, unknown>): boolean {
  return isManagedUserRole(value.role)
}

function hasActiveStatus(value: Record<string, unknown>): boolean {
  return value.status === 'active'
}

function hasCreatedAt(value: Record<string, unknown>): boolean {
  return isString(value.created_at)
}

function hasOptionalTenantName(value: Record<string, unknown>): boolean {
  return optionalString(value.tenant_name)
}

function hasStudentID(value: Record<string, unknown>): boolean {
  return isString(value.id)
}

function hasStudentName(value: Record<string, unknown>): boolean {
  return isString(value.name)
}

function hasStudentExternalID(value: Record<string, unknown>): boolean {
  return isString(value.external_id)
}

function hasStudentChannel(value: Record<string, unknown>): boolean {
  return isString(value.channel)
}

function hasStudentForm(value: Record<string, unknown>): boolean {
  return isString(value.form)
}

function hasStudentCreatedAt(value: Record<string, unknown>): boolean {
  return isString(value.created_at)
}

function hasPendingInviteID(value: Record<string, unknown>): boolean {
  return isString(value.id)
}

function hasPendingInviteEmail(value: Record<string, unknown>): boolean {
  return isString(value.email)
}

function hasPendingInviteRole(value: Record<string, unknown>): boolean {
  return isManagedUserRole(value.role)
}

function hasPendingStatus(value: Record<string, unknown>): boolean {
  return value.status === 'pending'
}

function hasPendingExpiry(value: Record<string, unknown>): boolean {
  return isString(value.expires_at)
}

function hasPendingCreatedAt(value: Record<string, unknown>): boolean {
  return isString(value.created_at)
}

function hasPendingInviter(value: Record<string, unknown>): boolean {
  return isString(value.invited_by)
}

function hasOptionalPendingDeliveryError(
  value: Record<string, unknown>,
): boolean {
  return optionalString(value.delivery_error)
}

function hasOptionalDeliverySentAt(value: Record<string, unknown>): boolean {
  return optionalString(value.delivery_sent_at)
}

function hasOptionalPendingTenantName(value: Record<string, unknown>): boolean {
  return optionalString(value.tenant_name)
}

function hasOptionalPendingDeliveryStatus(
  value: Record<string, unknown>,
): boolean {
  return optionalDeliveryStatus(value.delivery_status)
}

function isManagedUserRole(value: unknown): value is ManagedUserRole {
  return isString(value) && managedUserRoles.has(value)
}

function optionalDeliveryStatus(value: unknown): boolean {
  return value === undefined || isDeliveryStatus(value)
}

function optionalString(value: unknown): boolean {
  return value === undefined || value === null || isString(value)
}

function isDeliveryStatus(value: unknown): value is InviteDeliveryStatus {
  return isString(value) && inviteDeliveryStatuses.has(value)
}
