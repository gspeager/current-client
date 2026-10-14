import { nextSelection, type SectionPaths } from './selection'

const lists = (unstaged: string[], staged: string[] = [], conflicted: string[] = []): SectionPaths => ({
  conflicted,
  unstaged,
  staged,
})

describe('nextSelection', () => {
  it('opens the first conflicted, then unstaged, then staged file', () => {
    expect(nextSelection(lists(['b'], ['c'], ['a']), null, null)).toEqual({ path: 'a', section: 'conflicted' })
    expect(nextSelection(lists(['b'], ['c']), null, null)).toEqual({ path: 'b', section: 'unstaged' })
    expect(nextSelection(lists([], ['c']), null, null)).toEqual({ path: 'c', section: 'staged' })
  })

  it('opens nothing in a clean working tree', () => {
    expect(nextSelection(lists([]), null, null)).toBeNull()
  })

  it('keeps the open file while its list has it', () => {
    const open = { path: 'b', section: 'unstaged' as const }
    expect(nextSelection(lists(['a', 'b', 'c']), open, lists(['b', 'c']))).toBe(open)
  })

  it('opens the file that took its place when it leaves the list', () => {
    const open = { path: 'b', section: 'unstaged' as const }
    expect(nextSelection(lists(['a', 'c'], ['b']), open, lists(['a', 'b', 'c']))).toEqual({
      path: 'c',
      section: 'unstaged',
    })
    // The last one moves up to the one before it.
    const last = { path: 'c', section: 'unstaged' as const }
    expect(nextSelection(lists(['a', 'b']), last, lists(['a', 'b', 'c']))).toEqual({ path: 'b', section: 'unstaged' })
  })

  it('opens the first file when its list empties, as after a commit', () => {
    const open = { path: 'c', section: 'staged' as const }
    expect(nextSelection(lists(['a']), open, lists(['a'], ['c']))).toEqual({ path: 'a', section: 'unstaged' })
    expect(nextSelection(lists(['a']), null, lists(['a'], ['c']))).toEqual({ path: 'a', section: 'unstaged' })
  })

  it('opens a remembered file in whichever list has it on the first load', () => {
    expect(nextSelection(lists(['a'], ['b']), { path: 'b', section: 'unstaged' }, null)).toEqual({
      path: 'b',
      section: 'staged',
    })
  })
})
