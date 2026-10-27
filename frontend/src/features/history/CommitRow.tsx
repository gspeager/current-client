import type { CSSProperties, KeyboardEvent, MouseEvent } from 'react'
import { Ellipsis } from 'lucide-react'
import { isMenuKey } from '../../components/controls/ContextMenu'
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
import IdentityBadge from '../../components/git/IdentityBadge'
import ReferenceText from '../../components/git/ReferenceText'
import CIMark from '../../components/git/CIMark'
import RefBadge from './RefBadge'
import { EnvironmentMarker } from '../../components/git/EnvironmentMarker'
import { pairUpstreams } from './refPills'
import './CommitRow.scss'

interface CommitRowProps {
  commit: CommitInfo
  graphNode?: GraphNode
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
}

function CommitRow({
  commit,
  graphNode,
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
}: CommitRowProps) {
  useClockTick()
  const { laneColor } = useLaneColors()
  const prefix = (conventional && parsePrefix(commit.subject)?.prefix) || ''
  const classes = ['commit-row']
  if (selected) classes.push('commit-row-selected')
  if (isNew) classes.push('commit-row-new')

  return (
    <div
      className={classes.join(' ')}
      onClick={onSelect}
      onContextMenu={onContextMenu}
      style={{ ...style, opacity: dimmed ? 0.4 : 1 }}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (isMenuKey(e)) {
          onContextMenu(e)
          return
        }
        if (e.target !== e.currentTarget) return
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onSelect()
        }
      }}
    >
      <div className="commit-row-graph">
        {graphNode && <CommitGraph node={graphNode} laneCount={laneCount} rowHeight={rowHeight} />}
      </div>

      <div className="commit-row-commit">
        {refs && refs.length > 0 && (
          <span className="commit-row-refs">
            {pairUpstreams(refs, upstreams).map(({ ref: r, upstream }) => (
              <RefBadge
                key={`${r.kind}-${r.name}`}
                refInfo={r}
                upstream={upstream}
                color={laneColor(graphNode?.lane ?? 0)}
                onClick={onFocusRef}
                active={focusedSha === r.sha}
                change={changes?.[upstream?.name ?? (r.kind === 'remote' ? r.name : '')]}
              />
            ))}
          </span>
        )}
        {environments && environments.length > 0 && (
          <span className="commit-row-refs">
            {environments.map((env) => (
              <EnvironmentMarker key={env.environment} environment={env} />
            ))}
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
      </div>

      <div className="commit-row-author">
        {identity && (
          <IdentityBadge
            identity={identity}
            name={commit.authorName}
            email={commit.authorEmail}
            size={18}
            onClick={onHighlightAuthor}
            pressed={authorHighlighted}
          />
        )}
        <span className="commit-row-author-name">{highlightMatch(commit.authorName, searchQuery, 'search-match')}</span>
      </div>

      <span className="commit-row-sha">{highlightMatch(commit.sha.slice(0, 7), searchQuery, 'search-match')}</span>
      <span className="commit-row-ci">
        <CIMark repoPath={repoPath} mark={ci} />
      </span>
      <span className="commit-row-time">{relativeTime(new Date(commit.date))}</span>
      <span className="commit-row-actions">
        <button
          type="button"
          onClick={onContextMenu}
          aria-label={`More actions for ${commit.sha.slice(0, 7)}`}
          title="More actions"
        >
          <Ellipsis size={16} strokeWidth={1.75} />
        </button>
      </span>
    </div>
  )
}

export default CommitRow
