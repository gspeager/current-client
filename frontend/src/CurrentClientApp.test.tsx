import { render, waitFor } from '@testing-library/react'
import { PlatformService, SettingsService, type Settings } from '@current-client-bindings/app'
import CurrentClientApp from './CurrentClientApp'

function rootOf(container: HTMLElement): HTMLElement {
  return container.querySelector('.current-client') as HTMLElement
}

describe('CurrentClientApp', () => {
  it('puts the theme on its own root rather than the page', async () => {
    vi.mocked(SettingsService.GetSettings).mockResolvedValue({ theme: 'light' } as Settings)
    const { container } = render(<CurrentClientApp />)

    await waitFor(() => expect(rootOf(container)).toHaveAttribute('data-theme', 'light'))
    // Native scrollbars and form controls follow color-scheme.
    expect(getComputedStyle(rootOf(container)).colorScheme).toBe('light')
    expect(document.documentElement).not.toHaveAttribute('data-theme')
  })

  it('matches the window title bar to the theme', async () => {
    vi.mocked(SettingsService.GetSettings).mockResolvedValue({ theme: 'light' } as Settings)
    vi.mocked(PlatformService.SetDarkTitleBar).mockResolvedValue()
    render(<CurrentClientApp />)

    await waitFor(() => expect(PlatformService.SetDarkTitleBar).toHaveBeenLastCalledWith(false))
  })

  it('leaves the title bar to the host while inactive', async () => {
    vi.mocked(SettingsService.GetSettings).mockResolvedValue({ theme: 'light' } as Settings)
    const { container } = render(<CurrentClientApp active={false} />)

    await waitFor(() => expect(rootOf(container)).toHaveAttribute('data-theme', 'light'))
    expect(PlatformService.SetDarkTitleBar).not.toHaveBeenCalled()
  })

  it('stays mounted but hidden while inactive', async () => {
    vi.mocked(SettingsService.GetSettings).mockResolvedValue({ theme: 'dark' } as Settings)
    const { container, rerender } = render(<CurrentClientApp />)
    await waitFor(() => expect(SettingsService.GetSettings).toHaveBeenCalled())

    rerender(<CurrentClientApp active={false} />)

    expect(rootOf(container)).not.toBeVisible()
    expect(rootOf(container).childElementCount).toBeGreaterThan(0)
  })
})
