import { DashboardService } from '@current-client-bindings/app'
import { groupTopLanguages, languageColor } from './languageColors'
import { useAsyncData } from '../../lib/useAsyncData'
import './LanguageBreakdown.scss'

interface LanguageBreakdownProps {
  repoPath: string
  repoVersion: number
}

const DISPLAY_LIMIT = 5

function extensionLabel(extension: string): string {
  return extension === 'other' ? 'other' : `.${extension}`
}

function LanguageBreakdown({ repoPath, repoVersion }: LanguageBreakdownProps) {
  const { data: rawLanguages, error } = useAsyncData(
    () => DashboardService.GetLanguageBreakdown(repoPath),
    [repoPath],
    { refreshKey: repoVersion },
  )

  const languages = rawLanguages ? groupTopLanguages(rawLanguages, DISPLAY_LIMIT) : null
  const total = rawLanguages?.reduce((sum, l) => sum + l.count, 0) ?? 0
  const topTwo = languages
    ?.filter((l) => l.extension !== 'other')
    .slice(0, 2)
    .map((l) => extensionLabel(l.extension))
    .join(' + ')

  return (
    <div className="language-breakdown">
      <div className="language-breakdown-header">
        <span className="language-breakdown-title">Languages &amp; assets</span>
        {languages && (
          <span className="language-breakdown-total">
            {total} total file{total === 1 ? '' : 's'}
          </span>
        )}
      </div>

      <div className="language-breakdown-body">
        {error ? (
          <p className="language-breakdown-hint">Could not load language breakdown: {error}</p>
        ) : languages === null ? (
          <p className="language-breakdown-hint">Loading…</p>
        ) : languages.length === 0 ? (
          <p className="language-breakdown-hint">No tracked files.</p>
        ) : (
          <>
            <div className="language-breakdown-stacked-bar">
              {languages.map((l) => (
                <span
                  key={l.extension}
                  className="language-breakdown-segment"
                  style={{
                    width: `${total === 0 ? 0 : (l.count / total) * 100}%`,
                    background: languageColor(l.extension),
                  }}
                  title={`${extensionLabel(l.extension)}: ${l.count} files`}
                />
              ))}
            </div>

            <ul className="language-breakdown-list">
              {languages.map((l, i) => {
                const percent = total === 0 ? 0 : (l.count / total) * 100
                return (
                  <li key={l.extension} className="language-breakdown-row">
                    <span className="language-breakdown-dot" style={{ background: languageColor(l.extension) }} />
                    <span className="language-breakdown-extension">{extensionLabel(l.extension)}</span>
                    <span className="language-breakdown-count">
                      {l.count} file{l.count === 1 ? '' : 's'}
                    </span>
                    <span
                      className="language-breakdown-percent"
                      style={i === 0 ? { color: languageColor(l.extension) } : undefined}
                    >
                      {percent.toFixed(1)}%
                    </span>
                  </li>
                )
              })}
            </ul>
          </>
        )}
      </div>

      {topTwo && (
        <div className="language-breakdown-footer">
          <span>
            Primary: <strong>{topTwo}</strong>
          </span>
        </div>
      )}
    </div>
  )
}

export default LanguageBreakdown
