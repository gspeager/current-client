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

export function issueNumbersIn(texts: string[]): number[] {
  const numbers = new Set<number>()
  for (const text of texts) {
    for (const match of text.matchAll(ISSUE)) numbers.add(Number(match[1]))
  }
  return [...numbers]
}

export function trackerKeysIn(texts: string[], keyPattern: string): string[] {
  const keys = new Set<string>()
  const pattern = new RegExp(keyPattern, 'g')
  for (const text of texts) {
    for (const match of text.matchAll(pattern)) keys.add(match[1])
  }
  return [...keys]
}

export function useReferenceCards(repoPath: string, texts: string[], changes: BranchChanges): ReferenceCards {
  const numbers = useMemo(() => issueNumbersIn(texts), [texts])
  const key = numbers.join(',')
  const [issues, setIssues] = useState<Record<number, IssueInfo>>({})
  const asked = useRef(new Set<number>())
  // A tracker's keys get cards only while its integration is on.
  const tracker = useTracker()
  const trackerPattern = tracker?.reading ? tracker.keyPattern : ''
  const trackerKeys = useMemo(
    () => (trackerPattern ? trackerKeysIn(texts, trackerPattern) : []),
    [texts, trackerPattern],
  )
  const trackerKey = trackerKeys.join(',')
  const [trackerIssues, setTrackerIssues] = useState<Record<string, IssueInfo | undefined>>({})
  const askedKeys = useRef(new Set<string>())

  useEffect(() => {
    asked.current.clear()
    setIssues({})
    askedKeys.current.clear()
    setTrackerIssues({})
  }, [repoPath, trackerPattern])

  useEffect(() => {
    const fresh = trackerKeys.filter((k) => !askedKeys.current.has(k))
    if (fresh.length === 0) return
    const timer = window.setTimeout(() => {
      fresh.forEach((k) => askedKeys.current.add(k))
      HostService.FetchTrackerIssueTitles(repoPath, fresh)
        .then((answer) => setTrackerIssues((prev) => ({ ...prev, ...answer })))
        .catch(() => fresh.forEach((k) => askedKeys.current.delete(k)))
    }, DEBOUNCE_MS)
    return () => window.clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath, trackerKey])

  useEffect(() => {
    const fresh = numbers.filter((n) => !asked.current.has(n))
    if (fresh.length === 0) return
    const timer = window.setTimeout(() => {
      fresh.forEach((n) => asked.current.add(n))
      HostService.FetchIssueTitles(repoPath, fresh)
        .then((answer) => setIssues((prev) => ({ ...prev, ...answer })))
        .catch(() => fresh.forEach((n) => asked.current.delete(n)))
    }, DEBOUNCE_MS)
    return () => window.clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath, key])

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
