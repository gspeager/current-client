import { IdentityService, type IdentityInfo } from '@current-client-bindings/app'
import { useAsyncData } from './useAsyncData'

// Initials and color for people known only by name, such as a host's authors.
// Local, like every badge: nothing is fetched for them.
export function useNameIdentities(names: string[]): Map<string, IdentityInfo> {
  const key = [...new Set(names.filter(Boolean))].join('\n')
  const { data } = useAsyncData(async () => {
    const unique = key ? key.split('\n') : []
    if (unique.length === 0) return new Map<string, IdentityInfo>()
    const infos = await IdentityService.GetIdentities(unique.map((name) => ({ name, email: '' })))
    return new Map(unique.flatMap((name, i) => (infos?.[i] ? [[name, infos[i]] as const] : [])))
  }, [key])
  return data ?? new Map<string, IdentityInfo>()
}
