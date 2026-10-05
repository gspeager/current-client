import type { FileDiff } from '@current-client-bindings/app'

export interface SubmoduleChange {
  from: string | null
  to: string | null
  // The new side has uncommitted changes inside the submodule.
  dirty: boolean
}

const SUBPROJECT = /^Subproject commit ([0-9a-f]{7,64})(-dirty)?$/

// A submodule's diff is just "Subproject commit <sha>" lines; anything else isn't one.
export function submoduleChange(diff: FileDiff): SubmoduleChange | null {
  const lines = diff.hunks.flatMap((h) => h.lines)
  if (lines.length === 0 || !lines.every((l) => SUBPROJECT.test(l.content))) return null
  const change: SubmoduleChange = { from: null, to: null, dirty: false }
  for (const line of lines) {
    const [, sha, dirty] = SUBPROJECT.exec(line.content)!
    if (line.kind !== 'added') change.from = sha
    if (line.kind !== 'removed') {
      change.to = sha
      change.dirty = Boolean(dirty)
    }
  }
  return change
}
