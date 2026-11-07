import { useEffect, useMemo, useRef, useState } from 'react'
import { HostService, type IssueInfo } from '@current-client-bindings/app'
import type { BranchChanges } from './useBranchChanges'
import { useTracker } from './issuePanel'

export interface ReferenceCardInfo {
  title: string
  state: string
  // false when the host hasn't been asked about it (yet, or because
  // integration is off).
  known: boolean
}

// What a hover card can say about each "#123" or "!45" in the given texts:
// issue titles fetched once per number (the host service remembers them
// until a refresh, and never asks twice), merge request titles from the
// cache of open changes. Keyed by the reference's text.
export type ReferenceCards = Record<string, ReferenceCardInfo | undefined>

const ISSUE = /(?:^|[^\w/.])#(\d+)(?![\w-])/g
const DEBOUNCE_MS = 250

function issueNumbersIn(texts: string[]): number[] {
  const numbers = new Set<number>()
  for (const text of texts) {
    for (const match of text.matchAll(ISSUE)) numbers.add(Number(match[1]))
  }
  return [...numbers]
}

function trackerKeysIn(texts: string[], keyPattern: string): string[] {
  const keys = new Set<string>()
  const pattern = new RegExp(keyPattern, 'g')
  for (const text of texts) {
    for (const match of text.matchAll(pattern)) keys.add(match[1])
  }
  return [...keys]
}

type Titles = Record<string, IssueInfo | undefined>

// Each key is asked for once, after a pause; a failed ask may be retried.
function useTitlesOnce<K extends string | number>(
  repoPath: string,
  resetKey: string,
  keys: K[],
  fetch: (repoPath: string, keys: K[]) => Promise<Titles>,
): Titles {
  const [titles, setTitles] = useState<Titles>({})
  const asked = useRef(new Set<K>())
  const key = keys.join(',')

  useEffect(() => {
    asked.current.clear()
    setTitles({})
  }, [repoPath, resetKey])

  useEffect(() => {
    const fresh = keys.filter((k) => !asked.current.has(k))
    if (fresh.length === 0) return
    const timer = window.setTimeout(() => {
      fresh.forEach((k) => asked.current.add(k))
      fetch(repoPath, fresh)
        .then((answer) => setTitles((prev) => ({ ...prev, ...answer })))
        .catch(() => fresh.forEach((k) => asked.current.delete(k)))
    }, DEBOUNCE_MS)
    return () => window.clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath, key])

  return titles
}

export function useReferenceCards(repoPath: string, texts: string[], changes: BranchChanges): ReferenceCards {
  const numbers = useMemo(() => issueNumbersIn(texts), [texts])
  // A tracker's keys get cards only while its integration is on.
  const tracker = useTracker()
  const trackerPattern = tracker?.reading ? tracker.keyPattern : ''
  const trackerKeys = useMemo(
    () => (trackerPattern ? trackerKeysIn(texts, trackerPattern) : []),
    [texts, trackerPattern],
  )
  const issues = useTitlesOnce(repoPath, trackerPattern, numbers, HostService.FetchIssueTitles)
  const trackerIssues = useTitlesOnce(repoPath, trackerPattern, trackerKeys, HostService.FetchTrackerIssueTitles)

  return useMemo(() => {
    const cards: ReferenceCards = {}
    for (const n of numbers) {
      const issue = issues[n]
      cards[`#${n}`] = issue
        ? { title: issue.title, state: issue.state, known: issue.known }
        : { title: '', state: '', known: false }
    }
    for (const k of trackerKeys) {
      const issue = trackerIssues[k]
      cards[k] = issue
        ? { title: issue.title, state: issue.status || issue.state, known: issue.known }
        : { title: '', state: '', known: false }
    }
    for (const change of Object.values(changes)) {
      if (change?.title) {
        cards[`${change.prefix}${change.number}`] = {
          title: change.title,
          state: change.draft ? 'draft' : change.state,
          known: true,
        }
      }
    }
    return cards
  }, [numbers, issues, trackerKeys, trackerIssues, changes])
}
