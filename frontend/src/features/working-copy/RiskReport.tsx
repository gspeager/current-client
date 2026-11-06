import { TriangleAlert } from 'lucide-react'
import { CommitService, type FileStatus } from '@current-client-bindings/app'
import PathText from '../../components/git/PathText'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import './RiskReport.scss'

interface RiskReportProps {
  repoPath: string
  staged: FileStatus[]
  rulesOff: string[]
  onOpen: (path: string, line: number) => void
}

// Its own component with its own request, so reading a large staged change
// never holds up typing in the composer beside it.
function RiskReport({ repoPath, staged, rulesOff, onOpen }: RiskReportProps) {
  const stagedKey = JSON.stringify(staged.map((f) => [f.path, f.indexAdded, f.indexRemoved]))
  const anyStaged = staged.length > 0
  const { data: findings } = useAsyncData(
    () => (anyStaged ? CommitService.GetRiskReport(repoPath) : null),
    [repoPath, anyStaged],
    { refreshKey: `${stagedKey}${rulesOff.join()}`, keepData: true },
  )
  if (!anyStaged || !findings || findings.length === 0) return null

  return (
    <section className="risk-report" aria-label="Staged changes report">
      <header className="risk-report-header">
        <TriangleAlert size={12} strokeWidth={1.75} />
        <span>{plural(findings.length, 'thing')} to check before committing</span>
      </header>
      <ul className="risk-report-list">
        {findings.map((finding) => (
          <li key={`${finding.rule}-${finding.path}-${finding.line}-${finding.message}`}>
            <button type="button" className="risk-report-row" onClick={() => onOpen(finding.path, finding.line)}>
              <span className="risk-report-message">{finding.message}</span>
              <span className="risk-report-where">
                <PathText path={finding.path} />
                {finding.line > 0 && <span>:{finding.line}</span>}
              </span>
            </button>
          </li>
        ))}
      </ul>
    </section>
  )
}

export default RiskReport
