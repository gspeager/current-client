import { useEffect } from 'react'
import { HostService } from '@current-client-bindings/app'
import { useDialogs } from '../../lib/useDialogs'
import { providerLabel } from './IntegrationsSection'

// Asks once per host, when a repository on it is opened, whether to turn its
// integration on. Finding the host reads the remote URL only; nothing is
// contacted until the answer is Enable, and "Not now" is remembered.
export function useHostPrompt(repoPath: string | null, onChanged: () => void) {
  const dialogs = useDialogs()
  useEffect(() => {
    if (!repoPath) return
    let cancelled = false
    HostService.Detect(repoPath)
      .then(async (hosts) => {
        const host = hosts.find((h) => !h.enabled && !h.dismissed && !h.managed)
        if (!host || cancelled) return
        const provider = providerLabel(host.provider)
        const enable = await dialogs.confirm({
          title: `Turn on ${provider} integration?`,
          message: `Turns on ${provider} integration for ${host.hostname}. Current Client calls its API when you open merge requests or refresh, and at no other time. Nothing is sent until then.`,
          confirmLabel: 'Enable',
          cancelLabel: 'Not now',
        })
        await (enable ? HostService.SetEnabled(host.hostname, true) : HostService.Dismiss(host.hostname))
        onChanged()
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath, dialogs])
}
