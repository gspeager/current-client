import { useEffect, useMemo, useRef, useState } from 'react'
import {
  ExploreService,
  HistoryService,
  type ExploreComparison,
  type StructureInfo,
  type StructureWide,
} from '@current-client-bindings/app'
import { counted } from './structureWords'
import { relativeTime } from '../../lib/relativeTime'
import { shortSha } from '../../lib/text'
import { useAsyncData } from '../../lib/useAsyncData'
import StructureGraph from './StructureGraph'
import { drawnOf } from './structureLayout'
import './StructureView.scss'

const LATEST = 8

interface StructureViewProps {
  repoPath: string
  folder: string
  onFolder: (folder: string) => void
  comparison: ExploreComparison | null
  // Changes with what the code is, and with uncommitted edits.
  version: unknown
  commitsOf: (path: string) => number | undefined
  onHistory: (path: string) => void
  onBlame: (path: string) => void
  // Not given where nothing is compared.
  onDiff?: (path: string) => void
}

// The code as cards to go into, from the whole repository to a file.
function StructureView({
  repoPath,
  folder,
  onFolder,
  comparison,
  version,
  commitsOf,
  onHistory,
  onBlame,
  onDiff,
}: StructureViewProps) {
  const [all, setAll] = useState(false)
  const box = useRef<HTMLDivElement>(null)
  const { data: level, error } = useAsyncData(
    () => ExploreService.Structure(repoPath, folder, comparison),
    [repoPath, folder, comparison],
    { refreshKey: version, keepData: true },
  )
  // Going to another level replaces the card that had the focus.
  const within = useRef(false)
  const shown = level?.path
  useEffect(() => {
    if (within.current) box.current?.querySelector<HTMLElement>('.structure-card[tabindex="0"]')?.focus()
    within.current = false
  }, [shown])
  const drawn = useMemo(() => (level ? drawnOf(level.nodes, level.links, all) : { nodes: [], links: [] }), [level, all])
  const go = (path: string) => {
    within.current = !!box.current?.contains(document.activeElement)
    setAll(false)
    onFolder(path)
  }

  if (error) return <p className="explore-message explore-error">{error}</p>
  if (!level) return <p className="explore-message">Reading what uses what…</p>
  const above = level.crumbs.length > 1 ? level.crumbs[level.crumbs.length - 2].path : null
  const more = level.links.some((link) => !link.main)
  return (
    <div className="structure-view" ref={box}>
      <div className="explore-toolbar">
        <nav className="explore-path" aria-label="Folder">
          {level.crumbs.map(({ path, label }) => (
            <button
              key={path}
              type="button"
              className="explore-crumb"
              disabled={path === level.path}
              onClick={() => go(path)}
            >
              {label || 'All'}
            </button>
          ))}
        </nav>
        <span className="explore-total">
          {counted(level.count.files, 'file')} · {counted(level.count.lines, 'line')}
          {level.compared && ` · ${level.count.changed.toLocaleString()} changed`}
        </span>
        {level.target && <span className="explore-note">against {level.target}</span>}
        {level.partial && <span className="explore-note">Not compared: a partial clone would fetch</span>}
        {more && (
          <button type="button" className="structure-all" aria-pressed={all} onClick={() => setAll(!all)}>
            {all ? 'Main links' : `All links (${level.links.length})`}
          </button>
        )}
      </div>
      {level.doc && <p className="structure-doc">{level.doc}.</p>}
      <Widely level={level} onOpen={go} />
      {level.nodes.length === 0 ? (
        <p className="explore-message">No files to show.</p>
      ) : (
        <StructureGraph
          key={level.path}
          label={`Structure of ${level.path || 'the repository'}`}
          nodes={drawn.nodes}
          links={drawn.links}
          commitsOf={commitsOf}
          compared={level.compared}
          onOpen={(path) => path !== level.path && go(path)}
          onUp={above === null ? undefined : () => go(above)}
        />
      )}
      {level.rest.length > 0 && (
        <details className="structure-rest">
          <summary>{level.rest.length.toLocaleString()} more, less connected</summary>
          <ul>
            {level.rest.map(({ path, label, state }) => (
              <li key={path}>
                <button type="button" className="structure-place" title={path} onClick={() => go(path)}>
                  {label}
                </button>
                {state && <span className="explore-note">{state === 'added' ? 'new' : 'removed'}</span>}
              </li>
            ))}
          </ul>
        </details>
      )}
      {!level.folder && (
        <FileEnd
          repoPath={repoPath}
          path={level.path}
          version={version}
          onHistory={onHistory}
          onBlame={onBlame}
          onDiff={level.compared && level.count.changed > 0 ? onDiff : undefined}
        />
      )}
      <p className="explore-hint">{whatIsRead(level)}</p>
    </div>
  )
}

function whatIsRead({ languages, unread, moreUsedBy, moreUses, compared }: StructureInfo): string {
  const others = moreUsedBy + moreUses
  return [
    languages.length > 0
      ? `Links are read from ${languages.join(', ')}.`
      : 'No links: none of its languages is read. Folders and history only.',
    unread > 0 && `${counted(unread, 'file')} of code in other languages ${unread === 1 ? "isn't" : "aren't"} read.`,
    compared && '+ is a link that’s new, − one that’s gone.',
    others > 0 && `${counted(others, 'other place')} outside ${others === 1 ? "isn't" : "aren't"} shown.`,
    'Tests, generated files and lockfiles are left out.',
  ]
    .filter(Boolean)
    .join(' ')
}

// What most cards share, in words instead of a line from each.
function Widely({ level, onOpen }: { level: StructureInfo; onOpen: (path: string) => void }) {
  const said = (words: string, places: StructureWide[]) =>
    places.length > 0 && (
      <span>
        {words}{' '}
        {places.map(({ path, label, cards }) => (
          <span key={path} className="structure-wide">
            <button type="button" className="structure-place" title={path} onClick={() => onOpen(path)}>
              {label}
            </button>{' '}
            {cards.toLocaleString()} of {level.cards.toLocaleString()}
          </span>
        ))}
      </span>
    )
  if (level.widelyUsedBy.length + level.widelyUses.length === 0) return null
  return (
    <p className="structure-widely">
      {said('Widely used by', level.widelyUsedBy)}
      {said('Widely uses', level.widelyUses)}
    </p>
  )
}

interface FileEndProps {
  repoPath: string
  path: string
  version: unknown
  onHistory: (path: string) => void
  onBlame: (path: string) => void
  onDiff?: (path: string) => void
}

function FileEnd({ repoPath, path, version, onHistory, onBlame, onDiff }: FileEndProps) {
  const { data: commits, error } = useAsyncData(
    () => HistoryService.GetFileHistory(repoPath, path, LATEST, 0),
    [repoPath, path],
    { refreshKey: version },
  )
  return (
    <section className="structure-file" aria-label="Latest commits">
      <div className="explore-toolbar">
        <h3>Latest commits</h3>
        <button type="button" className="structure-all" onClick={() => onHistory(path)}>
          History
        </button>
        <button type="button" className="structure-all" onClick={() => onBlame(path)}>
          Blame
        </button>
        {onDiff && (
          <button type="button" className="structure-all" onClick={() => onDiff(path)}>
            Diff
          </button>
        )}
      </div>
      {error && <p className="explore-message explore-error">{error}</p>}
      {commits?.length === 0 && <p className="explore-hint">No commits for this file.</p>}
      <ul>
        {commits?.map(({ sha, subject, authorName, date }) => (
          <li key={sha}>
            <span className="structure-sha">{shortSha(sha)}</span>
            <span className="structure-subject">{subject}</span>
            <span className="explore-note">
              {authorName}, {relativeTime(new Date(date))}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}

export default StructureView
