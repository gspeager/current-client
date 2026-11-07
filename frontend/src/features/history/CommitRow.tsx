import { NoteMark } from '../notes/notes'
import type { CSSProperties, KeyboardEvent, MouseEvent } from 'react'
import { Ellipsis } from 'lucide-react'
import { rowKeyDown } from '../../components/controls/ContextMenu'
import { parsePrefix } from '../../lib/conventionalCommit'
import { highlightMatch } from '../../lib/highlightMatch'
import { useLaneColors } from '../../lib/laneColor'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import type {
  CIMark as CIMarkInfo,
  CommitInfo,
  EnvironmentInfo,
  GraphNode,
  IdentityInfo,
  RefInfo,
} from '@current-client-bindings/app'
import type { Links } from '@current-client-bindings/core/hostlink'
import type { BranchChanges } from '../../lib/useBranchChanges'
import type { ReferenceCards } from '../../lib/useReferenceCards'
import CommitGraph from './CommitGraph'
import { withoutRemote, type RowLaneColors } from './laneOwners'
import IdentityBadge from '../../components/git/IdentityBadge'
import ReferenceText from '../../components/git/ReferenceText'
import CIMark from '../../components/git/CIMark'
import RefBadge from './RefBadge'
import { EnvironmentMarker } from '../../components/git/EnvironmentMarker'
import { pairUpstreams } from './refPills'
import { shortSha } from '../../lib/text'
import './CommitRow.scss'

interface CommitRowProps {
  commit: CommitInfo
  graphNode?: GraphNode
  laneColors?: RowLaneColors
  isHead?: boolean
  currentBranch?: string
  laneCount: number
  rowHeight: number
  identity?: IdentityInfo
  refs?: RefInfo[]
  upstreams?: Record<string, string>
  selected: boolean
  dimmed?: boolean
  isNew?: boolean
  focusedSha?: string | null
  onFocusRef?: (sha: string) => void
  searchQuery?: string
  conventional?: boolean
  hostLinks?: Links | null
  changes?: BranchChanges
  repoPath?: string
  ci?: CIMarkInfo
  // The environments whose latest deployment is this commit.
  environments?: EnvironmentInfo[]
  referenceCards?: ReferenceCards
  style: CSSProperties
  onSelect: () => void
  onContextMenu: (e: MouseEvent | KeyboardEvent) => void
  onHighlightAuthor?: () => void
  authorHighlighted?: boolean
  // Selects the commit with its Checks open, from a job in its CI mark's list.
  onOpenChecks?: () => void
}

function CommitRow({
  commit,
  graphNode,
  laneColors,
  isHead = false,
  currentBranch,
  laneCount,
  rowHeight,
  identity,
  refs,
  upstreams,
  selected,
  dimmed = false,
  isNew = false,
  focusedSha = null,
  onFocusRef,
  searchQuery = '',
  conventional = false,
  hostLinks = null,
  changes,
  repoPath = '',
  ci,
  environments,
  referenceCards,
  style,
  onSelect,
  onContextMenu,
  onHighlightAuthor,
  authorHighlighted = false,
  onOpenChecks,
}: CommitRowProps) {
  useClockTick()
  const { branchColor } = useLaneColors()
  const prefix = (conventional && parsePrefix(commit.subject)?.prefix) || ''
  const pills = refs ? pairUpstreams(refs, upstreams) : []
  const classes = ['commit-row']
  if (selected) classes.push('commit-row-selected')
  if (isNew) classes.push('commit-row-new')

  return (
    <div
      className={classes.join(' ')}
      onClick={onSelect}
      onContextMenu={onContextMenu}
      style={{ ...style, opacity: dimmed ? 0.4 : 1 }}
      role="row"
      aria-selected={selected}
      aria-label={`${commit.subject}, ${shortSha(commit.sha)}, ${commit.authorName}`}
      tabIndex={0}
      onKeyDown={rowKeyDown(onSelect, onContextMenu)}
    >
      <div className="commit-row-graph" role="gridcell">
        {graphNode && (
          <CommitGraph node={graphNode} laneCount={laneCount} rowHeight={rowHeight} colors={laneColors} head={isHead} />
        )}
      </div>

      <div className="commit-row-commit" role="gridcell">
        {isHead && <span className="commit-row-head">HEAD</span>}
        {pills.length > 0 && (
          <span className="commit-row-refs">
            {pills.map(({ ref: r, upstream }) => (
              <RefBadge
                key={`${r.kind}-${r.name}`}
                refInfo={r}
                upstream={upstream}
                color={branchColor(withoutRemote(r), r.kind === 'branch' && r.name === currentBranch)}
                onClick={onFocusRef}
                active={focusedSha === r.sha}
                change={changes?.[upstream?.name ?? (r.kind === 'remote' ? r.name : '')]}
              />
            ))}
            {pills.length > 1 && (
              <span
                className="commit-row-refs-more"
                title={pills
                  .slice(1)
                  .map((p) => p.ref.name)
                  .join(', ')}
              >
                +{pills.length - 1}
              </span>
            )}
          </span>
        )}
        <span className="commit-row-subject">
          {prefix && <span className="commit-row-type">{highlightMatch(prefix, searchQuery, 'search-match')}</span>}
          <ReferenceText
            text={commit.subject.slice(prefix.length)}
            links={hostLinks}
            cards={referenceCards}
            render={(segment) => highlightMatch(segment, searchQuery, 'search-match')}
          />
        </span>
        <NoteMark kind="commit" target={commit.sha} />
        {environments && environments.length > 0 && <EnvironmentMarker environments={environments} />}
      </div>

      <div className="commit-row-author" role="gridcell">
        {identity && (
          <IdentityBadge
            identity={identity}
            name={commit.authorName}
            email={commit.authorEmail}
            size={20}
            onClick={onHighlightAuthor}
            pressed={authorHighlighted}
          />
        )}
        <span className="commit-row-author-name">{highlightMatch(commit.authorName, searchQuery, 'search-match')}</span>
      </div>

      <span className="commit-row-sha" role="gridcell">
        {highlightMatch(shortSha(commit.sha), searchQuery, 'search-match')}
      </span>
      <span className="commit-row-ci" role="gridcell">
        <CIMark repoPath={repoPath} mark={ci} onOpenJobs={onOpenChecks} />
      </span>
      <span className="commit-row-time" role="gridcell">
        {relativeTime(new Date(commit.date))}
      </span>
      <span className="commit-row-actions" role="gridcell">
        <button
          type="button"
          onClick={onContextMenu}
          aria-label={`More actions for ${shortSha(commit.sha)}`}
          title="More actions"
        >
          <Ellipsis size={16} strokeWidth={1.75} />
        </button>
      </span>
    </div>
  )
}

export default CommitRow
