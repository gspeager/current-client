import { PlatformService, RemoteService } from '@current-client-bindings/app'
import { ignoreDecorativeFailure } from './errors'

// Starts a merge request / pull request for branch: the in-app form when the
// host is on, else the host's pre-filled page, built without the API.
export function createChangeFor(
  repoPath: string,
  branch: string,
  onCreateChange?: (branch: string) => void,
): Promise<void> {
  if (onCreateChange) {
    onCreateChange(branch)
    return Promise.resolve()
  }
  return RemoteService.CreateChangeLink(repoPath, branch)
    .then((link) => (link ? PlatformService.OpenWebPage(link) : undefined))
    .catch(ignoreDecorativeFailure)
}
