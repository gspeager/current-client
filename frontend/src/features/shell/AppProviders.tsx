import type { ContextType, ReactNode } from 'react'
import type { Settings } from '@current-client-bindings/app'
import { DiffWrapContext } from '../diff/diffWrap'
import { NotesContext, type useNotesState } from '../notes/notes'
import { ReadThroughContext } from '../read-through/readThrough'
import { DensityContext } from '../../lib/density'
import { IssuePanelContext } from '../../lib/issuePanel'
import { LaneSlotsContext, LaneThemeContext, toLaneThemeName } from '../../lib/laneColor'
import { OpenCommitContext } from '../../lib/openCommit'

interface AppProvidersProps {
  settings: Settings
  onDiffWrapChange: (wrap: boolean) => void
  laneSlots: Map<string, number>
  issueActions: ContextType<typeof IssuePanelContext>
  notes: ReturnType<typeof useNotesState>['context']
  onReadThrough: ContextType<typeof ReadThroughContext>
  onOpenCommit: (sha: string) => void
  children: ReactNode
}

function AppProviders({
  settings,
  onDiffWrapChange,
  laneSlots,
  issueActions,
  notes,
  onReadThrough,
  onOpenCommit,
  children,
}: AppProvidersProps) {
  return (
    <DensityContext.Provider value={settings.density === 'compact' ? 'compact' : 'comfortable'}>
      <LaneThemeContext.Provider value={toLaneThemeName(settings.laneColorTheme)}>
        <LaneSlotsContext.Provider value={laneSlots}>
          <DiffWrapContext.Provider value={{ wrap: !settings.diffNoWrap, setWrap: onDiffWrapChange }}>
            <IssuePanelContext.Provider value={issueActions}>
              <NotesContext.Provider value={notes}>
                <ReadThroughContext.Provider value={onReadThrough}>
                  <OpenCommitContext.Provider value={onOpenCommit}>{children}</OpenCommitContext.Provider>
                </ReadThroughContext.Provider>
              </NotesContext.Provider>
            </IssuePanelContext.Provider>
          </DiffWrapContext.Provider>
        </LaneSlotsContext.Provider>
      </LaneThemeContext.Provider>
    </DensityContext.Provider>
  )
}

export default AppProviders
