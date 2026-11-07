import { RemoteService } from '@current-client-bindings/app'

export async function primaryRemoteName(repoPath: string): Promise<string> {
  const remotes = await RemoteService.List(repoPath)
  const name = remotes.find((r) => r.name === 'origin')?.name ?? remotes[0]?.name
  if (!name) throw new Error('No remote configured.')
  return name
}
