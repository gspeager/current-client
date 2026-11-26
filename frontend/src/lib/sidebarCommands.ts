import { createContext, useContext, useEffect } from 'react'
import { useEventCallback } from 'usehooks-ts'

// Something elsewhere in the app, such as the palette, asking a sidebar
// section to open one of its forms or dialogs.
export type SidebarCommand =
  | 'new-branch'
  | 'clean-up'
  | 'branch-review'
  | 'conflict-grid'
  | 'tidy-up'
  | 'restack'
  | 'new-tag'
  | 'publish-release'
  | 'new-stash'
  | 'show-stash'
  | 'new-worktree'
  | 'new-remote'

// The sidebar reveals the command's section first. A collapsed sidebar draws
// no sections, so a command asked then waits for its handler to register.
export function createSidebarCommands() {
  const handlers = new Map<SidebarCommand, () => void>()
  let reveal: ((command: SidebarCommand) => void) | null = null
  let pending: SidebarCommand | null = null
  return {
    run(command: SidebarCommand) {
      reveal?.(command)
      const handler = handlers.get(command)
      pending = handler ? null : command
      handler?.()
    },
    onReveal(show: (command: SidebarCommand) => void) {
      reveal = show
      return () => {
        if (reveal === show) reveal = null
      }
    },
    register(command: SidebarCommand, handler: () => void) {
      handlers.set(command, handler)
      if (pending === command) {
        pending = null
        handler()
      }
      return () => {
        if (handlers.get(command) === handler) handlers.delete(command)
      }
    },
  }
}

export type SidebarCommands = ReturnType<typeof createSidebarCommands>

export const SidebarCommandsContext = createContext<SidebarCommands | null>(null)

export function useSidebarCommand(command: SidebarCommand, handler: () => void) {
  const commands = useContext(SidebarCommandsContext)
  const run = useEventCallback(handler)
  useEffect(() => commands?.register(command, run), [commands, command, run])
}
