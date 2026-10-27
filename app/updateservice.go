package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/internal/update"
)

// UpdateService checks for a newer release only when Check is called from
// the Settings button: never at start-up, on a timer or in the background,
// and the result isn't kept between launches.
type UpdateService struct{}

func (s *UpdateService) ReleasesURL() string {
	return update.ReleasesURL
}

// Check contacts github.com once, with a short timeout; cancelling the
// call from the frontend stops it.
func (s *UpdateService) Check(ctx context.Context) (update.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return (&update.Checker{}).Check(ctx, appVersion)
}
