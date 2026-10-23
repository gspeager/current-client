import Tab from '../../components/controls/Tab'
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
}

// Core views first, the host's next, reports last.
function TabBar({ active, onChange, showChangelog, changesLabel = null, changedCount = 0 }: TabBarProps) {
  return (
    <nav className="tab-bar">
      <Tab active={active === 'working-copy'} onClick={() => onChange('working-copy')} label="Changes">
        Changes
        {changedCount > 0 && <span className="tab-count">{changedCount}</span>}
      </Tab>
      <Tab active={active === 'history'} onClick={() => onChange('history')}>
        History
      </Tab>
      {changesLabel && (
        <Tab active={active === 'changes'} onClick={() => onChange('changes')}>
          {changesLabel}
        </Tab>
      )}
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
