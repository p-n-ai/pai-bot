import { AlertCircleIcon } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

export function AuthErrorAlert({
  message,
  title,
}: {
  message: string
  title: string
}) {
  if (!message) {
    return null
  }

  return (
    <Alert role='alert' variant='destructive'>
      <AlertCircleIcon data-icon='inline-start' />
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  )
}
