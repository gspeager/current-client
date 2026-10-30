import { createLucideIcon, ExternalLink, type LucideIcon } from 'lucide-react'
import { siGithub, siGitlab, siJira } from 'simple-icons'
import type { Links } from '@current-client-bindings/core/hostlink'
import { useHostLinks } from '../../lib/useHostLinks'

// Simple Icons fill the whole 24px box, where Lucide's line icons keep 2px
// clear, so each logo is drawn into the same 20px to sit at the same size.
const logo = (name: string, d: string) =>
  createLucideIcon(name, [
    ['path', { d, fill: 'currentColor', stroke: 'none', transform: 'translate(2 2) scale(0.8333)', key: name }],
  ])

const LOGOS: Record<string, LucideIcon> = {
  github: logo('github', siGithub.path),
  gitlab: logo('gitlab', siGitlab.path),
  jira: logo('jira', siJira.path),
}

// The logo of the host a link opens, in the text's color; a host without
// one keeps the external-link icon.
export const hostLogo = (provider: string | undefined): LucideIcon => (provider && LOGOS[provider]) || ExternalLink

// Numbered issues are the repository host's; keys like PROJ-12 are Jira's.
export const issueProvider = (key: string, links: Links | null): string | undefined =>
  /^#?\d+$/.test(key) ? links?.style : 'jira'

export const useHostLogo = (repoPath: string): LucideIcon => hostLogo(useHostLinks(repoPath)?.style)
