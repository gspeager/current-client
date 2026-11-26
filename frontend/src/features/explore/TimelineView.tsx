import { useEffect, useMemo, useState } from 'react'
import { useEventCallback } from 'usehooks-ts'
import {
  ExploreService,
  type ExploreComparison,
  type TagInfo,
  type TimelineFileInfo,
} from '@current-client-bindings/app'
import { useOpenCommit } from '../../lib/openCommit'
import { plural } from '../../lib/plural'
import { shortSha } from '../../lib/text'
import { useAsyncData } from '../../lib/useAsyncData'
import FolderPath from './FolderPath'
import TimelineChart from './TimelineChart'
import { linesIn, spanStart, timelineLanes, when, type Bubble, type Entry, type Span } from './timelineLanes'
import type { TimelineLayout } from './timelineLayout'

interface TimelineViewProps {
  repoPath: string
  comparison: ExploreComparison
  // Today or this week in place of the comparison, from its start when read.
  span: Span | null
  // Changes with uncommitted edits.
  version: unknown
  // The code at HEAD, which its folder areas come from.
  code: { path: string; lines: number }[]
  slotOf: (path: string) => number
  listed: (file: TimelineFileInfo) => boolean
  tags: TagInfo[]
  folder: string
  onFolder: (folder: string) => void
  onOpenWorkingFile: (path: string) => void
  // null while there's nothing to save.
  onLayout: (layout: TimelineLayout | null) => void
}

// When and where the code changed: a lane for each of the folder's areas, a bubble
// for each commit's lines in one.
function TimelineView({
  repoPath,
  comparison,
  span,
  version,
  code,
  slotOf,
  listed,
  tags,
  folder,
  onFolder,
  onOpenWorkingFile,
  onLayout,
}: TimelineViewProps) {
  const openCommit = useOpenCommit()
  const [active, setActive] = useState<Entry | null>(null)
  const { data, error } = useAsyncData(
    () => {
      const asked: ExploreComparison = span ? { kind: 'since', ref: spanStart(span).toISOString() } : comparison
      return ExploreService.Timeline(repoPath, asked).then((info) => ({ info, comparison: asked, at: Date.now() }))
    },
    [repoPath, comparison, span],
    { refreshKey: version, keepData: true },
  )
  const { lanes, entries } = useMemo(
    () => (data ? timelineLanes(data.info, code, folder, slotOf, listed, data.at) : { lanes: [], entries: [] }),
    [data, code, folder, slotOf, listed],
  )
  const range = useMemo(
    () => ({
      since: data?.comparison.kind === 'since' ? Date.parse(data.comparison.ref) : null,
      now: data?.info.now ? data.at : null,
    }),
    [data],
  )
  const tagged = useMemo(() => new Map(tags.map((tag) => [tag.sha, tag.name])), [tags])
  const empty = lanes.length === 0
  const report = useEventCallback(onLayout)
  useEffect(() => {
    if (empty) report(null)
  }, [empty, report])
  useEffect(() => () => report(null), [report])

  const describe = (entry: Entry) =>
    entry.sha
      ? `${entry.subject}, ${shortSha(entry.sha)} at ${when(entry.time)}, ${linesIn(entry)}`
      : `Uncommitted changes, ${linesIn(entry)}`

  const card = (entry: Entry) => (
    <>
      <p className="hover-card-title">{entry.subject}</p>
      <p className="hover-card-line hover-card-data">
        {entry.sha ? `${shortSha(entry.sha)} · ${when(entry.time)}` : 'Not committed'}
      </p>
      <p className="hover-card-line">{linesIn(entry)}</p>
      {entry.sha && tagged.has(entry.sha) && (
        <p className="hover-card-line">
          Tagged <span className="hover-card-data">{tagged.get(entry.sha)}</span>
        </p>
      )}
      {(!entry.sha || openCommit) && (
        <p className="hover-card-line hover-card-muted">
          {entry.sha ? 'Click to open in History' : 'Click to open in Changes'}
        </p>
      )}
    </>
  )

  const open = (entry: Entry, bubble: Bubble | null) => {
    if (entry.sha) openCommit?.(entry.sha)
    else onOpenWorkingFile((bubble ?? entry.bubbles[0]).path)
  }

  const committed = entries.filter((entry) => entry.sha !== null)
  const lines = entries.reduce((sum, entry) => sum + entry.lines, 0)
  const hint = [
    'A bubble is a commit’s lines in a folder, dashed when not committed. A folder’s name opens it.',
    data?.info.limit && `Only the latest ${data.info.limit.toLocaleString()} commits are shown.`,
  ]
    .filter(Boolean)
    .join(' ')
  return (
    <>
      <div className="explore-toolbar">
        <FolderPath folder={folder} onFolder={onFolder} />
        <span className="explore-total">
          {plural(committed.length, 'commit')} · {lines.toLocaleString()} lines changed
        </span>
        {data?.info.target && <span className="explore-note">against {data.info.target}</span>}
      </div>
      {error ? (
        <p className="explore-message explore-error">{error}</p>
      ) : !data ? (
        <p className="explore-message">Reading commits…</p>
      ) : empty ? (
        <p className="explore-message">
          {data.comparison.kind === 'branch' && !data.info.target
            ? 'This repository has no default branch to compare with.'
            : 'Nothing has changed.'}
        </p>
      ) : (
        <TimelineChart
          lanes={lanes}
          entries={entries}
          tags={tags}
          range={range}
          folder={folder}
          describe={describe}
          card={card}
          onOpen={open}
          onFolder={onFolder}
          onActive={setActive}
          onLayout={report}
        />
      )}
      <div className="explore-facts">
        {active ? (
          <span className="explore-fact">{describe(active)}</span>
        ) : (
          <span className="explore-hint">{hint}</span>
        )}
      </div>
    </>
  )
}

export default TimelineView
