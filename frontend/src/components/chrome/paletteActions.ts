import type { WorkspaceAction } from './useWorkspaceActions'

// The palette's action groups, in the order they are listed.
export const PALETTE_SECTIONS = ['Views', 'Create', 'Release', 'Sync', 'Tools', 'Settings'] as const

export type PaletteSection = (typeof PALETTE_SECTIONS)[number]

export interface PaletteAction extends WorkspaceAction {
  section: PaletteSection
}

export function matchesAction(action: WorkspaceAction, query: string): boolean {
  return [action.label, ...(action.keywords ?? [])].some((text) => text.toLowerCase().includes(query))
}
