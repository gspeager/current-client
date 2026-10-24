import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RepositoryService, type RepoSummaryInfo } from '@current-client-bindings/app'
import type { useRepositoryLifecycle } from './useRepositoryLifecycle'
import RepoQuickList from './RepoQuickList'

function summary(path: string, name: string, available = true): RepoSummaryInfo {
  return { path, name, available, currentBranch: 'main', ahead: 0, behind: 0, dirty: 0, activity: [] }
}

describe('RepoQuickList', () => {
  it('lists recent repositories as rows, marks the current and open ones, and hands off to the dialog', async () => {
    vi.mocked(RepositoryService.GetRepoSummaries).mockResolvedValue([
      summary('/repos/app', 'app'),
      summary('/repos/site', 'site'),
      summary('/tmp/gone', 'gone', false),
    ])
    const repo = {
      repoPath: '/repos/app',
      recentRepos: ['/repos/app', '/repos/site', '/tmp/gone'],
      tabs: ['/repos/app', '/repos/site'],
      openRecent: vi.fn(),
    } as unknown as ReturnType<typeof useRepositoryLifecycle>
    const onOpenAnother = vi.fn()
    render(<RepoQuickList repo={repo} onOpenAnother={onOpenAnother} />)

    const app = await screen.findByRole('button', { name: /^app/ })
    expect(app).toHaveClass('repo-quick-row-current')
    expect(app).toHaveTextContent('open')
    expect(screen.getByRole('button', { name: /^gone/ })).toBeDisabled()
    expect(screen.queryByRole('textbox', { name: 'repository url' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /^site/ }))
    expect(repo.openRecent).toHaveBeenCalledWith('/repos/site')

    await userEvent.click(screen.getByRole('button', { name: 'Open another…' }))
    expect(onOpenAnother).toHaveBeenCalled()
  })
})
