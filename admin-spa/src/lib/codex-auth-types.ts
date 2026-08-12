import { Option, Schema, flow } from 'effect'

/** State reported by the server-owned Codex device authorization process. */
const CodexAuthStateSchema = Schema.Literals([
  'disconnected',
  'starting',
  'awaiting_authorization',
  'connected',
  'failed',
])

export type CodexAuthState = typeof CodexAuthStateSchema.Type

/** Safe status fields exposed by the Codex device authorization API. */
export interface CodexAuthStatus {
  state: CodexAuthState
  verificationUrl: string
  userCode: string
  message: string
}

const CodexAuthStatusWireSchema = Schema.Struct({
  state: CodexAuthStateSchema,
  verificationUrl: Schema.optionalKey(Schema.String),
  userCode: Schema.optionalKey(Schema.String),
  message: Schema.optionalKey(Schema.String),
})

const decodeCodexAuthStatus = Schema.decodeUnknownOption(
  CodexAuthStatusWireSchema,
)

function normalizeCodexAuthStatus(
  status: typeof CodexAuthStatusWireSchema.Type,
): CodexAuthStatus {
  return {
    state: status.state,
    verificationUrl: status.verificationUrl ?? '',
    userCode: status.userCode ?? '',
    message: status.message ?? '',
  }
}

/** Parses a Codex status response and normalizes omitted display fields. */
export const readCodexAuthStatus = flow(
  decodeCodexAuthStatus,
  Option.map(normalizeCodexAuthStatus),
  Option.getOrNull,
)
