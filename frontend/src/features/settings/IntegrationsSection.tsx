import { useState } from 'react'
import { HostService, type HostInfo } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import {
  blurOnEnter,
  ConnectionActions,
  IntegrationHead,
  useIntegrationAction,
  type IntegrationAction,
} from './IntegrationControls'

const PROVIDER_LABELS: Record<string, string> = { github: 'GitHub', gitlab: 'GitLab' }

export function providerLabel(provider: string): string {
  return PROVIDER_LABELS[provider] ?? provider
}

interface IntegrationsSectionProps {
  repoPath: string | null
  onChanged?: () => void
}

function IntegrationsSection({ repoPath, onChanged }: IntegrationsSectionProps) {
  const dialogs = useDialogs()
  const {
    data: hosts,
    error,
    reload,
  } = useAsyncData(() => HostService.List(repoPath ? [repoPath] : []), [repoPath], {
    keepData: true,
  })
  const { run, actionError } = useIntegrationAction(reload, onChanged)

  const addHost = () =>
    dialogs
      .prompt({ title: 'Add host', label: 'Hostname, such as gitlab.example.com', confirmLabel: 'Add' })
      .then((hostname) => (hostname ? run(HostService.Add(hostname, '')) : undefined))

  return (
    <div className="settings-integrations">
      <h4 className="settings-subheading">Hosts</h4>
      <p className="settings-hint">
        Off by default. A host’s API is contacted only while its switch is on, and only when merge requests are opened
        or refreshed. Links to the host work either way.
      </p>
      {error && <p className="settings-error">{error}</p>}
      {hosts?.map((host) => (
        <HostRow key={host.hostname} host={host} run={run} />
      ))}
      {hosts?.length === 0 && (
        <p className="settings-hint">No hosts yet. Open a repository with a remote, or add one.</p>
      )}
      <div className="settings-actions">
        <button type="button" className="settings-button" onClick={addHost}>
          Add host…
        </button>
      </div>
      {actionError && <p className="settings-error">{actionError}</p>}
    </div>
  )
}

interface HostRowProps {
  host: HostInfo
  run: IntegrationAction
}

function HostRow({ host, run }: HostRowProps) {
  const dialogs = useDialogs()
  const [apiUrl, setApiUrl] = useState(host.apiUrl)

  const saveApiUrl = () => {
    const value = apiUrl.trim()
    if (value === host.apiUrl) return
    void run(HostService.SetHost(host.hostname, host.provider, value === host.defaultApiUrl ? '' : value))
  }

  const setProvider = (provider: string) =>
    run(HostService.SetHost(host.hostname, provider, apiUrl.trim() === host.defaultApiUrl ? '' : apiUrl.trim()))

  const setToken = () =>
    dialogs
      .prompt({
        title: `API token for ${host.hostname}`,
        label:
          'Personal access token. Read-only scope is enough: GitLab read_api, or GitHub read access to pull requests, checks, contents and issues.',
        confirmLabel: 'Save',
        secret: true,
      })
      .then((token) => (token ? run(HostService.SaveToken(host.hostname, token)) : undefined))

  const remove = () =>
    dialogs
      .confirm({
        title: `Remove ${host.hostname}?`,
        message: host.hasToken
          ? 'Forgets the settings for this host and removes its token from Git’s credential helper.'
          : 'Forgets the settings for this host.',
        confirmLabel: 'Remove',
        destructive: true,
      })
      .then((ok) => (ok ? run(HostService.Remove(host.hostname)) : undefined))

  return (
    <div className="settings-host" aria-label={host.hostname}>
      <IntegrationHead
        name={host.hostname}
        enabled={host.enabled}
        locked={host.managed}
        onToggle={(on) => void run(HostService.SetEnabled(host.hostname, on))}
      >
        <span className="settings-hint">
          {host.remote ? (
            <>
              {host.remote} · <span className="settings-host-project">{host.project}</span>
            </>
          ) : (
            'Added by hand'
          )}
        </span>
      </IntegrationHead>

      <div className="settings-host-fields">
        <label className="settings-host-field">
          <span className="settings-hint">Provider</span>
          <select
            className="settings-field"
            value={host.provider}
            onChange={(e) => void setProvider(e.target.value)}
            aria-label={`Provider for ${host.hostname}`}
          >
            <option value="github">GitHub</option>
            <option value="gitlab">GitLab</option>
          </select>
        </label>
        <label className="settings-host-field settings-host-field-wide">
          <span className="settings-hint">API address</span>
          <input
            className="settings-field settings-host-api"
            value={apiUrl}
            onChange={(e) => setApiUrl(e.target.value)}
            onBlur={saveApiUrl}
            onKeyDown={blurOnEnter}
            spellCheck={false}
            aria-label={`API address for ${host.hostname}`}
          />
        </label>
      </div>

      {host.sslVerifyOff && (
        <p className="settings-error">
          Git has http.sslVerify off for this address. Certificate checks stay on here; configure http.sslCAInfo for the
          host’s certificate authority instead.
        </p>
      )}

      <ConnectionActions
        enabled={host.enabled}
        hasToken={host.hasToken}
        noTokenHint="Uses Git’s own credential for the host."
        onSetToken={() => void setToken()}
        onRemoveToken={() => void run(HostService.RemoveToken(host.hostname))}
        testConnection={() => HostService.TestConnection(host.hostname)}
        onRemove={() => void remove()}
      />
    </div>
  )
}

export default IntegrationsSection
