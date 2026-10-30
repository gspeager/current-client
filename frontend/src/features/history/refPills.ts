import type { RefInfo } from '@current-client-bindings/app'

export interface RefPill {
  ref: RefInfo
  upstream?: RefInfo
}

// A local branch and its upstream at the same commit share one pill; the
// remote ref keeps its own only when they differ.
export function pairUpstreams(refs: RefInfo[], upstreams: Record<string, string> = {}): RefPill[] {
  const remotes = new Map(refs.filter((r) => r.kind === 'remote').map((r) => [r.name, r]))
  const paired = new Set<string>()
  const pills: RefPill[] = []
  for (const ref of refs) {
    if (ref.kind === 'branch') {
      const upstream = remotes.get(upstreams[ref.name] ?? '')
      if (upstream) paired.add(upstream.name)
      pills.push({ ref, upstream })
    }
  }
  for (const ref of refs) {
    if (ref.kind !== 'branch' && !paired.has(ref.name)) pills.push({ ref })
  }
  return pills
}
