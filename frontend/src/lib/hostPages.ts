import type { Links } from '@current-client-bindings/core/hostlink'

// The host's pages for things in the app, built from the prefixes
// core/hostlink derived from the remote. No request is made to build one.

export function commitPage(links: Links, sha: string): string {
  return links.commit + sha
}

export function filePage(links: Links, sha: string, path: string, line?: number): string {
  const url = links.blob + sha + '/' + path.split('/').map(encodeURIComponent).join('/')
  return line ? `${url}#L${line}` : url
}

export function branchPage(links: Links, branch: string): string {
  return links.tree + branch.split('/').map(encodeURIComponent).join('/')
}

export function tagPage(links: Links, tag: string): string {
  return links.tag + encodeURIComponent(tag)
}

export function repositoryPage(links: Links): string {
  return links.project
}

// A local branch's page is its upstream's on the host; the remote name is
// not part of the branch there.
export function remoteBranchName(upstream: string, remotes: string[]): string {
  for (const remote of [...remotes].sort((a, b) => b.length - a.length)) {
    if (upstream.startsWith(remote + '/')) return upstream.slice(remote.length + 1)
  }
  return upstream
}

// "GitHub" or "GitLab" when the style is known, else the host's name.
export function hostName(links: Links): string {
  if (links.style === 'github') return 'GitHub'
  if (links.style === 'gitlab') return 'GitLab'
  try {
    return new URL(links.root).host
  } catch {
    return 'host'
  }
}
