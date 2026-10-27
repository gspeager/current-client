//go:build !nohost

package app

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

// ReviewCapabilities says which review writes the repository's host offers;
// none with the host off or a provider that takes none.
func (s *HostService) ReviewCapabilities(repoPath string) (hostapi.ReviewCapabilities, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return hostapi.ReviewCapabilities{}, err
	}
	if reviewer, ok := providerFor(info).(hostapi.Reviewer); ok {
		return reviewer.ReviewCapabilities(), nil
	}
	return hostapi.ReviewCapabilities{}, nil
}

// ReplyToThread posts the reply, then reads the change's threads again so
// the caller shows what the host has.
func (s *HostService) ReplyToThread(repoPath string, number int, threadID, body string) (ReviewThreads, error) {
	return s.reviewWrite(repoPath, number, false, func(ctx context.Context, r hostapi.Reviewer, info HostInfo, change hostapi.Change) error {
		return r.Reply(ctx, info.Project, change.Number, threadID, body)
	})
}

func (s *HostService) ResolveThread(repoPath string, number int, threadID string, resolved bool) (ReviewThreads, error) {
	return s.reviewWrite(repoPath, number, false, func(ctx context.Context, r hostapi.Reviewer, info HostInfo, change hostapi.Change) error {
		return r.ResolveThread(ctx, info.Project, change.Number, threadID, resolved)
	})
}

// CommentOnLine comments against the head the last refresh saw.
func (s *HostService) CommentOnLine(repoPath string, number int, comment hostapi.NewComment) (ReviewThreads, error) {
	return s.reviewWrite(repoPath, number, false, func(ctx context.Context, r hostapi.Reviewer, info HostInfo, change hostapi.Change) error {
		return r.CommentOnLine(ctx, info.Project, change.Number, change.HeadSHA, comment)
	})
}

// SubmitReview sends the verdict, summary and drafted comments, then reads
// the threads and the reviews again.
func (s *HostService) SubmitReview(repoPath string, number int, review hostapi.ReviewSubmission) (ReviewThreads, error) {
	return s.reviewWrite(repoPath, number, true, func(ctx context.Context, r hostapi.Reviewer, info HostInfo, change hostapi.Change) error {
		return r.SubmitReview(ctx, info.Project, change.Number, change.HeadSHA, review)
	})
}

// reviewWrite runs one write on the change's host, then refreshes what the
// write changed: the threads, and the reviews after a verdict. Each is one
// request, on the user's click.
func (s *HostService) reviewWrite(repoPath string, number int, verdict bool, write func(context.Context, hostapi.Reviewer, HostInfo, hostapi.Change) error) (ReviewThreads, error) {
	_, info, change, err := cachedChange(repoPath, number)
	if err != nil {
		return ReviewThreads{}, err
	}
	provider := providerFor(info)
	reviewer, ok := provider.(hostapi.Reviewer)
	if !ok {
		return ReviewThreads{}, errors.New("this host takes no reviews from the app")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := write(ctx, reviewer, info, change); err != nil {
		return ReviewThreads{}, err
	}
	threads, err := provider.Threads(ctx, info.Project, change.Number, change.HeadSHA)
	if err != nil {
		return ReviewThreads{}, err
	}
	hostClient.Cache.Put(info.Hostname, repoPath, cacheThreads, strconv.Itoa(change.Number), threads)
	var reviews []hostapi.Review
	if verdict {
		reviews, _ = provider.Reviews(ctx, info.Project, change)
	}
	updateOpenChange(info, repoPath, change.Number, func(c *hostapi.Change) {
		c.UnresolvedThreads = 0
		for _, thread := range threads {
			if !thread.Resolved {
				c.UnresolvedThreads++
			}
		}
		if reviews != nil {
			c.Reviews = reviews
		}
	})
	return renderThreads(info, change.Number, threads), nil
}

// updateOpenChange rewrites one change in the cached open list in place.
func updateOpenChange(info HostInfo, repoPath string, number int, update func(*hostapi.Change)) {
	cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheChanges, info.Project)
	if !ok {
		return
	}
	open := cached.(cachedChanges)
	changes := make([]hostapi.Change, len(open.changes))
	copy(changes, open.changes)
	for i := range changes {
		if changes[i].Number == number {
			update(&changes[i])
		}
	}
	hostClient.Cache.Put(info.Hostname, repoPath, cacheChanges, info.Project, cachedChanges{changes, open.truncated})
}
