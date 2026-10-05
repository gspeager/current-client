import { createContext, useContext } from 'react'
import Checkbox from '../../components/forms/Checkbox'

interface DiffWrap {
  wrap: boolean
  setWrap: (wrap: boolean) => void
}

// App provides the saved choice; without a provider (tests, a host app) lines wrap.
export const DiffWrapContext = createContext<DiffWrap>({ wrap: true, setWrap: () => {} })

export function useDiffWrap(): DiffWrap {
  return useContext(DiffWrapContext)
}

export function DiffWrapToggle() {
  const { wrap, setWrap } = useDiffWrap()
  return <Checkbox checked={wrap} onChange={setWrap} label="Wrap lines" />
}
