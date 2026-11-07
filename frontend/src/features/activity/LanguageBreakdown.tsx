import { DashboardService } from '@current-client-bindings/app'
import { groupByLanguage, groupTopLanguages, languageColor, languageName } from './languageColors'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import './LanguageBreakdown.scss'

interface LanguageBreakdownProps {
  repoPath: string
  repoVersion: number
}

const DISPLAY_LIMIT = 5

function LanguageBreakdown({ repoPath, repoVersion }: LanguageBreakdownProps) {
  const { data: rawLanguages, error } = useAsyncData(
    () => DashboardService.GetLanguageBreakdown(repoPath),
    [repoPath],
    { refreshKey: repoVersion },
  )

  const languages = rawLanguages ? groupTopLanguages(groupByLanguage(rawLanguages), DISPLAY_LIMIT) : null
  const total = rawLanguages?.reduce((sum, l) => sum + l.count, 0) ?? 0

  return (
    <div className="language-breakdown">
      <div className="language-breakdown-header">
        <h3 className="language-breakdown-title">Languages</h3>
        {languages && <span className="language-breakdown-total">{plural(total, 'total file')}</span>}
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
                  title={`${languageName(l.extension)}: ${l.count} files`}
                />
              ))}
            </div>

            <ul className="language-breakdown-list">
              {languages.map((l) => {
                const percent = total === 0 ? 0 : (l.count / total) * 100
                return (
                  <li key={l.extension} className="language-breakdown-row">
                    <span className="language-breakdown-dot" style={{ background: languageColor(l.extension) }} />
                    <span className="language-breakdown-extension">{languageName(l.extension)}</span>
                    <span className="language-breakdown-count">{plural(l.count, 'file')}</span>
                    <span className="language-breakdown-percent">{percent.toFixed(1)}%</span>
                  </li>
                )
              })}
            </ul>
          </>
        )}
      </div>
    </div>
  )
}

export default LanguageBreakdown
