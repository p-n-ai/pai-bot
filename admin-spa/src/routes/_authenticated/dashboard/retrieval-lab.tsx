import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { RetrievalLab } from '@/components/retrieval/retrieval-lab'

export const Route = createFileRoute('/_authenticated/dashboard/retrieval-lab')(
  {
    component: RetrievalLabRoute,
  },
)

function RetrievalLabRoute() {
  return (
    <AdminPageSection
      description='Try retrieval queries, filters, and repeat runs through the backend search endpoint.'
      eyebrow='Retrieval'
      title='BM25 query lab'
    >
      <RetrievalLab />
    </AdminPageSection>
  )
}
