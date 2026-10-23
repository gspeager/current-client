import { useState } from 'react'
import { Check, Copy, Search, SlidersHorizontal } from 'lucide-react'
import type { HistoryFilterInfo } from '@current-client-bindings/app'
import Checkbox from '../../components/forms/Checkbox'
import { BREAKING_FILTER, COMMIT_TYPES } from '../../lib/conventionalCommit'
import { sinceDaysAgo } from '../../lib/relativeTime'
import { useDismissibleDetails } from '../../lib/useDismissibleDetails'
import './HistoryToolbar.scss'

interface HistoryToolbarProps {
  search: string
  onSearchChange: (value: string) => void
  hideMerges: boolean
  onHideMergesChange: (value: boolean) => void
  branches: string[]
  filter: HistoryFilterInfo
  onFilterChange: (filter: HistoryFilterInfo) => void
  showTypeFilter: boolean
  onCopyAsciiGraph: () => void
  copied: boolean
}

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
  showTypeFilter,
  onCopyAsciiGraph,
  copied,
}: HistoryToolbarProps) {
  const [sinceDays, setSinceDays] = useState(0)
  const filtersRef = useDismissibleDetails()
  const active = [filter.since !== '', showTypeFilter && filter.type !== '', filter.author !== '', hideMerges].filter(
    Boolean,
  ).length

  return (
    <div className="history-toolbar">
      <div className="history-toolbar-search-wrap">
        <Search size={14} strokeWidth={1.5} className="history-toolbar-search-icon" />
        <input
          className="history-toolbar-search"
          type="search"
          placeholder="Search commits, authors, hashes…"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>
      <select
        className="history-toolbar-filter"
        value={filter.ref}
        onChange={(e) => onFilterChange({ ...filter, ref: e.target.value })}
        aria-label="Branch filter"
      >
        <option value="">Current branch</option>
        <option value="--all">All branches</option>
        {branches.map((name) => (
          <option key={name} value={name}>
            {name}
          </option>
        ))}
      </select>
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
              value={sinceDays}
              onChange={(e) => {
                const days = Number(e.target.value)
                setSinceDays(days)
                onFilterChange({ ...filter, since: sinceDaysAgo(days) })
              }}
              aria-label="Date filter"
            >
              <option value={0}>All time</option>
              <option value={1}>Today</option>
              <option value={7}>Past 7 days</option>
              <option value={30}>Past 30 days</option>
              <option value={90}>Past 90 days</option>
            </select>
          </label>
          {showTypeFilter && (
            <label className="history-toolbar-filters-row">
              <span>Type</span>
              <select
                className="history-toolbar-filter"
                value={filter.type}
                onChange={(e) => onFilterChange({ ...filter, type: e.target.value })}
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
              value={filter.author}
              onChange={(e) => onFilterChange({ ...filter, author: e.target.value })}
              aria-label="Author filter"
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
    </div>
  )
}

export default HistoryToolbar
