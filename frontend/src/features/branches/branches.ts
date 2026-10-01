// Checking out "origin/fix/x" literally would detach HEAD; the short name lets
// git create or reuse a local tracking branch instead.
export function localNameFor(remoteBranch: string): string {
  return remoteBranch.slice(remoteBranch.indexOf('/') + 1)
}
