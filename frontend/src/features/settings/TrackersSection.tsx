import { useState, type FormEvent } from 'react'
import { TrackerService, type TrackerInfo } from '@current-client-bindings/app'
import Checkbox from '../../components/forms/Checkbox'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import {
  blurOnEnter,
  ConnectionActions,
  IntegrationHead,
  useIntegrationAction,
  type IntegrationAction,
} from './IntegrationControls'

const KIND_LABELS: Record<string, string> = { 'jira-cloud': 'Jira Cloud', 'jira-data-center': 'Jira Data Center' }

function splitKeys(text: string): string[] {
  return text
    .split(/[\s,]+/)
    .map((k) => k.trim())
    .filter(Boolean)
}

function repoName(path: string): string {
  return path.split(/[\\/]/).filter(Boolean).pop() ?? path
}

// Keys in branch names, commit messages and titles link to the tracker's site;
// the app builds the address and contacts nothing.
// repoPath, the open repository, is ticked for a tracker as it is added.
function TrackersSection({ repoPath = null, onChanged }: { repoPath?: string | null; onChanged?: () => void }) {
  const { data: list, error, reload } = useAsyncData(() => TrackerService.List(), [], { keepData: true })
  const [adding, setAdding] = useState(false)
  const { run, actionError } = useIntegrationAction(reload, onChanged)

  return (
    <section className="settings-trackers" aria-label="Issue trackers">
      <h4 className="settings-subheading">Issue trackers</h4>
      <p className="settings-hint">
        Issue keys such as PROJ-123 link to the tracker’s site; the link is built here. The site’s API is contacted only
        while its switch is on, and only to read issues on a refresh or when one is opened.
      </p>
      {error && <p className="settings-error">{error}</p>}
      {list?.default && <TrackerRow tracker={list.default} openRepos={[]} run={run} />}
      {list?.trackers.map((t) => (
        <TrackerRow key={t.site} tracker={t} openRepos={list.openRepos} run={run} />
      ))}
      {adding ? (
        <AddTracker
          onAdd={(site, kind, keys) =>
            run(
              TrackerService.Add(site, kind, keys)
                .then((added) => (repoPath ? TrackerService.SetRepos(added.site, [repoPath]) : undefined))
                .then(() => setAdding(false)),
            )
          }
          onCancel={() => setAdding(false)}
        />
      ) : (
        <div className="settings-actions">
          <button type="button" className="settings-button" onClick={() => setAdding(true)}>
            Add issue tracker…
          </button>
        </div>
      )}
      {actionError && <p className="settings-error">{actionError}</p>}
    </section>
  )
}

function AddTracker({
  onAdd,
  onCancel,
}: {
  onAdd: (site: string, kind: string, keys: string[]) => void
  onCancel: () => void
}) {
  const [site, setSite] = useState('')
  const [kind, setKind] = useState('jira-cloud')
  const [keys, setKeys] = useState('')
  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (site.trim()) onAdd(site.trim(), kind, splitKeys(keys))
  }
  return (
    <form className="settings-host" onSubmit={submit} aria-label="Add issue tracker">
      <div className="settings-host-fields">
        <label className="settings-host-field">
          <span className="settings-hint">Kind</span>
          <select className="settings-field" value={kind} onChange={(e) => setKind(e.target.value)}>
            {Object.entries(KIND_LABELS).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <label className="settings-host-field settings-host-field-wide">
          <span className="settings-hint">Site address</span>
          <input
            className="settings-field settings-host-api"
            value={site}
            onChange={(e) => setSite(e.target.value)}
            placeholder="https://example.atlassian.net"
            spellCheck={false}
            autoFocus
          />
        </label>
      </div>
      <label className="settings-host-field">
        <span className="settings-hint">Project keys, optional; without them any ABC-123 links</span>
        <input
          className="settings-field settings-host-api"
          value={keys}
          onChange={(e) => setKeys(e.target.value)}
          spellCheck={false}
        />
      </label>
      <div className="settings-actions">
        <button type="submit" className="settings-button settings-button-small" disabled={!site.trim()}>
          Add
        </button>
        <button type="button" className="settings-button settings-button-small" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </form>
  )
}

interface TrackerRowProps {
  tracker: TrackerInfo
  openRepos: string[]
  run: IntegrationAction
}

function TrackerRow({ tracker, openRepos, run }: TrackerRowProps) {
  const dialogs = useDialogs()
  const [keys, setKeys] = useState(tracker.keys.join(', '))
  const repos = [...new Set([...openRepos, ...tracker.repos])]

  const saveKeys = () => {
    if (keys !== tracker.keys.join(', ')) void run(TrackerService.SetKeys(tracker.site, splitKeys(keys)))
  }
  const toggleRepo = (path: string, on: boolean) =>
    run(TrackerService.SetRepos(tracker.site, on ? [...tracker.repos, path] : tracker.repos.filter((r) => r !== path)))
  const remove = () =>
    dialogs
      .confirm({
        title: `Remove ${tracker.site}?`,
        message: 'Forgets this tracker. Its repositories’ issue keys stop linking.',
        confirmLabel: 'Remove',
        destructive: true,
      })
      .then((ok) => (ok ? run(TrackerService.Remove(tracker.site)) : undefined))
  const cloud = tracker.kind === 'jira-cloud'
  const setToken = async () => {
    const email = cloud
      ? await dialogs.prompt({
          title: `Account for ${tracker.hostName}`,
          label: 'The Atlassian account’s email',
          confirmLabel: 'Next',
        })
      : ''
    if (cloud && !email) return
    const token = await dialogs.prompt({
      title: `Token for ${tracker.hostName}`,
      label: cloud
        ? 'An API token for that account, from id.atlassian.com. Reading issues is all it is used for.'
        : 'A personal access token from a Jira profile. Reading issues is all it is used for.',
      confirmLabel: 'Save',
      secret: true,
    })
    if (token) await run(TrackerService.SaveToken(tracker.site, email ?? '', token))
  }

  return (
    <div className="settings-host" aria-label={tracker.site}>
      <IntegrationHead
        name={tracker.site}
        enabled={tracker.enabled}
        locked={tracker.locked}
        onToggle={(on) => void run(TrackerService.SetEnabled(tracker.site, on))}
      >
        <span className="settings-hint">
          {KIND_LABELS[tracker.kind] ?? tracker.kind}
          {tracker.managed ? ' · Set by organization policy' : ''}
        </span>
        {!tracker.managed && tracker.repos.length === 0 && (
          <span className="settings-warning">Not used by any repository</span>
        )}
      </IntegrationHead>
      <ConnectionActions
        enabled={tracker.enabled}
        hasToken={tracker.hasToken}
        noTokenHint="No token saved."
        onSetToken={() => void setToken()}
        onRemoveToken={() => void run(TrackerService.RemoveToken(tracker.site))}
        testConnection={() => TrackerService.TestConnection(tracker.site)}
        onRemove={tracker.managed ? undefined : () => void remove()}
      />
      {tracker.managed ? (
        <p className="settings-hint">
          Used by every repository without a tracker of its own
          {tracker.keys.length > 0 ? `, for ${tracker.keys.join(', ')}` : ''}.
        </p>
      ) : (
        <>
          <label className="settings-host-field">
            <span className="settings-hint">Project keys, optional; without them any ABC-123 links</span>
            <input
              className="settings-field settings-host-api"
              value={keys}
              onChange={(e) => setKeys(e.target.value)}
              onBlur={saveKeys}
              onKeyDown={blurOnEnter}
              spellCheck={false}
              aria-label={`Project keys for ${tracker.site}`}
            />
          </label>
          <fieldset className="settings-tracker-repos">
            <legend className="settings-hint">Repositories</legend>
            {repos.length === 0 && <span className="settings-hint">Open a repository to give it this tracker.</span>}
            {repos.map((path) => (
              <span key={path} className="settings-tracker-repo" title={path}>
                <Checkbox
                  checked={tracker.repos.includes(path)}
                  onChange={(on) => void toggleRepo(path, on)}
                  label={repoName(path)}
                />
              </span>
            ))}
          </fieldset>
        </>
      )}
    </div>
  )
}

export default TrackersSection
