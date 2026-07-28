import { createFileRoute } from '@tanstack/react-router'

import { PublicStatusPage } from '@/components/status/public-status-page'

export const Route = createFileRoute('/health')({
  component: PublicStatusPage,
})
