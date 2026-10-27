import { createContext, useContext } from 'react'
import type { TrackerLinks } from '@current-client-bindings/app'

export type OpenIssue = (key: string) => void

export interface IssueDraft {
  title: string
  description: string
}

export type NewIssue = (draft?: IssueDraft) => void

// The open repository's issues. open is set while the host is on: a
// reference, a branch's issue card or a change's Closes list opens the issue
// panel through it instead of the host's page. create is there when the
// host takes issue writes: GitLab and GitHub. tracker is the issue tracker
// beside the host, such as a Jira site, host on or off.
export const IssuePanelContext = createContext<{
  open?: OpenIssue
  create?: NewIssue
  tracker?: TrackerLinks | null
} | null>(null)

export function useOpenIssue(): OpenIssue | null {
  return useContext(IssuePanelContext)?.open ?? null
}

export function useNewIssue(): NewIssue | null {
  return useContext(IssuePanelContext)?.create ?? null
}

export function useTracker(): TrackerLinks | null {
  return useContext(IssuePanelContext)?.tracker ?? null
}

const TODO_MARK = /\b(?:TODO|FIXME|XXX|HACK)\b(?:\([^)]*\))?[\s:-]*(.*)$/
const COMMENT_CLOSE = /\s*(?:\*\/|--!?>|#}|%>)\s*$/

// A diff line as a new issue: a TODO's text as the title, and where it is.
export function issueDraftFromLine(path: string, line: number, content: string): IssueDraft {
  const todo = TODO_MARK.exec(content)
  return {
    title: todo ? todo[1].replace(COMMENT_CLOSE, '').trim() : '',
    description: `\`${path}\` line ${line}\n\n\`\`\`\n${content.trim()}\n\`\`\``,
  }
}
