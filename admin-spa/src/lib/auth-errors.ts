export function readAuthDisplayError(
  caught: unknown,
  fallback: string,
  failedFetchMessage: string,
): string {
  if (!(caught instanceof Error)) {
    return fallback
  }

  return caught.message === 'Failed to fetch'
    ? failedFetchMessage
    : caught.message
}
