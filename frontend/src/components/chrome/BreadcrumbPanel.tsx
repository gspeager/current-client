import { FolderGit2, GitBranch } from 'lucide-react'
import SegmentedControl from '../controls/SegmentedControl'
import BranchSwitcher from '../../features/branches/BranchSwitcher'
import type { useRepositoryLifecycle } from '../../features/repositories/useRepositoryLifecycle'
import RepoQuickList from '../../features/repositories/RepoQuickList'
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
  onOpenAnother: () => void
}

function BreadcrumbPanel({
  repoPath,
  repo,
  view,
  onViewChange,
  onBranchChanged,
  onFetched,
  pruneOnFetch,
  onOpenAnother,
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
        <RepoQuickList repo={repo} onOpenAnother={onOpenAnother} />
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
