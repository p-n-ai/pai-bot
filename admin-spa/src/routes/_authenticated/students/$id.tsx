import { createFileRoute } from '@tanstack/react-router'

import { StudentDetailRouteSection } from '@/components/entity-detail-route-section'

export const Route = createFileRoute('/_authenticated/students/$id')({
  component: StudentDetailRoute,
})

function StudentDetailRoute() {
  const { id: studentID } = Route.useParams()

  return <StudentDetailRouteSection studentID={studentID} />
}
