import { useState } from 'react'
import PathText from '../../components/git/PathText'
import Tab from '../../components/controls/Tab'
import { joinPath } from '../../lib/paths'
import RecentRepoGrid from './RecentRepoGrid'
import type { useRepositoryLifecycle } from './useRepositoryLifecycle'
import './RepoSwitcher.scss'

type RepoTab = 'open' | 'clone' | 'new'

interface RepoSwitcherProps {
  repo: ReturnType<typeof useRepositoryLifecycle>
  // Where Clone and New put repositories, from Settings; empty means a chosen folder each time.
  defaultLocation?: string
  onOpenSettings?: () => void
}

// Open · Clone · New: the one place a repository is opened or created.
function RepoSwitcher({ repo, defaultLocation = '', onOpenSettings }: RepoSwitcherProps) {
  const [tab, setTab] = useState<RepoTab>('open')
  const [query, setQuery] = useState('')

  return (
    <div className="repo-switcher">
      <div className="repo-switcher-tabs" role="tablist">
        <Tab active={tab === 'open'} onClick={() => setTab('open')}>
          Open
        </Tab>
        <Tab active={tab === 'clone'} onClick={() => setTab('clone')}>
          Clone
        </Tab>
        <Tab active={tab === 'new'} onClick={() => setTab('new')}>
          New
        </Tab>
      </div>

      {repo.repoError && <p className="repo-switcher-error">Could not open repository: {repo.repoError}</p>}

      {tab === 'open' && (
        <div className="repo-switcher-open">
          <div className="repo-switcher-open-row">
            <input
              aria-label="Filter recent repositories"
              placeholder="Filter recent repositories…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              autoFocus
            />
            <button type="button" className="repo-switcher-primary" onClick={repo.openRepository}>
              Browse…
            </button>
          </div>
          <RecentRepoGrid repo={repo} query={query} />
        </div>
      )}

      {tab === 'clone' && (
        <div className="repo-switcher-clone">
          <input
            aria-label="repository url"
            placeholder="Repository URL"
            value={repo.cloneUrl}
            onChange={(e) => repo.setCloneUrl(e.target.value)}
            autoFocus
          />
          <div className="repo-switcher-clone-actions">
            <button type="button" className="repo-switcher-secondary" onClick={repo.chooseCloneDestination}>
              {repo.cloneDest || 'Choose destination'}
            </button>
            <input
              aria-label="folder name"
              placeholder="Folder name"
              value={repo.cloneFolder}
              onChange={(e) => repo.setCloneFolder(e.target.value)}
            />
          </div>
          {repo.cloneDest && repo.cloneFolder.trim() && (
            <p className="repo-switcher-clone-target">
              <span>Clones into</span>
              <PathText path={joinPath(repo.cloneDest, repo.cloneFolder.trim())} className="repo-switcher-clone-path" />
            </p>
          )}
          <div className="repo-switcher-clone-actions">
            <button
              type="button"
              className="repo-switcher-primary"
              onClick={repo.cloneRepository}
              disabled={!repo.cloneUrl || !repo.cloneDest || !repo.cloneFolder.trim() || repo.cloneOp.running}
            >
              {repo.cloneOp.running ? 'Cloning…' : 'Clone repository'}
            </button>
            {repo.cloneOp.running && (
              <button type="button" className="repo-switcher-secondary" onClick={repo.cloneOp.cancel}>
                Cancel
              </button>
            )}
          </div>
          {repo.cloneOp.error && <p className="repo-switcher-error">Could not clone: {repo.cloneOp.error}</p>}
        </div>
      )}

      {tab === 'new' && (
        <div className="repo-switcher-new">
          <p>Turns a folder into a Git repository, with an empty history, and opens it.</p>
          <button type="button" className="repo-switcher-primary" onClick={repo.initRepository}>
            Choose a folder and initialize…
          </button>
        </div>
      )}

      <p className="repo-switcher-footer">
        <span>{defaultLocation ? 'Dialogs start in' : 'Dialogs start in the last folder used.'}</span>
        {defaultLocation && <PathText path={defaultLocation} />}
        {onOpenSettings && (
          <button type="button" onClick={onOpenSettings}>
            Change in Settings
          </button>
        )}
      </p>
    </div>
  )
}

export default RepoSwitcher
