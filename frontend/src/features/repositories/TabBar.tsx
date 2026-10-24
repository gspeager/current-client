import Tab from '../../components/controls/Tab'
import { plural } from '../../lib/plural'
import './TabBar.scss'

export type AppTab = 'working-copy' | 'history' | 'activity' | 'changelog' | 'changes'

interface TabBarProps {
  active: AppTab
  onChange: (tab: AppTab) => void
  showChangelog: boolean
  // "Merge requests" or "Pull requests" while a host is on for the repository; null hides the tab.
  changesLabel?: string | null
  // Changed files in the working tree, shown on the Changes tab.
  changedCount?: number
  // Files still conflicted in a merge; shown instead of the changed count.
  conflictCount?: number
}

// Core views first, the host's next, reports last.
function TabBar({
  active,
  onChange,
  showChangelog,
  changesLabel = null,
  changedCount = 0,
  conflictCount = 0,
}: TabBarProps) {
  return (
    <nav className="tab-bar">
      <Tab
        active={active === 'working-copy'}
        onClick={() => onChange('working-copy')}
        label={
          conflictCount > 0
            ? `Changes, ${plural(conflictCount, 'conflicted file')}`
            : changedCount > 0
              ? `Changes, ${plural(changedCount, 'changed file')}`
              : 'Changes'
        }
      >
        Changes
        {conflictCount > 0 ? (
          <span className="tab-count tab-count-conflict">{conflictCount}</span>
        ) : (
          changedCount > 0 && <span className="tab-count">{changedCount}</span>
        )}
      </Tab>
      <Tab active={active === 'history'} onClick={() => onChange('history')}>
        History
      </Tab>
      {changesLabel && (
        <Tab active={active === 'changes'} onClick={() => onChange('changes')}>
          {changesLabel}
        </Tab>
      )}
      <span className="tab-bar-divider" aria-hidden="true" />
      <Tab active={active === 'activity'} onClick={() => onChange('activity')}>
        Activity
      </Tab>
      {showChangelog && (
        <Tab active={active === 'changelog'} onClick={() => onChange('changelog')}>
          Changelog
        </Tab>
      )}
    </nav>
  )
}

export default TabBar
