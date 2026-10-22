import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchChange, PlatformService } from '@current-client-bindings/app'
import ChangeMarker from './ChangeMarker'

describe('ChangeMarker', () => {
  it('shows number, state and CI, and opens the change on the host', async () => {
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    render(
      <ChangeMarker
        change={
          new BranchChange({
            number: 138,
            prefix: '!',
            state: 'open',
            draft: true,
            ci: 'failed',
            title: 'Add the thing',
            webUrl: 'https://gitlab.example.com/g/r/-/merge_requests/138',
          })
        }
      />,
    )
    const marker = screen.getByRole('button', { name: '!138 · Add the thing · draft · CI failed' })
    expect(marker).toHaveAttribute('data-state', 'draft')
    expect(marker.querySelector('.change-marker-ci-failed')).not.toBeNull()
    await userEvent.click(marker)
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.example.com/g/r/-/merge_requests/138')
  })

  it('shows the number alone when only Git supplied it', () => {
    render(<ChangeMarker change={new BranchChange({ number: 42, prefix: '#' })} />)
    const marker = screen.getByRole('button', { name: '#42' })
    expect(marker).toHaveAttribute('data-state', 'unknown')
    expect(marker).toBeDisabled()
    expect(marker.querySelector('.change-marker-ci')).toBeNull()
  })
})
