import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlatformService } from '@current-client-bindings/app'
import { Links } from '@current-client-bindings/core/hostlink'
import ReferenceText from './ReferenceText'
import { IssuePanelContext } from '../../lib/issuePanel'

const links = new Links({ issue: 'https://github.com/o/r/issues/', root: 'https://github.com/', issuePath: '/issues/' })

describe('ReferenceText', () => {
  it('opens a reference on the host without acting on the row underneath', async () => {
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    const onRowClick = vi.fn()
    render(
      <div role="button" onClick={onRowClick}>
        <ReferenceText text="Fix login (#12)" links={links} />
      </div>,
    )

    const link = screen.getByRole('link', { name: '#12' })
    expect(link).toHaveAttribute('href', 'https://github.com/o/r/issues/12')
    expect(screen.getByRole('button')).toHaveTextContent('Fix login (#12)')

    await userEvent.click(link)

    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://github.com/o/r/issues/12')
    expect(onRowClick).not.toHaveBeenCalled()
  })

  it('opens an issue’s panel while the host is on, and a merge request on the host', async () => {
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    const openIssue = vi.fn()
    render(
      <IssuePanelContext.Provider value={{ open: openIssue }}>
        <ReferenceText
          text="Fix #12 via !45"
          links={new Links({ ...links, mergeRequest: 'https://github.com/o/r/pull/' })}
        />
      </IssuePanelContext.Provider>,
    )
    await userEvent.click(screen.getByRole('link', { name: '#12' }))
    expect(openIssue).toHaveBeenCalledWith('12')
    expect(PlatformService.OpenWebPage).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('link', { name: '!45' }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://github.com/o/r/pull/45')
  })

  it('decorates linked and plain text alike', () => {
    render(<ReferenceText text="Fix #12" links={links} render={(s) => <mark>{s}</mark>} />)

    expect(screen.getByRole('link', { name: '#12' }).querySelector('mark')).toHaveTextContent('#12')
    expect(screen.getAllByText((_, el) => el?.tagName === 'MARK' && el.textContent === 'Fix ')).toHaveLength(1)
  })

  it('is plain text without a host', () => {
    render(<ReferenceText text="Fix #12" links={null} />)

    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.getByText('Fix #12')).toBeInTheDocument()
  })
})

describe('ReferenceText hover cards', () => {
  it('shows an issue title and state from the cards, and says when none was fetched', async () => {
    render(
      <ReferenceText
        text="Fixes #4 and #9"
        links={links}
        cards={{
          '#4': { title: 'Login fails', state: 'open', known: true },
          '#9': { title: '', state: '', known: false },
        }}
      />,
    )
    await userEvent.hover(screen.getByRole('link', { name: '#4' }))
    const card = await screen.findByRole('tooltip')
    expect(card).toHaveTextContent('Login fails')
    expect(card).toHaveTextContent('open')
    await userEvent.unhover(screen.getByRole('link', { name: '#4' }))

    await userEvent.hover(screen.getByRole('link', { name: '#9' }))
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Title not fetched')
  })
})
