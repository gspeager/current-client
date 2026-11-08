import type { ReactNode } from 'react'

function DialogError({ children }: { children: ReactNode }) {
  if (!children) return null
  return (
    <p className="dialog-error" role="alert">
      {children}
    </p>
  )
}

export default DialogError
