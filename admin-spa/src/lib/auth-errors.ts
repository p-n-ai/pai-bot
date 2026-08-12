export function readAuthDisplayError(
  cause: unknown,
  fallback: string,
  failedFetchMessage: string,
): string {
  if (!(cause instanceof Error)) {
    return fallback
  }

  return cause.message === 'Failed to fetch'
    ? failedFetchMessage
    : cause.message
}
