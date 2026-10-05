import type { FileDiff } from '@current-client-bindings/app'

export interface LfsChange {
  oldSize: number | null
  newSize: number | null
}

// The lines of a Git LFS pointer file: https://github.com/git-lfs/git-lfs/blob/main/docs/spec.md
const POINTER_LINE = /^(version https:\/\/git-lfs\.github\.com\/spec\/v\d+|oid sha256:[0-9a-f]+|size \d+|ext-\S+ \S+)$/

// Git diffs an LFS file as its pointer text; this reads the sizes out of it.
export function lfsPointerChange(diff: FileDiff): LfsChange | null {
  const lines = diff.hunks.flatMap((h) => h.lines)
  if (!lines.some((l) => l.content.startsWith('version https://git-lfs.github.com/spec/'))) return null
  if (!lines.every((l) => POINTER_LINE.test(l.content))) return null
  const change: LfsChange = { oldSize: null, newSize: null }
  for (const line of lines) {
    if (!line.content.startsWith('size ')) continue
    const size = Number(line.content.slice(5))
    if (line.kind !== 'added') change.oldSize = size
    if (line.kind !== 'removed') change.newSize = size
  }
  return change
}
