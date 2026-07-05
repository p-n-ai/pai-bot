const adminRoles = [
  'student',
  'teacher',
  'parent',
  'admin',
  'platform_admin',
] as const

export type AdminRole = (typeof adminRoles)[number]

export interface AuthUser {
  user_id: string
  tenant_id?: string
  tenant_slug?: string
  tenant_name?: string
  role: AdminRole
  name?: string
  email?: string
  can_manage_ai_settings?: boolean
}

export interface SchoolChoice {
  tenant_id: string
  tenant_slug?: string
  tenant_name: string
}

export interface AuthSession {
  expires_at: string
  user: AuthUser
  tenant_choices?: Array<SchoolChoice>
}

export function isAuthSession(value: unknown): value is AuthSession {
  if (!isRecord(value)) {
    return false
  }

  return typeof value.expires_at === 'string' && isAuthUser(value.user)
}

export function isSchoolChoices(value: unknown): value is Array<SchoolChoice> {
  return Array.isArray(value) && value.every(isSchoolChoice)
}

function isAuthUser(value: unknown): value is AuthUser {
  if (!isRecord(value)) {
    return false
  }

  return typeof value.user_id === 'string' && isAdminRole(value.role)
}

function isAdminRole(value: unknown): value is AdminRole {
  return adminRoles.some((role) => role === value)
}

function isSchoolChoice(value: unknown): value is SchoolChoice {
  if (!isRecord(value)) {
    return false
  }

  return (
    typeof value.tenant_id === 'string' && typeof value.tenant_name === 'string'
  )
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
