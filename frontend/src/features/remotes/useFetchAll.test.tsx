import type { ReactNode } from 'react'
import { act, renderHook } from '@testing-library/react'
import { RemoteService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { useFetchAll } from './useFetchAll'

const REPO = '/repos/app'

function wrapper({ children }: { children: ReactNode }) {
  return <DialogProvider>{children}</DialogProvider>
}

describe('useFetchAll', () => {
  it('fetches without pruning by default', async () => {
    vi.mocked(RemoteService.FetchAll).mockResolvedValue()
    const onFetched = vi.fn()
    const { result } = renderHook(() => useFetchAll(REPO, onFetched), { wrapper })

    act(() => result.current.fetchAll())
    await vi.waitFor(() => expect(onFetched).toHaveBeenCalled())

    expect(RemoteService.FetchAll).toHaveBeenCalledWith(REPO, null)
    expect(RemoteService.FetchAllPrune).not.toHaveBeenCalled()
  })

  it('prunes gone remote branches when asked', async () => {
    vi.mocked(RemoteService.FetchAllPrune).mockResolvedValue()
    const onFetched = vi.fn()
    const { result } = renderHook(() => useFetchAll(REPO, onFetched, true), { wrapper })

    act(() => result.current.fetchAll())
    await vi.waitFor(() => expect(onFetched).toHaveBeenCalled())

    expect(RemoteService.FetchAllPrune).toHaveBeenCalledWith(REPO, null)
    expect(RemoteService.FetchAll).not.toHaveBeenCalled()
  })
})

describe('useFetchAll primary remote', () => {
  it('fetches origin, pruning only when the setting says so', async () => {
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'upstream' }, { name: 'origin' }] as never)
    vi.mocked(RemoteService.Fetch).mockResolvedValue()
    const onFetched = vi.fn()
    const { result } = renderHook(() => useFetchAll(REPO, onFetched), { wrapper })

    act(() => result.current.fetchPrimary())
    await vi.waitFor(() => expect(onFetched).toHaveBeenCalled())

    expect(RemoteService.Fetch).toHaveBeenCalledWith(REPO, 'origin', null)
    expect(RemoteService.FetchPrune).not.toHaveBeenCalled()
  })

  it('prunes once from the menu whatever the setting says', async () => {
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'origin' }] as never)
    vi.mocked(RemoteService.FetchPrune).mockResolvedValue()
    const onFetched = vi.fn()
    const { result } = renderHook(() => useFetchAll(REPO, onFetched), { wrapper })

    act(() => result.current.fetchPrune())
    await vi.waitFor(() => expect(onFetched).toHaveBeenCalled())

    expect(RemoteService.FetchPrune).toHaveBeenCalledWith(REPO, 'origin', null)
  })

  it('says when there is no remote to fetch', async () => {
    vi.mocked(RemoteService.List).mockResolvedValue([])
    const { result } = renderHook(() => useFetchAll(REPO), { wrapper })

    act(() => result.current.fetchPrimary())

    await vi.waitFor(() => expect(result.current.error).toBe('No remote configured to push to.'))
  })
})
