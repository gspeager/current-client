import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlatformService } from '@current-client-bindings/app'
import { Links } from '@current-client-bindings/core/hostlink'
import ReferenceText from './ReferenceText'

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
