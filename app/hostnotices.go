//go:build !nohost

package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/hostapi"
)

// HostNotice is one thing a refresh found changed on the checked-out
// branch's change. Kind is "threads", "ci" or "state"; Path is the file of
// the first new thread, for the threads kind.
type HostNotice struct {
	Kind   string `json:"kind"`
	Text   string `json:"text"`
	Number int    `json:"number"`
	Path   string `json:"path"`
}

const cacheNotices = "notices"

// branchSnapshot is what the last refresh knew about the checked-out
// branch's change, taken before a refresh so the next answer can be
// compared with it.
type branchSnapshot struct {
	number   int
	state    hostapi.ChangeState
	ci       string
	notes    map[string]int
	hasShape bool
}

func snapshotCurrentBranch(info HostInfo, repoPath string) branchSnapshot {
	var s HostService
	current, err := s.CurrentChange(repoPath)
	if err != nil || current.Change == nil {
		return branchSnapshot{}
	}
	snap := branchSnapshot{number: current.Change.Number, state: hostapi.ChangeState(current.Change.State), ci: current.Change.CI, notes: map[string]int{}, hasShape: true}
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheThreads, strconv.Itoa(snap.number)); ok {
		for _, thread := range cached.([]hostapi.Thread) {
			snap.notes[thread.ID] = len(thread.Notes)
		}
	}
	return snap
}

// noticesSince compares the refreshed state of the same change with the
// snapshot: new or longer threads, CI turning to failed, the change leaving
// the open state. Nothing is reported for an unchanged answer, for a change
// that wasn't known before, or for other branches.
func noticesSince(before branchSnapshot, info HostInfo, repoPath string) []HostNotice {
	if !before.hasShape {
		return nil
	}
	var s HostService
	current, err := s.CurrentChange(repoPath)
	if err != nil || current.Change == nil || current.Change.Number != before.number {
		return nil
	}
	after := current.Change
	ref := changePrefix(info) + strconv.Itoa(after.Number)
	var notices []HostNotice

	newNotes, firstPath := 0, ""
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheThreads, strconv.Itoa(after.Number)); ok {
		for _, thread := range cached.([]hostapi.Thread) {
			if added := len(thread.Notes) - before.notes[thread.ID]; added > 0 {
				newNotes += added
				if firstPath == "" {
					firstPath = thread.Path
				}
			}
		}
	}
	if newNotes > 0 {
		notices = append(notices, HostNotice{Kind: "threads", Number: after.Number, Path: firstPath,
			Text: fmt.Sprintf("%d new comment%s on %s", newNotes, plural(newNotes), ref)})
	}
	if after.CI == string(hostapi.CIFailed) && before.ci != string(hostapi.CIFailed) {
		text := "Pipeline failed"
		if len(current.FailedJobs) > 0 {
			text += ": " + strings.Join(current.FailedJobs, ", ")
		}
		notices = append(notices, HostNotice{Kind: "ci", Number: after.Number, Text: text + " on " + ref})
	}
	if before.state == hostapi.ChangeOpen && hostapi.ChangeState(after.State) != hostapi.ChangeOpen {
		notices = append(notices, HostNotice{Kind: "state", Number: after.Number, Text: ref + " was " + after.State})
	}
	return notices
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// TakeNotices returns what the last refresh found changed on the
// checked-out branch, once: the next call is empty until another refresh.
func (s *HostService) TakeNotices(repoPath string) ([]HostNotice, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok {
		return []HostNotice{}, err
	}
	cached, _, found := hostClient.Cache.Get(info.Hostname, repoPath, cacheNotices, "current")
	if !found {
		return []HostNotice{}, nil
	}
	hostClient.Cache.ClearKind(info.Hostname, repoPath, cacheNotices)
	return cached.([]HostNotice), nil
}
