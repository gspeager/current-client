import { useRef, useState } from 'react'
import { errorMessage } from './errors'

interface Cancellable<T> extends Promise<T> {
  cancel(): Promise<void>
}

function isCancellation(err: unknown): boolean {
  return err instanceof Error && err.name === 'CancelError'
}

export function useCancellableOperation() {
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const current = useRef<Cancellable<unknown> | null>(null)

  // onError returns true when it has handled the failure itself.
  function run<T>(promise: Cancellable<T>, onSuccess?: (value: T) => void, onError?: (message: string) => boolean) {
    setError(null)
    setRunning(true)
    current.current = promise
    return promise
      .then((value) => onSuccess?.(value))
      .catch((err: unknown) => {
        if (isCancellation(err)) return
        const message = errorMessage(err)
        if (!onError?.(message)) {
          setError(message)
        }
      })
      .finally(() => {
        setRunning(false)
        current.current = null
      })
  }

  function cancel() {
    current.current?.cancel()
  }

  return { running, error, setError, run, cancel }
}
