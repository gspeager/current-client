import { createContext, useContext } from 'react'
import type { ReadThroughRequest } from '@current-client-bindings/app'

export type ReadThroughTarget = Pick<ReadThroughRequest, 'ref' | 'path' | 'target' | 'uncommitted'>

export interface ReadThroughOpening extends Partial<ReadThroughTarget> {
  // A file to open at, and a line of its new version.
  at?: { path: string; line: number }
}

type OpenReadThrough = (opening: ReadThroughOpening) => void

// What a note's target names: a ref, or a worktree by its path.
export function openingOf(key: string): ReadThroughOpening {
  const worktree = /^worktree:(.*)$/s.exec(key)
  return worktree ? { path: worktree[1] } : { ref: key }
}

export const ReadThroughContext = createContext<OpenReadThrough>(() => undefined)

export function useOpenReadThrough(): OpenReadThrough {
  return useContext(ReadThroughContext)
}
