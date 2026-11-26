import { arrowIntoList } from '../../lib/useRovingRows'
import { Check, Copy, ImageDown, Search, SlidersHorizontal } from 'lucide-react'
import type { HistoryFilterInfo } from '@current-client-bindings/app'
import Checkbox from '../../components/forms/Checkbox'
import { BREAKING_FILTER, COMMIT_TYPES } from '../../lib/conventionalCommit'
import { useDismissibleDetails } from '../../lib/useDismissibleDetails'
import { ALL_BRANCHES } from './branchFilter'
import type { FilterKey } from './typedFilters'
import './HistoryToolbar.scss'

interface HistoryToolbarProps {
  search: string
  onSearchChange: (value: string) => void
  hideMerges: boolean
  onHideMergesChange: (value: boolean) => void
  branches: string[]
  filter: HistoryFilterInfo
  onFilterChange: (filter: HistoryFilterInfo) => void
  // As typed in the box, which the menu writes to.
  typed: Record<FilterKey, string>
  onTypedChange: (key: FilterKey, value: string) => void
  showTypeFilter: boolean
  onCopyAsciiGraph: () => void
  // Not given while there's nothing to save.
  onSaveImage?: () => void
  copied: boolean
}

const SINCE_PRESETS = ['today', '7d', '30d', '90d']

// Search and the branch select stay in the bar at every width; the rest of the
// filters live in one menu, whose badge counts the ones in effect.
function HistoryToolbar({
  search,
  onSearchChange,
  hideMerges,
  onHideMergesChange,
  branches,
  filter,
  onFilterChange,
  typed,
  onTypedChange,
  showTypeFilter,
  onCopyAsciiGraph,
  onSaveImage,
  copied,
}: HistoryToolbarProps) {
  const filtersRef = useDismissibleDetails()
  const active = [
    typed.since !== '',
    typed.until !== '',
    showTypeFilter && typed.type !== '',
    typed.author !== '',
    hideMerges,
  ].filter(Boolean).length
  const since = SINCE_PRESETS.includes(typed.since.toLowerCase()) ? typed.since.toLowerCase() : typed.since && 'custom'

  return (
    <div className="history-toolbar">
      <div className="history-toolbar-search-wrap">
        <Search size={14} strokeWidth={1.5} className="history-toolbar-search-icon" />
        <input
          className="history-toolbar-search"
          type="search"
          placeholder="Search commits…"
          title="Search by subject, author or hash. Filter with author:, type:, since: and until:, such as author:maya since:7d"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          onKeyDown={arrowIntoList}
        />
      </div>
      <select
        className="history-toolbar-filter"
        value={filter.ref}
        onChange={(e) => onFilterChange({ ...filter, ref: e.target.value })}
        aria-label="Branch filter"
      >
        <option value="">Current branch</option>
        <option value={ALL_BRANCHES}>All branches</option>
        {branches.map((name) => (
          <option key={name} value={name}>
            {name}
          </option>
        ))}
      </select>
      <span className="history-toolbar-first-parent">
        <Checkbox
          checked={filter.firstParent}
          onChange={(on) => onFilterChange({ ...filter, firstParent: on })}
          label="First parent only"
        />
      </span>
      <details ref={filtersRef} className="history-toolbar-filters">
        <summary className="history-toolbar-filters-toggle" aria-label={active ? `Filters, ${active} on` : 'Filters'}>
          <SlidersHorizontal size={14} strokeWidth={1.5} />
          Filters
          {active > 0 && <span className="history-toolbar-filters-count">{active}</span>}
        </summary>
        <div className="history-toolbar-filters-menu">
          <label className="history-toolbar-filters-row">
            <span>Time</span>
            <select
              className="history-toolbar-filter"
              value={since}
              onChange={(e) => onTypedChange('since', e.target.value)}
              aria-label="Date filter"
            >
              <option value="">All time</option>
              <option value="today">Today</option>
              <option value="7d">Past 7 days</option>
              <option value="30d">Past 30 days</option>
              <option value="90d">Past 90 days</option>
              {since === 'custom' && <option value="custom">Since {typed.since}</option>}
            </select>
          </label>
          {showTypeFilter && (
            <label className="history-toolbar-filters-row">
              <span>Type</span>
              <select
                className="history-toolbar-filter"
                value={filter.type}
                onChange={(e) =>
                  onTypedChange('type', e.target.value === BREAKING_FILTER ? 'breaking' : e.target.value)
                }
                aria-label="Type filter"
              >
                <option value="">All types</option>
                <option value={BREAKING_FILTER}>Breaking changes</option>
                {COMMIT_TYPES.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
            </label>
          )}
          <label className="history-toolbar-filters-row">
            <span>Author</span>
            <input
              className="history-toolbar-author-input"
              type="text"
              placeholder="Any author"
              value={typed.author}
              onChange={(e) => onTypedChange('author', e.target.value)}
              aria-label="Author filter"
            />
          </label>
          <label className="history-toolbar-filters-row">
            <span>Until</span>
            <input
              className="history-toolbar-author-input"
              type="text"
              placeholder="Any date"
              value={typed.until}
              onChange={(e) => onTypedChange('until', e.target.value)}
              aria-label="Until filter"
            />
          </label>
          <div className="history-toolbar-filters-row">
            <span>Merges</span>
            <Checkbox checked={hideMerges} onChange={onHideMergesChange} label="Hide merges" />
          </div>
        </div>
      </details>
      <button
        type="button"
        className="history-toolbar-copy-graph"
        onClick={onCopyAsciiGraph}
        aria-label="Copy as ASCII graph"
        title="Copy as ASCII graph"
      >
        {copied ? <Check size={14} strokeWidth={1.75} /> : <Copy size={14} strokeWidth={1.5} />}
      </button>
      <button
        type="button"
        className="history-toolbar-copy-graph"
        disabled={!onSaveImage}
        onClick={onSaveImage}
        aria-label="Save as image"
        title="Save as image"
      >
        <ImageDown size={14} strokeWidth={1.5} />
      </button>
    </div>
  )
}

export default HistoryToolbar
