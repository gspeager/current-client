import { useMemo } from 'react'

// The value from an earlier render while it serializes the same, so a hook can
// depend on what an array or object holds rather than on its identity.
export function useStableValue<T>(value: T): T {
  const key = JSON.stringify(value)
  // eslint-disable-next-line react-hooks/exhaustive-deps -- keyed by what value holds
  return useMemo(() => value, [key])
}
