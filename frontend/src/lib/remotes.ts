import { RemoteService } from '@current-client-bindings/app'

// origin, or else the first remote.
export async function primaryRemoteName(repoPath: string): Promise<string> {
  const remotes = await RemoteService.List(repoPath)
  const name = remotes.find((r) => r.name === 'origin')?.name ?? remotes[0]?.name
  if (!name) throw new Error('No remote configured to push to.')
  return name
}
