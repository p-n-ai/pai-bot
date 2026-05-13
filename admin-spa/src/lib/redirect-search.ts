import { isSafeRedirectPath } from './rbac'

export function readNextPath(value: unknown): string | undefined {
  return typeof value === 'string' && isSafeRedirectPath(value)
    ? value
    : undefined
}
