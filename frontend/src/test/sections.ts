import type { SidebarSections } from '../lib/useSidebarSections'

// Sections that follow their defaults and remember nothing.
export function defaultSections(): SidebarSections {
  return { isOpen: (_id, defaultOpen) => defaultOpen, toggle: vi.fn(), open: vi.fn() }
}
