import { render, screen } from '@testing-library/react'
import { BranchService, DashboardService, type BranchStatusInfo } from '@current-client-bindings/app'
import RepositoryHealth from './RepositoryHealth'

function stubRepository() {
  vi.mocked(BranchService.ListLocal).mockResolvedValue([])
  vi.mocked(DashboardService.GetRepoDiskUsage).mockResolvedValue({ looseObjectCount: 0 } as never)
  vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({
    current: 'main',
    upstream: 'origin/main',
    ahead: 0,
  } as BranchStatusInfo)
}

describe('RepositoryHealth', () => {
  it('shows uncommitted changes without counting them as an issue', async () => {
    stubRepository()
    render(<RepositoryHealth repoPath="/repos/app" repoVersion={0} dirty hasConflict={false} />)

    expect(await screen.findByText('Optimal')).toBeInTheDocument()
    expect(screen.getByText('Uncommitted changes present')).toBeInTheDocument()
  })

  it('counts a conflict as an issue', async () => {
    stubRepository()
    render(<RepositoryHealth repoPath="/repos/app" repoVersion={0} dirty hasConflict />)

    expect(await screen.findByText('1 issue')).toBeInTheDocument()
  })
})
