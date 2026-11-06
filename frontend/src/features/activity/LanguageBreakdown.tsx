import { DashboardService } from '@current-client-bindings/app'
import { groupTopLanguages } from './languageColors'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import './LanguageBreakdown.scss'

interface LanguageBreakdownProps {
  repoPath: string
  repoVersion: number
}

// One hue in steps by rank: a chart of its own, with no tie to the branch colors.
function typeColor(extension: string, rank: number): string {
  if (extension === 'other') return 'var(--surface-highest)'
  return `color-mix(in srgb, var(--on-surface-variant) ${Math.max(32, 100 - rank * 17)}%, var(--surface-low))`
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

  return (
    <div className="language-breakdown">
      <div className="language-breakdown-header">
        <span className="language-breakdown-title">File types</span>
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
                    background: typeColor(l.extension, languages.indexOf(l)),
                  }}
                  title={`${extensionLabel(l.extension)}: ${l.count} files`}
                />
              ))}
            </div>

            <ul className="language-breakdown-list">
              {languages.map((l) => {
                const percent = total === 0 ? 0 : (l.count / total) * 100
                return (
                  <li key={l.extension} className="language-breakdown-row">
                    <span
                      className="language-breakdown-dot"
                      style={{ background: typeColor(l.extension, languages.indexOf(l)) }}
                    />
                    <span className="language-breakdown-extension">{extensionLabel(l.extension)}</span>
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
