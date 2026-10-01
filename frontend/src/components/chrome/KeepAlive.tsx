import { useMemo, useState, type ReactNode } from 'react'
import { CurrentClientRootContext, useCurrentClientRoot } from '../../lib/currentClientRoot'
import './KeepAlive.scss'

// Keeps a view mounted from its first visit, so returning to it doesn't reload
// and redraw everything. While hidden it keeps the element it last rendered:
// new props (such as a bumped repoVersion) reach it only when it's shown again,
// and its window shortcuts pause.
function KeepAlive({ active, children }: { active: boolean; children: ReactNode }) {
  const root = useCurrentClientRoot()
  const [kept, setKept] = useState<ReactNode>(null)
  if (active && kept !== children) setKept(children)
  const context = useMemo(() => ({ ...root, active: root.active && active }), [root, active])

  const view = active ? children : kept
  if (view === null) return null
  return (
    <div className="keep-alive" hidden={!active}>
      <CurrentClientRootContext.Provider value={context}>{view}</CurrentClientRootContext.Provider>
    </div>
  )
}

export default KeepAlive
