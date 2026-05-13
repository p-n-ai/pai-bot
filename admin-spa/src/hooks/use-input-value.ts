import { useCallback, useState } from 'react'
import type { ChangeEvent } from 'react'

export function useInputValue(initialValue = '') {
  const [value, setValue] = useState(initialValue)
  const handleChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    setValue(event.target.value)
  }, [])

  return {
    handleChange,
    setValue,
    value,
  }
}
