import { useCallback, useState } from 'react'
import { errorMessage } from './errors'

// run clears the last error, runs the action and keeps its error, prefixed when
// a prefix is given; it resolves to undefined after a failure. running is true
// while one runs.
export function useAction() {
  const [error, setError] = useState<string | null>(null)
  const [running, setRunning] = useState(false)
  const fail = useCallback((err: unknown) => setError(errorMessage(err)), [])
  const run = useCallback(<T>(action: () => Promise<T>, prefix?: string): Promise<T | undefined> => {
    setError(null)
    setRunning(true)
    return action()
      .catch((err: unknown) => {
        const message = errorMessage(err)
        setError(prefix ? `${prefix}: ${message}` : message)
        return undefined
      })
      .finally(() => setRunning(false))
  }, [])
  return { run, running, error, setError, fail }
}
