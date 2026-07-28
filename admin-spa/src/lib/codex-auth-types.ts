/** State reported by the server-owned Codex device authorization process. */
export type CodexAuthState =
  | 'disconnected'
  | 'starting'
  | 'awaiting_authorization'
  | 'connected'
  | 'failed'

/** Safe status fields exposed by the Codex device authorization API. */
export interface CodexAuthStatus {
  state: CodexAuthState
  verificationUrl: string
  userCode: string
  message: string
}

function readCodexAuthState(value: unknown): CodexAuthState | null {
  switch (value) {
    case 'disconnected':
    case 'starting':
    case 'awaiting_authorization':
    case 'connected':
    case 'failed':
      return value
    default:
      return null
  }
}

function readOptionalString(
  value: object,
  property: 'verificationUrl' | 'userCode' | 'message',
): string | null {
  const field = Reflect.get(value, property)
  return field === undefined ? '' : typeof field === 'string' ? field : null
}

/** Parses a Codex status response and normalizes omitted display fields. */
export function readCodexAuthStatus(value: unknown): CodexAuthStatus | null {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return null
  }

  const state = readCodexAuthState(Reflect.get(value, 'state'))
  const verificationUrl = readOptionalString(value, 'verificationUrl')
  const userCode = readOptionalString(value, 'userCode')
  const message = readOptionalString(value, 'message')
  if (
    state === null ||
    verificationUrl === null ||
    userCode === null ||
    message === null
  ) {
    return null
  }

  return { state, verificationUrl, userCode, message }
}
