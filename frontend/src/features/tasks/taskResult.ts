import type { TaskResultInfo } from '@current-client-bindings/app'
import { relativeTime } from '../../lib/relativeTime'

const MOST_NAMED = 3

export const STATE_LABEL: Record<string, string> = {
  current: 'on this code',
  outdated: 'out of date',
  changed: 'files changed while it ran',
  unknown: 'may be out of date',
}

// Whether it passed, when, and whether that's on the code as it is now.
export function describeResult(result: TaskResultInfo): string {
  const when = relativeTime(new Date(result.at))
  return `${result.passed ? 'Passed' : 'Failed'} ${when}, ${STATE_LABEL[result.state]}`
}

export function explainResult(result: TaskResultInfo): string {
  const changed = result.changed ?? []
  if (result.state === 'changed') {
    const more = changed.length - MOST_NAMED
    return `Changed while it ran: ${changed.slice(0, MOST_NAMED).join(', ')}${more > 0 ? ` and ${more} more` : ''}. Run it again, or have Git ignore what it writes.`
  }
  return 'Compared by every file Git tracks or would add. Ignored files, uncommitted work inside submodules and untracked files over 10 MB aren’t counted.'
}
