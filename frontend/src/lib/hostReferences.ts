import type { TrackerLinks } from '@current-client-bindings/app'
import type { Links } from '@current-client-bindings/core/hostlink'

export interface HostReference {
  text: string
  url: string
}

export type ReferenceSegment = string | HostReference

// A reference starts the text or follows a non-word character, so "foo#1"
// inside an identifier and the fragment of a URL stay text.
const REFERENCE = /(^|[^\w/.])(?:([\w.-]+(?:\/[\w.-]+)+)#(\d+)|#(\d+)|!(\d+))(?![\w-])/g

function referenceUrl(links: Links, project: string, projectIssue: string, issue: string, mergeRequest: string) {
  if (project) return links.root + project + links.issuePath + projectIssue
  if (issue) return links.issue + issue
  return links.mergeRequest ? links.mergeRequest + mergeRequest : null
}

export function splitReferences(
  text: string,
  links: Links | null,
  tracker: TrackerLinks | null = null,
): ReferenceSegment[] {
  const host = links ? splitHostReferences(text, links) : [text]
  if (!tracker) return host
  return host.flatMap((segment) => (typeof segment === 'string' ? splitTrackerKeys(segment, tracker) : [segment]))
}

// A tracker's pattern has the key in group 1 after a character that isn't
// part of it, so the key starts where the match ends minus its length.
function splitTrackerKeys(text: string, tracker: TrackerLinks): ReferenceSegment[] {
  const segments: ReferenceSegment[] = []
  let last = 0
  for (const match of text.matchAll(new RegExp(tracker.keyPattern, 'g'))) {
    const key = match[1]
    const start = match.index + match[0].length - key.length
    if (start > last) segments.push(text.slice(last, start))
    segments.push({ text: key, url: tracker.browse + key })
    last = start + key.length
  }
  if (last < text.length) segments.push(text.slice(last))
  return segments
}

function splitHostReferences(text: string, links: Links): ReferenceSegment[] {
  const segments: ReferenceSegment[] = []
  let last = 0
  for (const match of text.matchAll(REFERENCE)) {
    const [whole, lead, project = '', projectIssue = '', issue = '', mergeRequest = ''] = match
    const url = referenceUrl(links, project, projectIssue, issue, mergeRequest)
    if (!url) continue
    const start = match.index + lead.length
    if (start > last) segments.push(text.slice(last, start))
    segments.push({ text: whole.slice(lead.length), url })
    last = match.index + whole.length
  }
  if (last < text.length) segments.push(text.slice(last))
  return segments
}

// Branches named like "123-fix-login" are for issue 123.
export function branchIssue(name: string, links: Links | null): HostReference | null {
  const match = /^(\d+)-/.exec(name)
  if (!links || !match) return null
  return { text: match[1], url: links.issue + match[1] }
}

// A branch named after a tracker key ("PROJ-12-fix-login", "feature/PROJ-12")
// and where in the name the key starts.
export function branchTrackerKey(
  name: string,
  tracker: TrackerLinks | null,
): (HostReference & { start: number }) | null {
  const match = tracker ? new RegExp(tracker.branchKeyPattern).exec(name) : null
  if (!tracker || !match) return null
  const key = match[1]
  return { text: key, url: tracker.browse + key, start: name.indexOf(key, match.index) }
}
