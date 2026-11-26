import { useEffect, useMemo, useRef, useState } from 'react'
import type { FileDiff, LineSelection } from '@current-client-bindings/app'
import { toggled } from '../../lib/sets'
import { lineKey } from './DiffPane'

// Changed lines picked for staging a part of a hunk, by hunk; another diff
// starts with none.
export function useLineSelection(diff: FileDiff) {
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const lastToggled = useRef<string | null>(null)

  useEffect(() => {
    setSelected(new Set())
    lastToggled.current = null
  }, [diff])

  // Shift-click selects every changed line between the last one toggled and this one.
  const toggle = (key: string, range: boolean) => {
    const order = diff.hunks.flatMap((h) => h.lines.map(lineKey).filter((k): k is string => k !== null))
    const from = lastToggled.current ? order.indexOf(lastToggled.current) : -1
    const to = order.indexOf(key)
    setSelected((prev) => {
      if (!range || from === -1) return toggled(prev, key)
      return new Set([...prev, ...order.slice(Math.min(from, to), Math.max(from, to) + 1)])
    })
    lastToggled.current = key
  }

  const byHunk = useMemo(() => {
    const result = new Map<string, LineSelection>()
    if (selected.size === 0) return result
    for (const h of diff.hunks) {
      const chosen = h.lines.filter((l) => selected.has(lineKey(l) ?? ''))
      result.set(h.raw, {
        added: chosen.filter((l) => l.kind === 'added').map((l) => l.newLine),
        removed: chosen.filter((l) => l.kind === 'removed').map((l) => l.oldLine),
      })
    }
    return result
  }, [diff, selected])

  return { selected, toggle, byHunk }
}
