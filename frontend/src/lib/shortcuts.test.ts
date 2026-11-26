import { matchesShortcut } from './shortcuts'

describe('matchesShortcut', () => {
  const key = (k: string, mods: Partial<Record<'metaKey' | 'ctrlKey' | 'shiftKey' | 'altKey', boolean>> = {}) => ({
    key: k,
    metaKey: false,
    ctrlKey: false,
    shiftKey: false,
    altKey: false,
    ...mods,
  })

  it('takes Command or Ctrl for mod, and the exact modifiers', () => {
    expect(matchesShortcut(key('k', { metaKey: true }), 'mod+K')).toBe(true)
    expect(matchesShortcut(key('K', { ctrlKey: true }), 'mod+K')).toBe(true)
    expect(matchesShortcut(key('K', { metaKey: true, shiftKey: true }), 'mod+K')).toBe(false)
    expect(matchesShortcut(key('K', { metaKey: true, shiftKey: true }), 'mod+shift+K')).toBe(true)
    expect(matchesShortcut(key('k'), 'mod+K')).toBe(false)
  })

  it('reads keys with no modifier, and alt', () => {
    expect(matchesShortcut(key('F7'), 'F7')).toBe(true)
    expect(matchesShortcut(key('F7', { shiftKey: true }), 'F7')).toBe(false)
    expect(matchesShortcut(key('F7', { shiftKey: true }), 'shift+F7')).toBe(true)
    expect(matchesShortcut(key('ArrowDown', { altKey: true }), 'alt+ArrowDown')).toBe(true)
    expect(matchesShortcut(key('`', { ctrlKey: true }), 'ctrl+`')).toBe(true)
    expect(matchesShortcut(key('`', { metaKey: true }), 'ctrl+`')).toBe(false)
  })
})
