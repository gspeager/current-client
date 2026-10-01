import Tab from '../../components/controls/Tab'
import './TabBar.scss'

export type AppTab = 'working-copy' | 'history' | 'activity' | 'changelog'

interface TabBarProps {
  active: AppTab
  onChange: (tab: AppTab) => void
  showChangelog: boolean
}

function TabBar({ active, onChange, showChangelog }: TabBarProps) {
  return (
    <nav className="tab-bar">
      <Tab active={active === 'activity'} onClick={() => onChange('activity')}>
        Activity
      </Tab>
      <Tab active={active === 'working-copy'} onClick={() => onChange('working-copy')}>
        Working Copy
      </Tab>
      <Tab active={active === 'history'} onClick={() => onChange('history')}>
        Branch Graph &amp; History
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
