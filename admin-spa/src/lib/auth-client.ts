import { isAuthSession, isSchoolChoices } from './auth-types'
import { isSafeRedirectPath } from './rbac-paths'
import type { AuthSession } from './auth-types'

class AuthContractError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'AuthContractError'
  }
}

export interface LoginInput {
  email: string
  password: string
  tenant_id?: string
}

export interface AcceptInviteInput {
  token: string
  name: string
  password: string
}

export interface TenantRequiredResult {
  kind: 'tenant_required'
  tenant_choices: NonNullable<AuthSession['tenant_choices']>
  message: string
}

interface TenantRequiredPayload {
  kind: 'tenant_required'
  error: 'tenant_required'
  tenant_choices: NonNullable<AuthSession['tenant_choices']>
}

export type LoginResult =
  | {
      kind: 'authenticated'
      session: AuthSession
    }
  | TenantRequiredResult

export async function readAuthSession(
  fetcher: typeof fetch = fetch,
): Promise<AuthSession | null> {
  const response = await fetcher('/api/auth/session', {
    credentials: 'include',
  })

  if (!response.ok) {
    return null
  }

  const payload: unknown = await response.json()

  if (!isAuthSession(payload)) {
    throw new AuthContractError('Invalid auth session response')
  }

  return payload
}

export async function loginWithPassword(
  input: LoginInput,
  fetcher: typeof fetch = fetch,
): Promise<LoginResult> {
  const response = await postAuthJSON('/api/auth/login', input, fetcher)
  const payload = await readLoginPayload(response)

  if (!response.ok) {
    return parseLoginFailure(payload, response.status)
  }

  if (!isAuthSession(payload)) {
    throw new AuthContractError('Invalid auth login response')
  }

  return {
    kind: 'authenticated',
    session: payload,
  }
}

export async function acceptInvite(
  input: AcceptInviteInput,
  fetcher: typeof fetch = fetch,
): Promise<AuthSession> {
  const response = await postAuthJSON(
    '/api/auth/invitations/accept',
    input,
    fetcher,
  )
  const payload = await readLoginPayload(response)

  if (!response.ok) {
    throw new Error(
      readErrorMessage(payload, `Invite activation failed: ${response.status}`),
    )
  }

  if (!isAuthSession(payload)) {
    throw new AuthContractError('Invalid invite activation response')
  }

  return payload
}

export async function logout(fetcher: typeof fetch = fetch): Promise<void> {
  const response = await fetcher('/api/auth/logout', {
    method: 'POST',
    credentials: 'include',
    cache: 'no-store',
  })

  if (!response.ok) {
    const payload = await readLoginPayload(response)

    throw new Error(
      readErrorMessage(payload, `Logout failed: ${response.status}`),
    )
  }
}

function postAuthJSON(
  path: string,
  input: unknown,
  fetcher: typeof fetch,
): Promise<Response> {
  return fetcher(path, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    cache: 'no-store',
    body: JSON.stringify(input),
  })
}

export function buildGoogleLoginURL(nextPath: string | undefined): string {
  const url = new URL('/api/auth/google/start', 'http://localhost')

  if (isSafeRedirectPath(nextPath)) {
    url.searchParams.set('next', nextPath)
  }

  return `${url.pathname}${url.search}`
}

async function readLoginPayload(response: Response): Promise<unknown> {
  const text = await response.text()

  if (!text.trim()) {
    return null
  }

  try {
    return JSON.parse(text) as unknown
  } catch {
    return text
  }
}

function parseLoginFailure(
  payload: unknown,
  status: number,
): TenantRequiredResult {
  if (isTenantRequiredPayload(payload)) {
    return {
      kind: 'tenant_required',
      tenant_choices: payload.tenant_choices,
      message: payload.error,
    }
  }

  throw new Error(readErrorMessage(payload, `Login failed: ${status}`))
}

function isTenantRequiredPayload(
  payload: unknown,
): payload is TenantRequiredPayload {
  return isRecord(payload) && hasTenantRequiredShape(payload)
}

function readErrorMessage(payload: unknown, fallback: string): string {
  if (isInternalAuthFailurePayload(payload)) {
    return 'Sign-in service is taking too long. Try again.'
  }

  return readStringPayload(payload) ?? readPayloadError(payload) ?? fallback
}

function isInternalAuthFailurePayload(payload: unknown): boolean {
  if (typeof payload !== 'string') {
    return false
  }

  const message = payload.toLowerCase()

  return (
    message.includes('context deadline exceeded') ||
    message.includes('insert session') ||
    message.includes('failed to create session')
  )
}

function hasTenantRequiredShape(payload: Record<string, unknown>): boolean {
  return (
    payload.kind === 'tenant_required' &&
    payload.error === 'tenant_required' &&
    isSchoolChoices(payload.tenant_choices)
  )
}

function readStringPayload(payload: unknown): string | undefined {
  return typeof payload === 'string' ? payload : undefined
}

function readPayloadError(payload: unknown): string | undefined {
  return isRecord(payload) && typeof payload.error === 'string'
    ? payload.error
    : undefined
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
