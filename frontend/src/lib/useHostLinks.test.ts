import { renderHook, waitFor } from '@testing-library/react'
import { RemoteService } from '@current-client-bindings/app'
import { useHostLinks } from './useHostLinks'

describe('useHostLinks', () => {
  it('asks nothing without a repository', () => {
    const { result } = renderHook(() => useHostLinks(''))

    expect(RemoteService.HostLinks).not.toHaveBeenCalled()
    expect(result.current).toBeNull()
  })

  it('loads the links of an open repository', async () => {
    vi.mocked(RemoteService.HostLinks).mockResolvedValue(null)

    renderHook(() => useHostLinks('/repos/app'))

    await waitFor(() => expect(RemoteService.HostLinks).toHaveBeenCalledWith('/repos/app'))
  })
})
