import { useState } from 'react'
import { CurrentClientApp } from '@current-client/embed'

type View = 'host' | 'current-client'

function ViewToggle({ view, onChange }: { view: View; onChange: (view: View) => void }) {
  return (
    <div className="host-toggle" role="group" aria-label="View">
      <button type="button" aria-pressed={view === 'host'} onClick={() => onChange('host')}>
        Host view
      </button>
      <button type="button" aria-pressed={view === 'current-client'} onClick={() => onChange('current-client')}>
        Current Client
      </button>
    </div>
  )
}

// Current Client stays mounted while the host view shows, so switching back is instant
// and keeps its state.
function Host() {
  const [view, setView] = useState<View>('host')
  const [repoPath, setRepoPath] = useState<string | null>(null)
  const toggle = <ViewToggle view={view} onChange={setView} />

  return (
    <>
      {view === 'host' && (
        <main className="host-view">
          <header className="host-header">
            <h1>Host view</h1>
            {toggle}
          </header>
          <p>{repoPath ? `Current Client has ${repoPath} open.` : 'No repository open in Current Client yet.'}</p>
        </main>
      )}
      <CurrentClientApp
        active={view === 'current-client'}
        headerAccessory={toggle}
        activeRepoPath={repoPath}
        onActiveRepoChange={setRepoPath}
      />
    </>
  )
}

export default Host
