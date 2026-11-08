import { render, screen } from '@testing-library/react'
import { ContributorsService, IdentityService } from '@current-client-bindings/app'
import ContributorsChart from './ContributorsChart'

describe('ContributorsChart', () => {
  it('shows a lone author at 100%', async () => {
    vi.mocked(ContributorsService.GetContributors).mockResolvedValue([
      { name: 'Ada', email: 'ada@example.com', commits: 138 },
    ] as never)
    vi.mocked(IdentityService.GetIdentities).mockResolvedValue([{ initials: 'A', color: '#d46ca4' }] as never)
    render(<ContributorsChart repoPath="/repos/app" repoVersion={0} />)

    expect(await screen.findByText(/138 commits · 100%/)).toBeInTheDocument()
  })
})
