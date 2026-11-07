import { renderHook, waitFor } from '@testing-library/react'
import { HostService } from '@current-client-bindings/app'
import { useBranchChanges } from './useBranchChanges'

describe('useBranchChanges', () => {
  it('asks nothing without a repository', () => {
    const { result } = renderHook(() => useBranchChanges(''))

    expect(HostService.BranchChanges).not.toHaveBeenCalled()
    expect(result.current).toEqual({})
  })

  it('loads the changes of an open repository', async () => {
    vi.mocked(HostService.BranchChanges).mockResolvedValue({})

    renderHook(() => useBranchChanges('/repos/app'))

    await waitFor(() => expect(HostService.BranchChanges).toHaveBeenCalledWith('/repos/app'))
  })
})
