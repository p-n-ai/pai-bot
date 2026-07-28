import { readPublicStatusSnapshot } from './public-status-types'
import type { PublicStatusSnapshot } from './public-status-types'

export class PublicStatusError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'PublicStatusError'
  }
}

export async function readPublicStatus(
  fetcher: typeof fetch = fetch,
): Promise<PublicStatusSnapshot> {
  const response = await fetcher('/health/status', {
    cache: 'no-store',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new PublicStatusError('Public status is unavailable')
  }

  const payload: unknown = await response.json()
  const status = readPublicStatusSnapshot(payload)
  if (!status) {
    throw new PublicStatusError('Invalid public status response')
  }

  return status
}
