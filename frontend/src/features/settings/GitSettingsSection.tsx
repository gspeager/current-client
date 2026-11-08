import { useState } from 'react'
import { GitService } from '@current-client-bindings/app'
import PathText from '../../components/git/PathText'
import { useAsyncData } from '../../lib/useAsyncData'
import SettingsRow from './SettingsRow'

interface GitSettingsSectionProps {
  repoPath: string
  refreshKey?: unknown
}

function GitSettingsSection({ repoPath, refreshKey }: GitSettingsSectionProps) {
  const [showAll, setShowAll] = useState(false)
  const { data: report, error } = useAsyncData(() => GitService.GetGitSettings(repoPath), [repoPath], { refreshKey })

  return (
    <SettingsRow
      label="Settings in effect"
      hint="The Git settings this repository uses and where each comes from, read from Git. Nothing is changed, and credentials are hidden. Git reads them in order, so the last value of a setting applies. Signing is checked by finding its program and key, without signing anything."
      error={error && `Could not read Git's settings: ${error}`}
      stacked
    >
      {report && (
        <div className="settings-git-config">
          {report.flags.length > 0 ? (
            <ul className="settings-git-flags">
              {report.flags.map((flag) => (
                <li key={flag.key} className="settings-warning">
                  {flag.message}
                </li>
              ))}
            </ul>
          ) : (
            <p className="settings-hint">Nothing unusual.</p>
          )}
          <div className="settings-actions">
            <button type="button" className="settings-button" onClick={() => setShowAll(!showAll)}>
              {showAll ? 'Hide settings' : `Show all ${report.entries.length} settings`}
            </button>
          </div>
          {showAll && (
            <table className="settings-git-table">
              <thead>
                <tr>
                  <th scope="col">Setting</th>
                  <th scope="col">Value</th>
                  <th scope="col">Scope</th>
                  <th scope="col">From</th>
                </tr>
              </thead>
              <tbody>
                {report.entries.map((entry, index) => (
                  <tr key={index}>
                    <td title={entry.key}>{entry.key}</td>
                    <td title={entry.value}>{entry.value}</td>
                    <td>{entry.scope}</td>
                    <td>
                      <PathText path={entry.origin} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </SettingsRow>
  )
}

export default GitSettingsSection
