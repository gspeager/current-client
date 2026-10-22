import Tab from '../../components/controls/Tab'
import './TabBar.scss'

export type AppTab = 'working-copy' | 'history' | 'activity' | 'changelog' | 'changes'

interface TabBarProps {
  active: AppTab
  onChange: (tab: AppTab) => void
  showChangelog: boolean
  // "Merge requests" or "Pull requests" while a host is on for the repository; null hides the tab.
  changesLabel?: string | null
}

function TabBar({ active, onChange, showChangelog, changesLabel = null }: TabBarProps) {
  return (
    <nav className="tab-bar">
      <Tab active={active === 'working-copy'} onClick={() => onChange('working-copy')}>
        Working Copy
      </Tab>
      <Tab active={active === 'history'} onClick={() => onChange('history')}>
        Branch Graph &amp; History
      </Tab>
      <Tab active={active === 'activity'} onClick={() => onChange('activity')}>
        Activity
      </Tab>
      {showChangelog && (
        <Tab active={active === 'changelog'} onClick={() => onChange('changelog')}>
          Changelog
        </Tab>
      )}
      {changesLabel && (
        <Tab active={active === 'changes'} onClick={() => onChange('changes')}>
          {changesLabel}
        </Tab>
      )}
    </nav>
  )
}

export default TabBar
