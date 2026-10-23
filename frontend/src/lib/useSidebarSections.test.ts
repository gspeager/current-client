import { act, renderHook } from '@testing-library/react'
import { SettingsService } from '@current-client-bindings/app'
import { useSidebarSections } from './useSidebarSections'

describe('useSidebarSections', () => {
  it('follows the default until a choice is stored for the repository', async () => {
    vi.mocked(SettingsService.GetSidebarSections).mockResolvedValue({ tags: false })
    vi.mocked(SettingsService.SetSidebarSection).mockResolvedValue()
    const { result } = renderHook(() => useSidebarSections('/repos/app'))

    await vi.waitFor(() => expect(result.current.isOpen('tags', true)).toBe(false))
    expect(result.current.isOpen('stash', false)).toBe(false)
    expect(result.current.isOpen('stash', true)).toBe(true)

    act(() => result.current.toggle('stash', false))
    expect(result.current.isOpen('stash', false)).toBe(true)
    expect(SettingsService.SetSidebarSection).toHaveBeenCalledWith('/repos/app', 'stash', true)

    act(() => result.current.open('tags'))
    expect(result.current.isOpen('tags', false)).toBe(true)
    expect(SettingsService.SetSidebarSection).toHaveBeenLastCalledWith('/repos/app', 'tags', true)

    act(() => result.current.open('tags'))
    expect(SettingsService.SetSidebarSection).toHaveBeenCalledTimes(2)
  })
})
