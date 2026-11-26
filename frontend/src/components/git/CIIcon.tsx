import { CircleCheck, CircleDashed, CircleMinus, CircleSlash, CircleX, Clock, type LucideIcon } from 'lucide-react'
import clsx from 'clsx'
import './CIIcon.scss'

// One map for every place that shows a CI result, so the mark, the PR
// marker and the job rows always agree. The running icon doesn't spin.
const STATUSES: Record<string, { icon: LucideIcon; label: string }> = {
  success: { icon: CircleCheck, label: 'CI passed' },
  failed: { icon: CircleX, label: 'CI failed' },
  running: { icon: CircleDashed, label: 'CI running' },
  pending: { icon: Clock, label: 'CI pending' },
  canceled: { icon: CircleSlash, label: 'CI canceled' },
  skipped: { icon: CircleMinus, label: 'CI skipped' },
}

export function ciLabel(status: string): string {
  return STATUSES[status]?.label ?? `CI ${status}`
}

// Quiet: a pass in outline, for a list where only a failure should stand out.
export function CIIcon({ status, size = 14, quiet = false }: { status: string; size?: number; quiet?: boolean }) {
  const Icon = STATUSES[status]?.icon ?? CircleDashed
  return (
    <Icon
      className={clsx('ci-icon', `ci-icon-${status}`, quiet && 'ci-icon-quiet')}
      size={size}
      strokeWidth={1.75}
      aria-hidden="true"
    />
  )
}
