import { render, screen } from '@testing-library/react'
import { ContributorsService, IdentityService } from '@current-client-bindings/app'
import ContributorsChart from './ContributorsChart'

describe('ContributorsChart', () => {
  it('draws the donut for a single author, as for many', async () => {
    vi.mocked(ContributorsService.GetContributors).mockResolvedValue([
      { name: 'Ada', email: 'ada@example.com', commits: 138 },
    ] as never)
    vi.mocked(IdentityService.GetIdentities).mockResolvedValue([{ initials: 'A', color: '#d46ca4' }] as never)
    render(<ContributorsChart repoPath="/repos/app" repoVersion={0} />)

    expect(await screen.findByText('100%')).toBeInTheDocument()
    expect(screen.getByText('1 AUTHOR')).toBeInTheDocument()
    expect(document.querySelectorAll('.contributors-donut circle')).toHaveLength(1)
    expect(screen.getByText('138 commits')).toBeInTheDocument()
  })
})
