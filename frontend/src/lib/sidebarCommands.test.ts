import { createSidebarCommands } from './sidebarCommands'

describe('createSidebarCommands', () => {
  it('reveals the section and runs the registered handler', () => {
    const commands = createSidebarCommands()
    const reveal = vi.fn()
    const handler = vi.fn()
    commands.onReveal(reveal)
    commands.register('new-tag', handler)
    commands.run('new-tag')
    expect(reveal).toHaveBeenCalledWith('new-tag')
    expect(handler).toHaveBeenCalledTimes(1)
  })

  it('keeps a command asked before its section is drawn, and runs it once', () => {
    const commands = createSidebarCommands()
    commands.run('new-stash')
    const handler = vi.fn()
    const unregister = commands.register('new-stash', handler)
    expect(handler).toHaveBeenCalledTimes(1)
    unregister()
    commands.register('new-stash', handler)
    expect(handler).toHaveBeenCalledTimes(1)
  })
})
