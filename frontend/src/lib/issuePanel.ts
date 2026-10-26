import { createContext, useContext } from 'react'

export type OpenIssue = (key: string) => void

// Set while the host is on; a reference, a branch's issue card or a change's
// Closes list opens the issue panel through it instead of the host's page.
export const IssuePanelContext = createContext<OpenIssue | null>(null)

export function useOpenIssue(): OpenIssue | null {
  return useContext(IssuePanelContext)
}
