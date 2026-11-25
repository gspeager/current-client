import type { RiskFindingInfo, RiskReportInfo } from '@current-client-bindings/app'
import PathText from '../../components/git/PathText'
import { RISK_RULES } from '../../lib/riskRules'

interface SecondLookProps {
  report: RiskReportInfo | null
  error: string | null
  listed: (path: string) => boolean
  onOpen: (finding: RiskFindingInfo) => void
}

// Facts about the change, by rule, each with why it's here.
function SecondLook({ report, error, listed, onOpen }: SecondLookProps) {
  const rules = RISK_RULES.map((rule) => ({
    ...rule,
    findings: report?.findings.filter((finding) => finding.rule === rule.id) ?? [],
    leftOut: report?.leftOut?.[rule.id] ?? 0,
  })).filter(({ findings }) => findings.length > 0)
  return (
    <section className="read-through-notes" aria-label="Worth a second look">
      <h3 className="changed-files-browser-label">Worth a second look</h3>
      {error ? (
        <p className="changed-files-browser-error">{error}</p>
      ) : !report ? (
        <p className="changed-files-browser-hint">Checking…</p>
      ) : rules.length === 0 ? (
        <p className="changed-files-browser-hint">None found.</p>
      ) : (
        rules.map(({ id, label, reason, findings, leftOut }) => (
          <div key={id} className="second-look-rule">
            <h4 className="second-look-label">
              {label} <span className="changed-files-browser-count">{findings.length + leftOut}</span>
              <span className="second-look-reason">{reason}</span>
            </h4>
            <ul className="read-through-note-list">
              {findings.map((finding) => (
                <li key={`${finding.path}-${finding.line}-${finding.oldLine}-${finding.message}`}>
                  <button
                    type="button"
                    className="second-look-finding"
                    disabled={!listed(finding.path)}
                    onClick={() => onOpen(finding)}
                  >
                    <span className="second-look-message">{finding.message}</span>
                    <PathText path={where(finding)} className="second-look-where" />
                  </button>
                </li>
              ))}
              {leftOut > 0 && <li className="second-look-left-out">{leftOut} more not listed</li>}
            </ul>
          </div>
        ))
      )}
      {report?.commitsNotRead && (
        <p className="changed-files-browser-hint">
          Only the total change was read for secrets. Commits are read one by one up to 500.
        </p>
      )}
    </section>
  )
}

function where({ path, line, oldLine }: RiskFindingInfo): string {
  return line ? `${path}:${line}` : oldLine ? `${path}:${oldLine} (old version)` : path
}

export default SecondLook
