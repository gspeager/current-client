import type { ReactNode } from 'react'
import './FactList.scss'

export function FactList({ children }: { children: ReactNode }) {
  return <dl className="fact-list">{children}</dl>
}

export function Fact({ label, className, children }: { label: string; className?: string; children: ReactNode }) {
  return (
    <>
      <dt>{label}</dt>
      <dd className={className}>{children}</dd>
    </>
  )
}
