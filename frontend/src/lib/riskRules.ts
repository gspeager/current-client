export interface RiskRule {
  id: string
  label: string
  // Why it's worth a second look.
  reason: string
}

// In the order they're listed.
export const RISK_RULES: RiskRule[] = [
  { id: 'secret', label: 'Secret-looking strings', reason: 'Anyone who can read the repository can read it.' },
  { id: 'conflict-marker', label: 'Conflict markers', reason: 'Left over from a merge.' },
  { id: 'large-file', label: 'Files over 10 MB', reason: 'Every clone downloads it.' },
  {
    id: 'lockfile',
    label: 'Lockfile out of step with its manifest',
    reason: 'What installs may not be what is declared.',
  },
  { id: 'generated', label: 'Sweeping changes to generated files', reason: 'Usually rebuilt rather than edited.' },
  { id: 'fewer-tests', label: 'Fewer tests', reason: 'What they covered may no longer be checked.' },
  { id: 'skipped-tests', label: 'Tests skipped', reason: 'They no longer run.' },
  { id: 'silenced-checker', label: 'Checkers silenced', reason: 'What it would report there is hidden.' },
  { id: 'ci', label: 'CI files', reason: 'They decide what runs on every push, and with which secrets.' },
  { id: 'agent-instructions', label: 'Agent instructions', reason: 'Coding agents follow them.' },
  { id: 'migration', label: 'Database migrations', reason: 'They change stored data.' },
  { id: 'exports', label: 'What a package exports', reason: 'Code that uses the package may need to change.' },
]

export function riskReason(rule: string): string | undefined {
  return RISK_RULES.find(({ id }) => id === rule)?.reason
}
