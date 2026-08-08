import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useCallback } from 'react'

import type { BuildAIPageKey } from '@/lib/build-ai-search'
import { BuildAIPage } from '@/components/build-ai/build-ai-page'
import { parseBuildAISearch } from '@/lib/build-ai-search'

export const Route = createFileRoute('/_authenticated/build-ai')({
  validateSearch: parseBuildAISearch,
  component: BuildAIRoute,
})

function BuildAIRoute() {
  const navigate = useNavigate({ from: '/build-ai' })
  const { page } = Route.useSearch()
  const handlePageChange = useCallback(
    (nextPage: BuildAIPageKey) => {
      navigate({ search: { page: nextPage } }).catch(() => {})
    },
    [navigate],
  )

  return <BuildAIPage onPageChange={handlePageChange} page={page} />
}
