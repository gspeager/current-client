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
    expect(marker).toHaveClass('change-marker-static')
    expect(marker.querySelector('.change-marker-ci')).toBeNull()
  })

  it('shows the cached facts in a hover card, and says what is unknown with the host off', async () => {
    render(
      <>
        <ChangeMarker
          change={
            new BranchChange({
              number: 138,
              prefix: '!',
              state: 'open',
              title: 'Add the thing',
              author: 'Ada',
              sourceBranch: 'feature',
              targetBranch: 'main',
              unresolvedThreads: 2,
              ci: 'failed',
            })
          }
        />
        <ChangeMarker change={new BranchChange({ number: 42, prefix: '#' })} />
      </>,
    )
    await userEvent.hover(screen.getByRole('button', { name: /^!138/ }))
    const card = await screen.findByRole('tooltip')
    expect(card).toHaveTextContent('Add the thing')
    expect(card).toHaveTextContent('by Ada')
    expect(card).toHaveTextContent('feature → main')
    expect(card).toHaveTextContent('2 unresolved')
    await userEvent.unhover(screen.getByRole('button', { name: /^!138/ }))

    await userEvent.hover(screen.getByRole('button', { name: '#42' }))
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Turn integration on for its title and state.')
  })
})
