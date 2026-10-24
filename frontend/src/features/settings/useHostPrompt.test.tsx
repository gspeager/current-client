import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { HostService, HostInfo } from '@current-client-bindings/app'
import { DialogContext, type Dialogs } from '../../lib/useDialogs'
import { useHostPrompt } from './useHostPrompt'

function withDialogs(confirm: Dialogs['confirm']) {
  const dialogs: Dialogs = { confirm, prompt: vi.fn(), signIn: vi.fn() }
  return ({ children }: { children: ReactNode }) => (
    <DialogContext.Provider value={dialogs}>{children}</DialogContext.Provider>
  )
}

const gitlab = new HostInfo({
  hostname: 'gitlab.internal.example',
  provider: 'gitlab',
  apiUrl: 'https://gitlab.internal.example/api/v4',
})

describe('useHostPrompt', () => {
  it('offers to enable a host it finds, naming it, and enables on Enable', async () => {
    vi.mocked(HostService.Detect).mockResolvedValue([gitlab])
    vi.mocked(HostService.SetEnabled).mockResolvedValue()
    const confirm = vi.fn().mockResolvedValue(true)
    const onChanged = vi.fn()

    renderHook(() => useHostPrompt('/repos/app', onChanged), { wrapper: withDialogs(confirm) })

    await waitFor(() => expect(HostService.SetEnabled).toHaveBeenCalledWith('gitlab.internal.example', true))
    expect(confirm).toHaveBeenCalledWith({
      title: 'Turn on GitLab integration?',
      message:
        'Turns on GitLab integration for gitlab.internal.example. Current Client calls its API when you open merge requests or refresh, and at no other time. Nothing is sent until then.',
      confirmLabel: 'Enable',
      cancelLabel: 'Not now',
    })
    expect(onChanged).toHaveBeenCalled()
  })

  it('remembers Not now', async () => {
    vi.mocked(HostService.Detect).mockResolvedValue([gitlab])
    vi.mocked(HostService.Dismiss).mockResolvedValue()
    const confirm = vi.fn().mockResolvedValue(false)

    renderHook(() => useHostPrompt('/repos/app', vi.fn()), { wrapper: withDialogs(confirm) })

    await waitFor(() => expect(HostService.Dismiss).toHaveBeenCalledWith('gitlab.internal.example'))
    expect(HostService.SetEnabled).not.toHaveBeenCalled()
  })

  it('asks nothing for a host already enabled or dismissed, or without a repository', async () => {
    vi.mocked(HostService.Detect).mockResolvedValue([
      new HostInfo({ ...gitlab, enabled: true }),
      new HostInfo({ ...gitlab, hostname: 'github.com', dismissed: true }),
    ])
    const confirm = vi.fn()

    renderHook(() => useHostPrompt('/repos/app', vi.fn()), { wrapper: withDialogs(confirm) })
    await waitFor(() => expect(HostService.Detect).toHaveBeenCalled())
    expect(confirm).not.toHaveBeenCalled()

    renderHook(() => useHostPrompt(null, vi.fn()), { wrapper: withDialogs(confirm) })
    expect(HostService.Detect).toHaveBeenCalledTimes(1)
  })
})
