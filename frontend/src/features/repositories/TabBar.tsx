import Tab from '../../components/controls/Tab'
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
      {offHost && !changesLabel && onOffHost && (
        <>
          <Tab
            active={false}
            onClick={onOffHost}
            label={`${offHost.changesLabel} off`}
            off={`${offHost.hostname} is off · Integration settings`}
          >
            {offHost.changesLabel}
          </Tab>
          {issueCount === null && (
            <Tab
              active={false}
              onClick={onOffHost}
              label="Issues off"
              off={`${offHost.hostname} is off · Integration settings`}
            >
              Issues
            </Tab>
          )}
        </>
      )}
      {issueCount !== null && (
        <Tab
          active={active === 'issues'}
          onClick={() => onChange('issues')}
          label={issueCount > 0 ? `Issues, ${plural(issueCount, 'assigned to you', 'assigned to you')}` : 'Issues'}
        >
          Issues
          {issueCount > 0 && <span className="tab-count">{issueCount}</span>}
        </Tab>
      )}
      <span className="tab-bar-divider" aria-hidden="true" />
      <Tab active={active === 'activity'} onClick={() => onChange('activity')}>
        Activity
      </Tab>
      <Tab active={active === 'changelog'} onClick={() => onChange('changelog')}>
        Releases
      </Tab>
    </nav>
  )
}

export default TabBar
