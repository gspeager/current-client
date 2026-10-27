import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlatformService, type EnvironmentInfo } from '@current-client-bindings/app'
import { EnvironmentChips, EnvironmentMarker, EnvironmentReach } from './EnvironmentMarker'

const production = {
  environment: 'production',
  sha: 'abc',
  ref: 'main',
  status: 'success',
  deployedAt: '2026-10-26T10:00:00Z',
  deployedBy: { name: 'Ada', username: 'ada' },
  webUrl: 'https://gitlab.example.com/g/r/-/environments/1',
  production: true,
  fetched: true,
} as EnvironmentInfo

describe('environments', () => {
  it('marks the deployed commit and opens the environment on the host', async () => {
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    render(<EnvironmentMarker environment={production} />)
    const marker = screen.getByRole('button', { name: 'production is deployed here' })
    await userEvent.hover(marker)
    expect(await screen.findByText(/from main · .* · by Ada/)).toBeInTheDocument()
    await userEvent.click(marker)
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.example.com/g/r/-/environments/1')
  })

  it('says where a commit has reached, and unknown rather than no when Git can’t tell', () => {
    render(
      <EnvironmentReach
        reach={[
          { environment: 'staging', state: 'in', production: false },
          { environment: 'production', state: 'out', production: true },
          { environment: 'review', state: 'unknown', production: false },
        ]}
      />,
    )
    const list = screen.getByRole('list', { name: 'Environments' })
    expect(list).toHaveTextContent('stagingReached')
    expect(list).toHaveTextContent('productionNot yet')
    expect(list).toHaveTextContent('reviewUnknown: the deployed commit isn’t fetched')
  })

  it('shows a branch tip only the environments it is in', () => {
    render(
      <EnvironmentChips
        reach={[
          { environment: 'staging', state: 'in', production: false },
          { environment: 'production', state: 'unknown', production: true },
        ]}
      />,
    )
    expect(screen.getByTitle('In staging')).toBeInTheDocument()
    expect(screen.queryByTitle('In production')).not.toBeInTheDocument()
  })
})
