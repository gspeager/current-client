import { createContext, useContext } from 'react'

export type OpenIssue = (key: string) => void

// Set while the host is on; a reference, a branch's issue card or a change's
// Closes list opens the issue panel through it instead of the host's page.
export const IssuePanelContext = createContext<OpenIssue | null>(null)

export function useOpenIssue(): OpenIssue | null {
  return useContext(IssuePanelContext)
}

export interface IssueDraft {
  title: string
  description: string
}

export type NewIssue = (draft?: IssueDraft) => void

// Set while the host is on and its tracker takes writes: GitLab and GitHub.
export const NewIssueContext = createContext<NewIssue | null>(null)

export function useNewIssue(): NewIssue | null {
  return useContext(NewIssueContext)
}

const TODO_MARK = /\b(?:TODO|FIXME|XXX|HACK)\b(?:\([^)]*\))?[\s:-]*(.*)$/
const COMMENT_CLOSE = /\s*(?:\*\/|-->|#}|%>)\s*$/

// A diff line as a new issue: a TODO's text as the title, and where it is.
export function issueDraftFromLine(path: string, line: number, content: string): IssueDraft {
  const todo = TODO_MARK.exec(content)
  return {
    title: todo ? todo[1].replace(COMMENT_CLOSE, '').trim() : '',
    description: `\`${path}\` line ${line}\n\n\`\`\`\n${content.trim()}\n\`\`\``,
  }
}
