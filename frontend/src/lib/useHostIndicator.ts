import { useEffect, useState } from 'react'
import { HostService, type HostInfo, type HostStatusInfo } from '@current-client-bindings/app'
import { useAsyncData } from './useAsyncData'
import { useClockTick } from './useClockTick'

const POLL_MS = 5_000

export interface HostIndicator {
  host: HostInfo
  apiHost: string
  status: HostStatusInfo | null
}

// The host the current repository has integration on for, with when it was
// last contacted. Both reads are local; hostsVersion changes when Settings
// or the first-use prompt may have changed a switch.
export function useHostIndicator(repoPath: string, hostsVersion: number, available = true): HostIndicator | null {
  const { data: hosts } = useAsyncData(() => (available ? HostService.Detect(repoPath) : null), [repoPath, available], {
    refreshKey: hostsVersion,
  })
  const host = hosts?.find((h) => h.enabled) ?? null
  const status = useHostStatus(host?.hostname ?? null)

  if (!host) return null
  return { host, apiHost: apiHostOf(host.apiUrl), status }
}

function apiHostOf(apiUrl: string): string {
  try {
    return new URL(apiUrl).host
  } catch {
    return apiUrl
  }
}

// When a host or tracker the client knows by hostname was last asked, read
// from the client's own counters every few seconds; null while hostname is.
export function useHostStatus(hostname: string | null): HostStatusInfo | null {
  const [status, setStatus] = useState<HostStatusInfo | null>(null)
  useClockTick()

  useEffect(() => {
    if (!hostname) {
      setStatus(null)
      return
    }
    let active = true
    const poll = () =>
      HostService.Status(hostname)
        .then((s) => active && setStatus(s))
        .catch(() => undefined)
    poll()
    const id = window.setInterval(poll, POLL_MS)
    return () => {
      active = false
      window.clearInterval(id)
    }
  }, [hostname])

  return status
}
