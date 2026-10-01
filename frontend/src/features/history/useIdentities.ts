import { useState } from 'react'
import { IdentityService, type CommitInfo, type IdentityInfo } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { identityKey } from './identityKey'
import { useAsyncData } from '../../lib/useAsyncData'

function loadIdentities(commits: CommitInfo[]): Promise<Map<string, IdentityInfo>> | null {
  if (commits.length === 0) return null
  const uniqueByKey = new Map<string, { name: string; email: string }>()
  for (const c of commits) {
    uniqueByKey.set(identityKey(c.authorName, c.authorEmail), { name: c.authorName, email: c.authorEmail })
  }
  const keys = [...uniqueByKey.keys()]
  return IdentityService.GetIdentities([...uniqueByKey.values()]).then(
    (infos) => new Map(keys.map((key, i) => [key, infos[i]])),
  )
}

export function useIdentities(commits: CommitInfo[]) {
  const { data, reload } = useAsyncData(() => loadIdentities(commits), [commits], { keepData: true })
  const [colorError, setColorError] = useState<string | null>(null)

  const setAuthorColor = (name: string, email: string, color: string) => {
    setColorError(null)
    IdentityService.SetIdentityColor(name, email, color)
      .then(reload)
      .catch((err: unknown) => setColorError(errorMessage(err)))
  }

  return { identities: data ?? new Map<string, IdentityInfo>(), colorError, setAuthorColor }
}
