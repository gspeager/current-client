import { createContext, useContext } from 'react'

// Selects a commit in History from wherever it's named, such as a blame line.
// null where History can't be reached, as in a test of one view.
export const OpenCommitContext = createContext<((sha: string) => void) | null>(null)

export function useOpenCommit() {
  return useContext(OpenCommitContext)
}
