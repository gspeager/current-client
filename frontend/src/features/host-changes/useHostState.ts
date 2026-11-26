import { useState } from 'react'
import { HostService, PlatformService, TrackerService, type HostNotice } from '@current-client-bindings/app'
import { changesLabelFor } from '../../lib/changeNoun'
import { useAsyncData } from '../../lib/useAsyncData'
import { useHostIndicator } from '../../lib/useHostIndicator'

// What the open repository's host and issue tracker offer, and the counters
// that make what reads them read again.
export function useHostState(repoPath: string | null, viewPath: string | null) {
  // false only in a build made with the nohost tag; until known, assume the full build.
  const { data: built } = useAsyncData(() => PlatformService.HostIntegrationAvailable(), [])
  const hostIntegration = built !== false
  // Bumped when a host switch may have changed, so the status bar's indicator follows.
  const [hostsVersion, setHostsVersion] = useState(0)
  const bumpHostsVersion = () => setHostsVersion((n) => n + 1)
  // Bumped after a refresh, so lists from the host's cache read it again.
  const [hostRefresh, setHostRefresh] = useState(0)
  const bumpHostRefresh = () => setHostRefresh((n) => n + 1)

  const [notices, setNotices] = useState<HostNotice[]>([])
  // A refresh may have found something changed on the checked-out branch's change.
  const collectNotices = () => {
    if (!repoPath || !hostIntegration) return
    HostService.TakeNotices(repoPath)
      .then((taken) => taken.length > 0 && setNotices(taken))
      .catch(() => undefined)
  }

  const integration = useHostIndicator(viewPath ?? '', hostsVersion, hostIntegration && !!viewPath)
  const hostOn = !!integration
  const changesLabel = integration ? changesLabelFor(integration.host.provider) : null
  const { data: canWriteIssues } = useAsyncData(
    () => (hostOn && viewPath ? HostService.CanWriteIssues(viewPath) : null),
    [hostOn, viewPath],
  )
  const { data: tracker } = useAsyncData(() => (viewPath ? TrackerService.Links(viewPath) : null), [viewPath], {
    refreshKey: hostsVersion,
  })
  // The Issues tab shows while the host or the tracker lists issues; its
  // count is the cache's, read again after each refresh.
  const issuesOn = hostIntegration && (hostOn || !!tracker?.reading)
  const { data: myIssues } = useAsyncData(
    () => (issuesOn && viewPath ? HostService.MyIssues(viewPath) : null),
    [issuesOn, viewPath],
    { refreshKey: hostRefresh },
  )
  // A host the remote points at with integration off, and not locked off,
  // shows its tabs dimmed; finding it reads the remote, nothing is asked.
  const { data: detectedHosts } = useAsyncData(
    () => (hostIntegration && viewPath && !hostOn ? HostService.Detect(viewPath) : null),
    [hostIntegration, viewPath, hostOn],
    { refreshKey: hostsVersion },
  )
  const offHostInfo = integration ? undefined : detectedHosts?.find((h) => !h.enabled && !h.managed)
  const offHost = offHostInfo
    ? { hostname: offHostInfo.hostname, changesLabel: changesLabelFor(offHostInfo.provider) }
    : null

  return {
    hostIntegration,
    hostsVersion,
    bumpHostsVersion,
    hostRefresh,
    bumpHostRefresh,
    notices,
    collectNotices,
    dismissNotices: () => setNotices([]),
    integration,
    hostOn,
    changesLabel,
    canWriteIssues: canWriteIssues ?? false,
    tracker: tracker ?? null,
    issuesOn,
    issueCount: issuesOn ? (myIssues?.issues?.length ?? 0) : null,
    offHost,
  }
}
