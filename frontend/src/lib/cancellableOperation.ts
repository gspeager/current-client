import { useRef, useState } from 'react'
import { errorMessage } from './errors'
import { withSignIn, type SignIn } from './signIn'
import { useDialogs } from './useDialogs'

interface Cancellable<T> extends Promise<T> {
  cancel(): Promise<void>
}

interface OperationState {
  running: boolean
  error: string | null
}

const IDLE: OperationState = { running: false, error: null }

function isCancellation(err: unknown): boolean {
  return err instanceof Error && err.name === 'CancelError'
}

// Each key (such as a repository path) has its own operation, so one started
// under a key keeps running and reporting there while another key is current.
export function useCancellableOperation(key = '') {
  const { signIn } = useDialogs()
  const [states, setStates] = useState<Record<string, OperationState>>({})
  const current = useRef(new Map<string, Cancellable<unknown>>())

  const update = (patch: Partial<OperationState>) =>
    setStates((prev) => ({ ...prev, [key]: { ...(prev[key] ?? IDLE), ...patch } }))

  // start is called again with what the user enters if the remote asks them
  // to sign in. onError returns true when it has handled the failure itself.
  function run<T>(
    start: (auth: SignIn | null) => Cancellable<T>,
    onSuccess?: (value: T) => void,
    onError?: (message: string) => boolean,
  ) {
    update({ running: true, error: null })
    return withSignIn((auth) => {
      const promise = start(auth)
      current.current.set(key, promise)
      return promise
    }, signIn)
      .then((value) => onSuccess?.(value))
      .catch((err: unknown) => {
        if (isCancellation(err)) return
        const message = errorMessage(err)
        if (!onError?.(message)) {
          update({ error: message })
        }
      })
      .finally(() => {
        update({ running: false })
        current.current.delete(key)
      })
  }

  function cancel() {
    current.current.get(key)?.cancel()
  }

  const { running, error } = states[key] ?? IDLE
  return { running, error, setError: (message: string | null) => update({ error: message }), run, cancel }
}
