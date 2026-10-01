import type { CSSProperties, MouseEvent } from 'react'
import { parsePrefix } from '../../lib/conventionalCommit'
import { highlightMatch } from '../../lib/highlightMatch'
import { useLaneColors } from '../../lib/laneColor'
import { relativeTime } from '../../lib/relativeTime'
import type { CommitInfo, GraphNode, IdentityInfo, RefInfo } from '@current-client-bindings/app'
import CommitGraph from './CommitGraph'
import IdentityBadge from '../../components/git/IdentityBadge'
import RefBadge from './RefBadge'
import './CommitRow.scss'

interface CommitRowProps {
  commit: CommitInfo
  graphNode?: GraphNode
  hasAbove: boolean
  laneCount: number
  rowHeight: number
  identity?: IdentityInfo
  refs?: RefInfo[]
  selected: boolean
  dimmed?: boolean
  isNew?: boolean
  focusedSha?: string | null
  onFocusRef?: (sha: string) => void
  searchQuery?: string
  conventional?: boolean
  style: CSSProperties
  onSelect: () => void
  onContextMenu: (e: MouseEvent) => void
}

function CommitRow({
  commit,
  graphNode,
  hasAbove,
  laneCount,
  rowHeight,
  identity,
  refs,
  selected,
  dimmed = false,
  isNew = false,
  focusedSha = null,
  onFocusRef,
  searchQuery = '',
  conventional = false,
  style,
  onSelect,
  onContextMenu,
}: CommitRowProps) {
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
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onSelect()
        }
      }}
    >
      <div className="commit-row-graph">
        {graphNode && <CommitGraph node={graphNode} hasAbove={hasAbove} laneCount={laneCount} rowHeight={rowHeight} />}
      </div>

      <div className="commit-row-commit">
        {refs?.map((r) => (
          <RefBadge
            key={`${r.kind}-${r.name}`}
            refInfo={r}
            color={laneColor(graphNode?.lane ?? 0)}
            onClick={onFocusRef}
            active={focusedSha === r.sha}
          />
        ))}
        <span className="commit-row-subject">
          {prefix && <span className="commit-row-type">{highlightMatch(prefix, searchQuery, 'search-match')}</span>}
          {highlightMatch(commit.subject.slice(prefix.length), searchQuery, 'search-match')}
        </span>
      </div>

      <div className="commit-row-author">
        {identity && (
          <IdentityBadge identity={identity} name={commit.authorName} email={commit.authorEmail} size={18} />
        )}
        <span className="commit-row-author-name">{highlightMatch(commit.authorName, searchQuery, 'search-match')}</span>
      </div>

      <div className="commit-row-hash">
        <span className="commit-row-sha">{highlightMatch(commit.sha.slice(0, 7), searchQuery, 'search-match')}</span>
        <span className="commit-row-time">{relativeTime(new Date(commit.date))}</span>
      </div>
    </div>
  )
}

export default CommitRow
