import { useMemo } from 'react'
import type {
  ChangeArea,
  ChangeReason,
  ChangeStructureInfo,
  StructureLink,
  StructureNode,
} from '@current-client-bindings/app'
import StructureGraph from '../explore/StructureGraph'
import { counted } from '../explore/structureWords'
import './ChangeStructure.scss'

// More than this many links between folders, or folders to draw, and a change is a
// restructure, listed rather than drawn.
const MOST_DRAWN = { links: 20, areas: 14 }
const MOST_LISTED = 30

interface ChangeStructureProps {
  shape: ChangeStructureInfo | null
  error: string | null
  onArea: (path: string) => void
  onFile: (path: string) => void
}

const folderName = (path: string) => (path === '' ? 'Top folder' : `${path}/`)

function grown({ added, removed }: ChangeArea): string {
  return `+${added.toLocaleString()} -${removed.toLocaleString()}`
}

const SKIM: Record<string, string> = {
  lockfile: 'Lockfile',
  generated: 'Generated',
  renamed: 'Renamed, unchanged',
}

function why({ why, folder, count }: ChangeReason): string {
  if (why === 'used') return folder ? `used by ${count} of the change’s folders` : `used by ${counted(count, 'file')}`
  if (why === 'few') return `${counted(count, 'line')} in all`
  return SKIM[why] ?? why
}

// The areas a change touches, the links between folders it adds or removes, what to
// read first and what to skim.
function ChangeStructure({ shape, error, onArea, onFile }: ChangeStructureProps) {
  const drawn = useMemo(() => (shape ? drawing(shape) : null), [shape])
  if (error) return <p className="changed-files-browser-error">{error}</p>
  if (!shape) return <p className="changed-files-browser-hint">Reading what uses what…</p>
  const large = shape.links.length > MOST_DRAWN.links || (drawn?.nodes.length ?? 0) > MOST_DRAWN.areas
  const added = shape.areas.filter((a) => a.state === 'added').length
  const removed = shape.areas.filter((a) => a.state === 'removed').length
  const open = (reason: ChangeReason) => (reason.folder ? onArea(reason.path) : onFile(reason.path))
  return (
    <section className="read-through-notes change-structure" aria-label="Structure">
      <h3 className="changed-files-browser-label">Structure</h3>
      <p className="change-structure-summary">
        {[
          counted(shape.areas.length, 'folder'),
          added > 0 && `${added} new`,
          removed > 0 && `${removed} gone`,
          shape.read &&
            (shape.links.length === 0
              ? 'no link between folders added or removed'
              : `${counted(shape.links.length, 'link')} between folders added or removed`),
        ]
          .filter(Boolean)
          .join(' · ')}
      </p>
      {!shape.read && (
        <p className="changed-files-browser-hint">
          {shape.partial
            ? 'Links aren’t read in a partial clone, where the base’s files would be fetched. Folders only.'
            : 'No links: none of its languages is read. Folders only.'}
        </p>
      )}
      {large ? (
        <Restructure shape={shape} onArea={onArea} />
      ) : (
        drawn &&
        drawn.links.length > 0 && (
          <StructureGraph
            label="Folders the change touches, and the links it adds or removes"
            nodes={drawn.nodes}
            links={drawn.links}
            commitsOf={() => undefined}
            compared
            describe={(node) => drawn.facts.get(node.path) ?? ''}
            onOpen={onArea}
          />
        )
      )}
      {!large && (drawn?.links.length ?? 0) === 0 && <AreaList areas={shape.areas} onArea={onArea} />}
      <Reasons title="Read first" reasons={shape.readFirst} open={open} />
      <Reasons title="Skim" reasons={shape.skim} open={open} />
    </section>
  )
}

// Each area a card, with the folders its new or gone links go to.
function drawing(shape: ChangeStructureInfo) {
  const facts = new Map<string, string>()
  const nodes: StructureNode[] = []
  const at = new Map<string, number>()
  const add = (path: string, area?: ChangeArea) => {
    if (at.has(path)) return at.get(path)!
    at.set(path, nodes.length)
    if (area) facts.set(path, `${counted(area.files.length, 'file')} · ${grown(area)}`)
    else facts.set(path, 'not changed')
    nodes.push({
      path,
      label: folderName(path),
      side: 'card',
      folder: true,
      count: { files: area?.files.length ?? 0, lines: 0, changed: area?.files.length ?? 0 },
      state: area?.state ?? '',
      doc: '',
      slot: area ? (nodes.length % 8) + 1 : 0,
    })
    return nodes.length - 1
  }
  shape.areas.forEach((area) => add(area.path, area))
  const links: StructureLink[] = shape.links.map(({ from, to, count, state }) => ({
    from: add(from),
    to: add(to),
    count,
    main: true,
    state,
  }))
  return { nodes, links, facts }
}

function AreaList({ areas, onArea }: { areas: ChangeArea[]; onArea: (path: string) => void }) {
  return (
    <ul className="read-through-note-list change-structure-list">
      {areas.map((area) => (
        <li key={area.path}>
          <button type="button" className="change-structure-place" onClick={() => onArea(area.path)}>
            {folderName(area.path)}
          </button>
          <span className="change-structure-facts">
            {counted(area.files.length, 'file')} · {grown(area)}
            {area.state && ` · ${area.state === 'added' ? 'new' : 'gone'}`}
          </span>
        </li>
      ))}
    </ul>
  )
}

// A large restructure, in words: which links come and go, then the folders.
function Restructure({ shape, onArea }: { shape: ChangeStructureInfo; onArea: (path: string) => void }) {
  const listed = shape.links.slice(0, MOST_LISTED)
  return (
    <div className="change-structure-restructure">
      <p className="changed-files-browser-hint">A large restructure, so it’s listed rather than drawn.</p>
      {listed.length > 0 && (
        <ul className="read-through-note-list change-structure-list" aria-label="Links added or removed">
          {listed.map(({ from, to, count, state }) => (
            <li key={`${from}>${to}`}>
              <span className="change-structure-mark">{state === 'added' ? '+' : '−'}</span>
              <button type="button" className="change-structure-place" onClick={() => onArea(from)}>
                {folderName(from)}
              </button>
              uses
              <button type="button" className="change-structure-place" onClick={() => onArea(to)}>
                {folderName(to)}
              </button>
              <span className="change-structure-facts">
                {counted(count, 'link')} {state === 'added' ? 'new' : 'gone'}
              </span>
            </li>
          ))}
        </ul>
      )}
      {shape.links.length > listed.length && (
        <p className="changed-files-browser-hint">{(shape.links.length - listed.length).toLocaleString()} more.</p>
      )}
      <AreaList areas={shape.areas} onArea={onArea} />
    </div>
  )
}

function Reasons({
  title,
  reasons,
  open,
}: {
  title: string
  reasons: ChangeReason[]
  open: (r: ChangeReason) => void
}) {
  if (reasons.length === 0) return null
  return (
    <>
      <h4 className="second-look-label">{title}</h4>
      <ul className="read-through-note-list change-structure-list" aria-label={title}>
        {reasons.map((reason) => (
          <li key={`${reason.why}:${reason.path}`}>
            <button type="button" className="change-structure-place" onClick={() => open(reason)}>
              {reason.folder ? folderName(reason.path) : reason.path}
            </button>
            <span className="change-structure-facts">{why(reason)}</span>
          </li>
        ))}
      </ul>
    </>
  )
}

export default ChangeStructure
