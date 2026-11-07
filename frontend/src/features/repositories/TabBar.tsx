import { ChartColumn, CircleDot, FileDiff, GitCommitVertical, GitPullRequest, Tag } from 'lucide-react'
import Tab from '../../components/controls/Tab'
import { useRovingRows } from '../../lib/useRovingRows'
import { plural } from '../../lib/plural'
import './TabBar.scss'

export type AppTab = 'working-copy' | 'history' | 'activity' | 'changelog' | 'changes' | 'issues'

interface TabBarProps {
  active: AppTab
  onChange: (tab: AppTab) => void
  // "Merge requests" or "Pull requests" while a host is on for the repository; null hides the tab.
  changesLabel?: string | null
  // Changed files in the working tree, shown on the Changes tab.
  changedCount?: number
  // Files still conflicted in a merge; shown instead of the changed count.
  conflictCount?: number
  // Issues assigned to you while a host or tracker lists them; null hides the tab.
  issueCount?: number | null
  // A host a remote points at whose integration is off (and not locked off):
  // its tabs show dimmed and open its settings, so the features can be found.
  offHost?: { hostname: string; changesLabel: string } | null
  onOffHost?: () => void
}

// Core views first, the host's next, reports last.
function TabBar({
  active,
  onChange,
  changesLabel = null,
  changedCount = 0,
  conflictCount = 0,
  issueCount = null,
  offHost = null,
  onOffHost,
}: TabBarProps) {
  const rovingTabs = useRovingRows('.tab', { horizontal: true })
  const hostTabs = (changesLabel ? 1 : 0) + (issueCount === null ? 0 : 1)
  const offTitle = offHost ? `${offHost.hostname} is off · Integration settings` : undefined
  return (
    <nav ref={rovingTabs} className="tab-bar" aria-label="Views" data-pane="views">
      <Tab
        active={active === 'working-copy'}
        onClick={() => onChange('working-copy')}
        shortcut="mod+1"
        label={
          conflictCount > 0
            ? `Changes, ${plural(conflictCount, 'conflicted file')}`
            : changedCount > 0
              ? `Changes, ${plural(changedCount, 'changed file')}`
              : 'Changes'
        }
      >
        <FileDiff size={14} strokeWidth={1.75} aria-hidden="true" />
        Changes
        {conflictCount > 0 ? (
          <span className="tab-count tab-count-conflict">{conflictCount}</span>
        ) : (
          changedCount > 0 && <span className="tab-count">{changedCount}</span>
        )}
      </Tab>
      <Tab active={active === 'history'} onClick={() => onChange('history')} shortcut="mod+2">
        <GitCommitVertical size={14} strokeWidth={1.75} aria-hidden="true" />
        History
      </Tab>
      {changesLabel && (
        <Tab active={active === 'changes'} onClick={() => onChange('changes')} shortcut="mod+3">
          <GitPullRequest size={14} strokeWidth={1.75} aria-hidden="true" />
          {changesLabel}
        </Tab>
      )}
      {offHost && !changesLabel && onOffHost && (
        <>
          <Tab active={false} onClick={onOffHost} label={`${offHost.changesLabel} off`} off={offTitle}>
            {offHost.changesLabel}
          </Tab>
          {issueCount === null && (
            <Tab active={false} onClick={onOffHost} label="Issues off" off={offTitle}>
              Issues
            </Tab>
          )}
        </>
      )}
      {issueCount !== null && (
        <Tab
          active={active === 'issues'}
          onClick={() => onChange('issues')}
          shortcut={changesLabel ? 'mod+4' : 'mod+3'}
          label={issueCount > 0 ? `Issues, ${plural(issueCount, 'assigned', 'assigned')}` : 'Issues'}
        >
          <CircleDot size={14} strokeWidth={1.75} aria-hidden="true" />
          Issues
          <span className="tab-count" style={issueCount > 0 ? undefined : { visibility: 'hidden' }}>
            {issueCount}
          </span>
        </Tab>
      )}
      <span className="tab-bar-divider" aria-hidden="true" />
      <Tab active={active === 'activity'} onClick={() => onChange('activity')} shortcut={`mod+${3 + hostTabs}`}>
        <ChartColumn size={14} strokeWidth={1.75} aria-hidden="true" />
        Activity
      </Tab>
      <Tab active={active === 'changelog'} onClick={() => onChange('changelog')} shortcut={`mod+${4 + hostTabs}`}>
        <Tag size={14} strokeWidth={1.75} aria-hidden="true" />
        Releases
      </Tab>
    </nav>
  )
}

export default TabBar
