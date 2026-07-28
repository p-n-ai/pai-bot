import { Option, Schema } from 'effect'

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

/** Parses a Codex status response and normalizes omitted display fields. */
export function readCodexAuthStatus(value: unknown): CodexAuthStatus | null {
  return Option.match(decodeCodexAuthStatus(value), {
    onNone: () => null,
    onSome: (status) => ({
      state: status.state,
      verificationUrl: status.verificationUrl ?? '',
      userCode: status.userCode ?? '',
      message: status.message ?? '',
    }),
  })
}
