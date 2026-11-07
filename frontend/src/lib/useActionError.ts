import { useCallback, useState } from 'react'
import { errorMessage } from './errors'

export function useActionError() {
  const [error, setError] = useState<string | null>(null)
  const fail = useCallback((err: unknown) => setError(errorMessage(err)), [])
  return { error, setError, fail }
}
