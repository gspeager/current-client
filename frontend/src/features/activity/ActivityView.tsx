import ActivityCalendar from './ActivityCalendar'
import CommitVelocityChart from './CommitVelocityChart'
import ContributorsChart from './ContributorsChart'
import FileChurnList from './FileChurnList'
import LanguageBreakdown from './LanguageBreakdown'
import RepositoryHealth from './RepositoryHealth'
import RepoStatStrip from './RepoStatStrip'
import './ActivityView.scss'

interface ActivityViewProps {
  repoPath: string
  repoVersion: number
  dirty: boolean
  hasConflict: boolean
}

function ActivityView({ repoPath, repoVersion, dirty, hasConflict }: ActivityViewProps) {
  return (
    <div className="activity-view">
      <RepoStatStrip repoPath={repoPath} repoVersion={repoVersion} dirty={dirty} />
      <ActivityCalendar repoPath={repoPath} repoVersion={repoVersion} />
      <div className="activity-view-pair">
        <ContributorsChart repoPath={repoPath} repoVersion={repoVersion} />
        <CommitVelocityChart repoPath={repoPath} repoVersion={repoVersion} />
        <RepositoryHealth repoPath={repoPath} repoVersion={repoVersion} dirty={dirty} hasConflict={hasConflict} />
      </div>
      <div className="activity-view-pair">
        <FileChurnList repoPath={repoPath} repoVersion={repoVersion} />
        <LanguageBreakdown repoPath={repoPath} repoVersion={repoVersion} />
      </div>
    </div>
  )
}

export default ActivityView
