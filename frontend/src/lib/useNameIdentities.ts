import { ContributorsService, IdentityService, type IdentityInfo } from '@current-client-bindings/app'
import { useAsyncData } from './useAsyncData'

interface Author {
  name: string
  email: string
}

const normalized = (text: string) => text.trim().toLowerCase()

// A host knows someone by a username or a display name, Git by the name and
// email on their commits. Matches on the full name, the email's local part or
// a first name only one author has, so a person keeps one badge everywhere.
export function matchAuthor(name: string, authors: Author[]): Author | null {
  const wanted = normalized(name)
  const exact = authors.find((a) => normalized(a.name) === wanted || normalized(a.email.split('@')[0] ?? '') === wanted)
  if (exact) return exact
  const byFirstName = authors.filter((a) => normalized(a.name).split(/\s+/)[0] === wanted)
  return byFirstName.length === 1 ? byFirstName[0] : null
}

// Initials and color for people known only by name, such as a host's authors.
// Local, like every badge: nothing is fetched for them.
export function useNameIdentities(names: string[], repoPath = ''): Map<string, IdentityInfo> {
  const key = [...new Set(names.filter(Boolean))].join('\n')
  const { data } = useAsyncData(async () => {
    const unique = key ? key.split('\n') : []
    if (unique.length === 0) return new Map<string, IdentityInfo>()
    const authors = repoPath ? await ContributorsService.GetContributors(repoPath, '', '').catch(() => null) : null
    const infos = await IdentityService.GetIdentities(
      unique.map((name) => matchAuthor(name, authors ?? []) ?? { name, email: '' }),
    )
    return new Map(unique.flatMap((name, i) => (infos?.[i] ? [[name, infos[i]] as const] : [])))
  }, [key, repoPath])
  return data ?? new Map<string, IdentityInfo>()
}
