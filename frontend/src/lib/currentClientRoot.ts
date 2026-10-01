import { createContext, useContext } from 'react'

// The element Current Client renders into: its styles are scoped to it, and dialogs
// portal into it. active is false while a host app shows another view.
interface CurrentClientRoot {
  element: HTMLElement | null
  active: boolean
}

export const CurrentClientRootContext = createContext<CurrentClientRoot>({ element: null, active: true })

export function useCurrentClientRoot(): CurrentClientRoot {
  return useContext(CurrentClientRootContext)
}
