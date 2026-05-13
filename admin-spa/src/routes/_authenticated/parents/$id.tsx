import { createFileRoute } from '@tanstack/react-router'

import { ParentSummaryRouteSection } from '@/components/entity-detail-route-section'

export const Route = createFileRoute('/_authenticated/parents/$id')({
  component: ParentSummaryRoute,
})

function ParentSummaryRoute() {
  const { id: parentID } = Route.useParams()

  return <ParentSummaryRouteSection parentID={parentID} />
}
