import { render, screen, within } from '@testing-library/react'
import { DashboardService, type FileChurnInfo } from '@current-client-bindings/app'
import ActivityView from './ActivityView'

describe('ActivityView', () => {
  it('keeps showing the current figures while a ref move refetches them', async () => {
    let resolveRefetch: (churn: FileChurnInfo[]) => void = () => {}
    vi.mocked(DashboardService.GetFileChurn)
      .mockResolvedValueOnce([{ path: 'src/old.ts', changes: 3 }] as never)
      .mockReturnValueOnce(new Promise((resolve) => (resolveRefetch = resolve)) as never)

    const { container, rerender } = render(
      <ActivityView repoPath="/repos/app" repoVersion={0} dirty={false} hasConflict={false} />,
    )
    const churn = within(container.querySelector<HTMLElement>('.file-churn')!)
    expect(await churn.findByText('src/old.ts')).toBeInTheDocument()

    rerender(<ActivityView repoPath="/repos/app" repoVersion={1} dirty={false} hasConflict={false} />)
    expect(churn.getByText('src/old.ts')).toBeInTheDocument()
    expect(churn.queryByText('Loading…')).not.toBeInTheDocument()

    resolveRefetch([{ path: 'src/new.ts', changes: 5 }])
    expect(await screen.findByText('src/new.ts')).toBeInTheDocument()
  })

  it('shows no age for a repository without commits', async () => {
    vi.mocked(DashboardService.GetRepoStats).mockResolvedValue({
      totalCommits: 0,
      contributorCount: 0,
      fileCount: 0,
      repoAgeDays: 0,
      branchCount: 0,
      tagCount: 0,
      stashCount: 0,
      currentBranch: 'main',
    })
    vi.mocked(DashboardService.GetRepoDiskUsage).mockResolvedValue({
      looseObjectCount: 0,
      looseSizeKb: 0,
      packCount: 0,
      packedSizeKb: 0,
      totalSizeKb: 0,
    })
    const { container } = render(
      <ActivityView repoPath="/repos/app" repoVersion={0} dirty={false} hasConflict={false} />,
    )

    const strip = within(container.querySelector<HTMLElement>('.repo-stat-strip')!)
    expect((await strip.findByText('Repository age')).nextElementSibling).toHaveTextContent('—')
  })
})
