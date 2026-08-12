import { Option, Schema, flow } from 'effect'

import { isSafeRedirectPath } from './rbac'

/** Parses a safe in-app redirect from unknown router input. */
export const readNextPath = flow(
  Schema.decodeUnknownOption(Schema.String),
  Option.filter(isSafeRedirectPath),
  Option.getOrUndefined,
)
