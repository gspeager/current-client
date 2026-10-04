import { FolderGit2, GitBranch } from 'lucide-react'
import SegmentedControl from '../controls/SegmentedControl'
import BranchSwitcher from '../../features/branches/BranchSwitcher'
import type { useRepositoryLifecycle } from '../../features/repositories/useRepositoryLifecycle'
import RepoSwitcher from '../../features/repositories/RepoSwitcher'
import './BreadcrumbPanel.scss'

export type PanelView = 'repo' | 'branch'

interface BreadcrumbPanelProps {
  repoPath: string
  repo: ReturnType<typeof useRepositoryLifecycle>
  view: PanelView
  onViewChange: (view: PanelView) => void
  onBranchChanged?: () => void
  onFetched?: () => void
  pruneOnFetch?: boolean
}

function BreadcrumbPanel({
  repoPath,
  repo,
  view,
  onViewChange,
  onBranchChanged,
  onFetched,
  pruneOnFetch,
}: BreadcrumbPanelProps) {
  return (
    <div className="breadcrumb-panel">
      <SegmentedControl
        value={view}
        onChange={onViewChange}
        options={[
          { value: 'repo', label: 'Repository', icon: FolderGit2 },
          { value: 'branch', label: 'Branch', icon: GitBranch },
        ]}
      />
      {view === 'repo' ? (
        <RepoSwitcher repo={repo} />
      ) : (
        <BranchSwitcher
          repoPath={repoPath}
          onBranchChanged={onBranchChanged}
          onFetched={onFetched}
          pruneOnFetch={pruneOnFetch}
        />
      )}
    </div>
  )
}

export default BreadcrumbPanel
