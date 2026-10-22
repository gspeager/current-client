import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlatformService } from '@current-client-bindings/app'
import HostMarkdown from './HostMarkdown'

describe('HostMarkdown', () => {
  it('opens http(s) links in the browser and nothing else', async () => {
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    render(
      <HostMarkdown
        html={
          '<p><a href="https://example.com/page">open</a> <a href="javascript:alert(1)">run</a> <a href="">none</a></p>'
        }
      />,
    )

    await userEvent.click(screen.getByText('open'))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://example.com/page')
    await userEvent.click(screen.getByText('run'))
    await userEvent.click(screen.getByText('none'))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledTimes(1)
  })

  it('shows the markup without running script in it', () => {
    const ran = vi.fn()
    Object.assign(window, { hostMarkdownRan: ran })
    render(<HostMarkdown html={'<p>text</p><script>window.hostMarkdownRan()</script>'} />)

    expect(screen.getByText('text')).toBeInTheDocument()
    expect(ran).not.toHaveBeenCalled()
  })
})
