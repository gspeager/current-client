// Checking out "origin/fix/x" literally would detach HEAD; the short name lets
// git create or reuse a local tracking branch instead. Remote names can contain
// slashes ("team/origin"), so the longest remote the name starts with wins.
export function localNameFor(remoteBranch: string, remotes: string[]): string {
  const remote = remotes
    .filter((r) => remoteBranch.startsWith(`${r}/`))
    .reduce((longest, r) => (r.length > longest.length ? r : longest), '')
  return remote ? remoteBranch.slice(remote.length + 1) : remoteBranch.slice(remoteBranch.indexOf('/') + 1)
}
