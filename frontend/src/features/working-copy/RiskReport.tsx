import { useState } from 'react'
import { ChevronDown, ChevronRight, TriangleAlert } from 'lucide-react'
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
  const [choice, setChoice] = useState<boolean | null>(null)
  if (!anyStaged || !findings || findings.length === 0) return null
  // A long list stays folded to its count, so the file list keeps its room.
  const open = choice ?? findings.length <= 2

  return (
    <section className="risk-report" aria-label="Staged changes report">
      <button type="button" className="risk-report-header" aria-expanded={open} onClick={() => setChoice(!open)}>
        <TriangleAlert size={12} strokeWidth={1.75} />
        <span>{plural(findings.length, 'thing')} to check before committing</span>
        {open ? <ChevronDown size={14} strokeWidth={1.5} /> : <ChevronRight size={14} strokeWidth={1.5} />}
      </button>
      <ul className="risk-report-list" hidden={!open}>
        {[...findings]
          .sort((a, b) => Number(b.rule === 'secret') - Number(a.rule === 'secret'))
          .map((finding) => (
            <li key={`${finding.rule}-${finding.path}-${finding.line}-${finding.message}`}>
              <button
                type="button"
                className={finding.rule === 'secret' ? 'risk-report-row risk-report-row-secret' : 'risk-report-row'}
                onClick={() => onOpen(finding.path, finding.line)}
              >
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
