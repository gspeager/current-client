import { useCallback, useEffect, useLayoutEffect, useRef, useState, type DependencyList } from 'react'
import { errorMessage } from './errors'

interface AsyncState<T> {
  data: T | null
  error: string | null
  loading: boolean
}

const IDLE = { data: null, error: null, loading: false }

interface AsyncDataOptions {
  refreshKey?: unknown
  keepData?: boolean
}

// Only the newest request may update state, so a slow earlier response can't
// overwrite a later one. A change in `deps` clears the data unless `keepData`
// is set; a change in `refreshKey` (or calling `reload`) always keeps it while
// refetching. `load` returning null means there is nothing to load.
export function useAsyncData<T>(
  load: () => Promise<T> | null,
  deps: DependencyList,
  { refreshKey, keepData = false }: AsyncDataOptions = {},
) {
  const [state, setState] = useState<AsyncState<T>>(IDLE)
  const loadRef = useRef(load)
  const requestId = useRef(0)
  const previousDeps = useRef<DependencyList | null>(null)

  useLayoutEffect(() => {
    loadRef.current = load
  })

  const run = useCallback((keepData: boolean) => {
    const id = ++requestId.current
    const request = loadRef.current()
    if (!request) {
      setState(IDLE)
      return
    }
    setState((prev) => ({ data: keepData ? prev.data : null, error: null, loading: true }))
    request.then(
      (data) => {
        if (id === requestId.current) setState({ data, error: null, loading: false })
      },
      (err: unknown) => {
        if (id === requestId.current) setState({ data: null, error: errorMessage(err), loading: false })
      },
    )
  }, [])

  const invalidate = useCallback(() => {
    requestId.current++
  }, [])

  useEffect(() => {
    const depsChanged =
      previousDeps.current === null || deps.some((dep, i) => !Object.is(dep, previousDeps.current?.[i]))
    previousDeps.current = deps
    run(keepData || !depsChanged)
    return invalidate
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, refreshKey])

  const reload = useCallback(() => run(true), [run])

  return { ...state, reload }
}
