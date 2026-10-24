import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { SettingsService } from '@current-client-bindings/app'

export interface SidebarSections {
  // The stored choice for the section, else `defaultOpen`.
  isOpen: (id: string, defaultOpen: boolean) => boolean
  toggle: (id: string, defaultOpen: boolean) => void
  open: (id: string) => void
}

// Which sidebar sections are open, remembered per repository. A section the
// user has never toggled follows the default its content decides.
export function useSidebarSections(repoPath: string): SidebarSections {
  const [choices, setChoices] = useState<Record<string, boolean | undefined>>({})
  const loadedPath = useRef(repoPath)

  useEffect(() => {
    loadedPath.current = repoPath
    setChoices({})
    SettingsService.GetSidebarSections(repoPath)
      .then((stored) => {
        if (loadedPath.current === repoPath) setChoices(stored ?? {})
      })
      .catch(() => {})
  }, [repoPath])

  const set = useCallback(
    (id: string, open: boolean) => {
      setChoices((prev) => ({ ...prev, [id]: open }))
      SettingsService.SetSidebarSection(repoPath, id, open).catch(() => {})
    },
    [repoPath],
  )

  return useMemo(() => {
    const isOpen = (id: string, defaultOpen: boolean) => choices[id] ?? defaultOpen
    return {
      isOpen,
      toggle: (id, defaultOpen) => set(id, !isOpen(id, defaultOpen)),
      open: (id) => {
        if (!isOpen(id, false)) set(id, true)
      },
    }
  }, [choices, set])
}
