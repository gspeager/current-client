//go:build !nohost

package app

import (
	"strings"

	"github.com/gspeager/current-client/core/hostapi"
)

// releaseHostChecks reads the cache only: the candidate's pipeline and the
// target's protection, as the last refresh left them.
func releaseHostChecks(repoPath, sha, target string) []ReleaseCheck {
	ci := ReleaseCheck{Label: "CI on the candidate"}
	protection := ReleaseCheck{Label: "Protection on " + target}
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		ci.Level, ci.Value = "unknown", "Host integration is off"
		protection.Level, protection.Value = "unknown", "Host integration is off"
		return []ReleaseCheck{ci, protection}
	}
	if cached, at, ok := hostClient.Cache.Get(info.Hostname, repoPath, cachePipeline, sha); ok {
		mark := cached.(CIMark)
		ci.AsOf = &at
		switch hostapi.CIStatus(mark.Status) {
		case hostapi.CISuccess:
			ci.Level, ci.Value = "ok", "Passed"
		case hostapi.CIFailed, hostapi.CICanceled:
			ci.Level, ci.Value = "warn", strings.ToUpper(mark.Status[:1])+mark.Status[1:]
		case "":
			ci.Level, ci.Value = "warn", "No pipeline for this commit"
		default:
			ci.Level, ci.Value = "info", strings.ToUpper(mark.Status[:1])+mark.Status[1:]
		}
	} else {
		ci.Level, ci.Value = "unknown", "Not fetched; refresh, or show the commit in History"
	}
	cached, at, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheRules, "all")
	switch {
	case !ok:
		protection.Level, protection.Value = "unknown", "Not fetched; refresh"
	default:
		protection.AsOf = &at
		rule, matched := hostapi.MatchRule(cached.([]hostapi.BranchRule), target)
		if !matched {
			protection.Level, protection.Value = "warn", "The last refresh found no rule for "+target
			break
		}
		protection.Level, protection.Value = "ok", "Protected"
		if rule.Pattern != target {
			protection.Evidence = append(protection.Evidence, "by the rule "+rule.Pattern)
		}
		for _, r := range []struct {
			on   bool
			what string
		}{{rule.NoPush, "no direct push"}, {rule.RestrictedPush, "push restricted"}, {rule.NoForcePush, "no force push"}, {rule.NoDelete, "no delete"}} {
			if r.on {
				protection.Evidence = append(protection.Evidence, r.what)
			}
		}
	}
	return []ReleaseCheck{ci, protection}
}
