import { useCallback, useState } from 'react'

export function useSubmitStatus(initialError = '') {
  const [error, setError] = useState(initialError)
  const [isPending, setIsPending] = useState(false)

  const beginSubmit = useCallback(() => {
    setIsPending(true)
    setError('')
  }, [])

  const finishSubmit = useCallback(() => {
    setIsPending(false)
  }, [])

  return {
    beginSubmit,
    error,
    finishSubmit,
    isPending,
    setError,
  }
}
