import type { ReactNode } from 'react'
import GitOutput from './GitOutput'

function DialogError({ children }: { children: ReactNode }) {
  if (!children) return null
  return (
    <div className="dialog-error" role="alert">
      {children}
      {typeof children === 'string' && <GitOutput message={children} />}
    </div>
  )
}

export default DialogError
