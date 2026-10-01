import { useRef, useState } from 'react'
import { errorMessage } from './errors'
import { withSignIn, type SignIn } from './signIn'
import { useDialogs } from './useDialogs'

interface Cancellable<T> extends Promise<T> {
  cancel(): Promise<void>
}

function isCancellation(err: unknown): boolean {
  return err instanceof Error && err.name === 'CancelError'
}

export function useCancellableOperation() {
  const { signIn } = useDialogs()
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const current = useRef<Cancellable<unknown> | null>(null)

  // start is called again with what the user enters if the remote asks them
  // to sign in. onError returns true when it has handled the failure itself.
  function run<T>(
    start: (auth: SignIn | null) => Cancellable<T>,
    onSuccess?: (value: T) => void,
    onError?: (message: string) => boolean,
  ) {
    setError(null)
    setRunning(true)
    return withSignIn((auth) => {
      const promise = start(auth)
      current.current = promise
      return promise
    }, signIn)
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
